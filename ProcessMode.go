package main

import (
	"Paktum/Database"
	"Paktum/ImageScraper"
	"bytes"
	"context"
	"encoding/gob"
	"github.com/go-redis/redis/v8"
	"github.com/meilisearch/meilisearch-go"
	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func ProcessMode(redisClient *redis.Client, meiliClient *meilisearch.Client, imageDir string) {
	log.Info("Process mode launching, ingesting data from redis...")

	// read data from redis
	// decode gob
	// process images, check for duplicates
	// send to meili

	task, err := meiliClient.CreateIndex(&meilisearch.IndexConfig{
		Uid:        "images",
		PrimaryKey: "ID",
	})
	if err != nil {
		log.Fatal("Failed to create MeiliSearch index:", err)
	}
	if !waitForMeilisearchTask(task, meiliClient) {
		log.Fatal("Failed to create MeiliSearch index")
	}

	imageCollection := meiliClient.Index("images")
	task, err = imageCollection.UpdateFilterableAttributes(&[]string{"ID", "Tagstring"})
	if err != nil {
		log.Fatal("Failed to update filterable attributes:", err)
	}
	if !waitForMeilisearchTask(task, meiliClient) {
		os.Exit(1)
		return
	}

	for {
		// read from redis
		log.Info("Sending BLPOP to redis at key paktum:metadata_process")
		result, err := redisClient.BLPop(context.TODO(), time.Second*10, "paktum:metadata_process").Result()
		log.Info("Received BLPOP response from redis")
		if err != nil {
			if err != redis.Nil {
				log.Error("Error reading from redis:", err.Error())
			}
			continue
		}

		log.Debug("Got", len(result), "items from redis")

		var images []ImageScraper.Image
		dec := gob.NewDecoder(bytes.NewBuffer([]byte(result[1])))
		err = dec.Decode(&images)
		if err != nil {
			log.Error("Failed to decode image gob:", err.Error())
			continue
		}
		log.Debug("Decoded", len(images), "images")

		var wg sync.WaitGroup
		pbar := progressbar.Default(int64(len(images)), "Downloading...")

		type ProcessedImages struct {
			ImageIDs map[string]string
			mutex    sync.Mutex
		}
		processedImages := ProcessedImages{
			ImageIDs: make(map[string]string),
			mutex:    sync.Mutex{},
		}

		type WrappedMeiliDocs struct {
			Docs []Database.ImageEntry
			sync.Mutex
		}
		wrappedMeiliDocs := WrappedMeiliDocs{
			Docs: make([]Database.ImageEntry, 0, len(images)),
		}

		for _, image := range images {
			wg.Add(1)
			go func(image ImageScraper.Image, wg *sync.WaitGroup, imageCollection *meilisearch.Index, processedImages *ProcessedImages, pbar *progressbar.ProgressBar, wrappedMeiliDocs *WrappedMeiliDocs) {
				// check if image already exists
				// if it does, skip
				// if it doesn't, download and add to meili
				defer wg.Done()
				defer func(pbar *progressbar.ProgressBar, num int) {
					_ = pbar.Add(num)
				}(pbar, 1)

				md5 := strings.TrimSuffix(image.Filename, filepath.Ext(image.Filename))

				processedImages.mutex.Lock()
				if _, ok := processedImages.ImageIDs[md5]; ok {
					processedImages.mutex.Unlock()
					log.Info("Found MD5 already being processed, duplicate image in queue, skipping...")
					return
				}
				if imageExists(imageCollection, md5) {
					processedImages.mutex.Unlock()
					log.Info("Image", md5, "already exists, skipping...")
					return
				}
				processedImages.ImageIDs[md5] = image.Filename
				processedImages.mutex.Unlock()

				if len(md5) != 32 {
					log.Error("MD5 is not 32 characters long, skipping...")
					return
				}

				if len(image.Tags) == 0 {
					log.Error("Image has no tags, skipping...")
					return
				}

				if image.Rating != "explicit" && image.Rating != "questionable" && image.Rating != "safe" && image.Rating != "general" {
					log.Error("Image has no rating, skipping...")
					return
				}

				if image.FileURL == "" {
					log.Error("Image has malformed file URL ", image.FileURL, " , skipping...")
					return
				}

				err, phash := downloadImage(image.FileURL, imageDir, md5)
				if err != nil {
					log.Error("Failed to download image", image.Filename)
					return
				}

				wrappedMeiliDocs.Lock()
				wrappedMeiliDocs.Docs = append(wrappedMeiliDocs.Docs, Database.ImageEntry{
					ID:        md5,
					URL:       image.FileURL,
					Tags:      image.Tags,
					Tagstring: strings.Join(image.Tags, " "),
					Rating:    image.Rating,
					Added:     strconv.FormatUint(uint64(time.Now().Unix()), 10),
					PHash:     phash,
				})
				wrappedMeiliDocs.Unlock()

			}(image, &wg, imageCollection, &processedImages, pbar, &wrappedMeiliDocs)
		}

		wg.Wait()
		_ = pbar.Finish()
		log.Info("Finished processing image batch.")

		if len(wrappedMeiliDocs.Docs) > 0 {
			// add to meili
			_, err = imageCollection.AddDocuments(wrappedMeiliDocs.Docs)
			if err != nil {
				log.Error("Failed to add documents to MeiliSearch:", err.Error())
			}
		}
		log.Info("Sent image batch of size", len(wrappedMeiliDocs.Docs), "to MeiliSearch")
	}
}
