package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"time"
)

func init() {
	lvl, ok := os.LookupEnv("LOG_LEVEL")
	// LOG_LEVEL not set, let's default to debug
	if !ok {
		lvl = "debug"
	}
	// parse string, this is built-in feature of logrus
	ll, err := log.ParseLevel(lvl)
	if err != nil {
		ll = log.DebugLevel
	}
	// set global log level
	log.SetLevel(ll)
}

func main() {
	var mode string
	flag.StringVar(&mode, "mode", "", "The mode to run in. Either 'scrape', 'process', 'cleanup' or 'server'")

	var serverBaseURL string
	flag.StringVar(&serverBaseURL, "base-url", "http://localhost:8080", "The base URL of the server. No trailing slash.")

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

	flag.Parse()

	if mode != "scrape" && mode != "server" && mode != "process" && mode != "cleanup" {
		log.Error("Please choose either scraping or server mode")
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
		ScrapeMode(redisClient)
	} else if mode == "process" {
		ProcessMode(redisClient, meiliClient, imageDir)
	} else if mode == "cleanup" {
		CleanupMode(meiliClient, redisClient)
	} else if mode == "server" {
		ServerMode(meiliClient, redisClient, imageDir, serverBaseURL)
	}
}

func imageExists(meiliIndex *meilisearch.Index, md5 string) bool {
	filter := []string{fmt.Sprintf(`ID = %s`, md5)}

	log.Trace("Constructed meilisearch filter:", filter)
	search, err := meiliIndex.Search("", &meilisearch.SearchRequest{
		Filter: filter,
	})
	if err != nil {
		log.Error("Failed to search meili:", err.Error())
		return false
	}
	if len(search.Hits) != 0 {
		log.Info("Image already exists, skipping")
		return true
	}
	return false
}

// download image
// returns the pHash as uint64
// and the size in bytes as int
// and the image dimensions, width and height as int
func downloadImage(url string, imageDir string, filename string) (error, uint64, int, int, int) {
	temporaryImageFile, err := os.CreateTemp(imageDir, "temp-paktum-")
	if err != nil {
		log.Error("Failed to create file:", err.Error())
		return err, 0, 0, 0, 0
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Error("Failed to download image:", err.Error())
		return err, 0, 0, 0, 0
	}

	buf := bytes.Buffer{}
	tee := io.TeeReader(resp.Body, &buf)

	size := int(resp.ContentLength)

	_, err = io.Copy(temporaryImageFile, tee)
	if err != nil {
		log.Error("Failed to write data into image:", err.Error())
		return err, 0, 0, 0, 0
	}
	err = temporaryImageFile.Close()
	if err != nil {
		return err, 0, 0, 0, 0
	}

	// rename temp image file to proper name
	err = os.Rename(temporaryImageFile.Name(), imageDir+"/"+filename)
	if err != nil {
		log.Error("Failed to move image:", err.Error())
		return err, 0, 0, 0, 0
	}

	// calculate pHash
	decodedImage := DecodeImage(io.NopCloser(&buf))
	if decodedImage == nil {
		return nil, 0, size, 0, 0
	}

	return nil, GeneratePHash(decodedImage), size, decodedImage.Bounds().Dx(), decodedImage.Bounds().Dy()
}

func waitForMeilisearchTask(info *meilisearch.TaskInfo, client *meilisearch.Client) bool {
	for {
		task, err := client.GetTask(info.TaskUID)
		if err != nil {
			log.Fatal("Failed to get task:", err)
			return false
		}
		if task.Status == "failed" {
			if task.Error.Code == "index_already_exists" {
				return true
			}
			log.Error("MeiliSearch task failed:", task.Error.Message, "-", task.Error.Code)
			return false
		}
		if task.Status == "succeeded" {
			return true
		}
		time.Sleep(time.Millisecond * 500)
	}
}
