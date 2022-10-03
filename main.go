package main

import (
	"Paktum/ImageScraper"
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/meilisearch/meilisearch-go"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	var mode string
	flag.StringVar(&mode, "mode", "", "The mode to run in. Either 'scrape', 'process', or 'server'")

	// redis is shared by server and scrape mode and used as a transfer layer
	var redisHostname string
	flag.StringVar(&redisHostname, "redis", "localhost:6379", "The redis server to connect to")
	var redisPass string
	flag.StringVar(&redisPass, "redisPass", "", "The password for the redis server")

	// meili is shared by server and process mode and used as search index
	var meiliHostname string
	flag.StringVar(&meiliHostname, "meilihost", "http://localhost:7700", "The meilisearch server to connect to")
	var meiliKey string
	flag.StringVar(&meiliKey, "meilikey", "", "The meilisearch master-key to use")

	// process mode is used to process the images
	var imageDir string
	flag.StringVar(&imageDir, "imageDir", "./images/", "The directory to store images in")

	// server mode
	var port int
	flag.IntVar(&port, "port", 9000, "The port to run the server on")

	// scrape mode
	var sqlite string
	flag.StringVar(&sqlite, "sqlite", "paktum.db", "The sqlite database to connect to")

	flag.Parse()

	if mode != "scrape" && mode != "server" && mode != "process" {
		fmt.Println("Please choose either scraping or server mode")
		flag.Usage()
		os.Exit(1)
		return
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisHostname,
		Password: redisPass, // no password set
		DB:       0,         // use default DB
	})
	meiliClient := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   meiliHostname,
		APIKey: meiliKey,
	})

	if mode == "scrape" {
		println("Scraping mode launching")

		// read from stdin until EOF
		// for each line, add the space-seperated tags into an array
		// call scrape with the array

		tags := readStdinTagsIntoArray()

		progress := make(chan int, len(tags))
		pbar := progressbar.Default(int64(len(tags)), "Fetch tag metadata...")

		for _, tag := range tags {
			go func(tag []string) {
				err, images := ImageScraper.Scrape(tag, uuid.New().String())
				if err != nil {
					fmt.Println(err)
					progress <- 1
					return
				}

				//encode image array into gob and send to redis
				var buf bytes.Buffer
				enc := gob.NewEncoder(&buf)
				err = enc.Encode(images)
				if err != nil {
					fmt.Println(err)
					progress <- 1
					return
				}

				println("Sending", len(images), "images to redis")
				_, err = redisClient.RPush(context.Background(), "paktum:metadata_process", buf.Bytes()).Result()
				if err != nil {
					fmt.Println("Failed to push data to redis:", err)
					progress <- 1
					return
				}

				progress <- 1
			}(tag)
		}

		// wait for all goroutines to finish
		for i := 0; i < len(tags); i++ {
			<-progress
			_ = pbar.Add(1)
		}

	} else if mode == "process" {
		println("Process mode launching")

		// read data from redis
		// decode gob
		// process images, check for duplicates
		// send to sqlite & meili

		task, err := meiliClient.CreateIndex(&meilisearch.IndexConfig{
			Uid:        "images",
			PrimaryKey: "ID",
		})
		if err != nil {
			fmt.Println("Failed to create MeiliSearch index:", err)
			os.Exit(1)
			return
		}
		if !waitForMeilisearchTask(task, meiliClient) {
			os.Exit(1)
			return
		}

		imageCollection := meiliClient.Index("images")
		task, err = imageCollection.UpdateFilterableAttributes(&[]string{"ID", "Tagstring"})
		if err != nil {
			fmt.Println("Failed to update filterable attributes:", err)
			return
		}
		if !waitForMeilisearchTask(task, meiliClient) {
			os.Exit(1)
			return
		}

		for {
			result, err := redisClient.BLPop(context.TODO(), time.Second*10, "paktum:metadata_process").Result()
			if err != nil {
				if err == redis.Nil {
					//println("No data in redis in 180s, re-fetching")
				} else {
					println("Error reading from redis:", err.Error())
				}
				continue
			}

			println("Got", len(result), "items from redis")

			var images []ImageScraper.Image
			dec := gob.NewDecoder(bytes.NewBuffer([]byte(result[1])))
			err = dec.Decode(&images)
			if err != nil {
				println("Failed to decode image gob:", err.Error())
				continue
			}
			println("Decoded", len(images), "images")

			var wg sync.WaitGroup
			pbar := progressbar.Default(int64(len(images)), "Downloading...")

			for _, image := range images {
				wg.Add(1)
				go func(image ImageScraper.Image, wg *sync.WaitGroup, imageCollection *meilisearch.Index, pbar *progressbar.ProgressBar) {
					// check if image already exists
					// if it does, skip
					// if it doesn't, download and add to meili
					defer wg.Done()
					defer func(pbar *progressbar.ProgressBar, num int) {
						_ = pbar.Add(num)
					}(pbar, 1)

					md5 := strings.TrimSuffix(image.Filename, filepath.Ext(image.Filename))

					if imageExists(imageCollection, md5) {
						println("Image", md5, "already exists, skipping")
						return
					}

					err := downloadImage(image.FileURL, imageDir, image.Filename)
					if err != nil {
						println("Failed to download image", image.Filename)
						return
					}

					type ImageEntry struct {
						ID        string   `json:"ID"`
						URL       string   `json:"URL"`
						Tags      []string `json:"Tags"`
						Tagstring string   `json:"Tagstring"`
					}

					// add to meili
					_, err = imageCollection.AddDocuments([]ImageEntry{{
						ID:        md5,
						URL:       image.FileURL,
						Tags:      image.Tags,
						Tagstring: strings.Join(image.Tags, " "),
					}})
				}(image, &wg, imageCollection, pbar)
			}

			wg.Wait()
			_ = pbar.Finish()
		}
	}
}

func imageExists(meiliIndex *meilisearch.Index, md5 string) bool {
	search, err := meiliIndex.Search("", &meilisearch.SearchRequest{
		Filter: []string{fmt.Sprintf(`ID = %s`, md5)},
	})
	if err != nil {
		println("Failed to search meili:", err.Error())
		return false
	}
	if len(search.Hits) != 0 {
		println("Image already exists, skipping")
		return true
	}
	return false
}

func downloadImage(url string, imageDir string, filename string) error {
	temporaryImageFile, err := os.CreateTemp(imageDir, "temp-paktum-")
	if err != nil {
		println("Failed to create file:", err.Error())
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		println("Failed to download image:", err.Error())
		return err
	}

	_, err = io.Copy(temporaryImageFile, resp.Body)
	if err != nil {
		println("Failed to write data into image:", err.Error())
		return err
	}
	err = temporaryImageFile.Close()
	if err != nil {
		return err
	}

	// rename temp image file to proper name
	err = os.Rename(temporaryImageFile.Name(), imageDir+"/"+filename)
	if err != nil {
		println("Failed to move image:", err.Error())
		return err
	}

	return nil
}

func waitForMeilisearchTask(info *meilisearch.TaskInfo, client *meilisearch.Client) bool {
	for {
		task, err := client.GetTask(info.TaskUID)
		if err != nil {
			fmt.Println("Failed to get task:", err)
			return false
		}
		if task.Status == "failed" {
			if task.Error.Code == "index_already_exists" {
				return true
			}
			println("MeiliSearch task failed:", task.Error.Message, "-", task.Error.Code)
			return false
		}
		if task.Status == "succeeded" {
			return true
		}
		time.Sleep(time.Millisecond * 500)
	}
}

func readStdinTagsIntoArray() [][]string {
	reader := bufio.NewReader(os.Stdin)
	var tags [][]string
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		tags = append(tags, strings.Split(strings.TrimSpace(text), " "))
	}
	return tags
}
