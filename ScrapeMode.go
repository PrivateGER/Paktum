package main

import (
	"Paktum/ImageScraper"
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

func ScrapeMode(redisClient *redis.Client) {
	log.Info("Scraping mode launching")

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
				log.Error(err)
				progress <- 1
				return
			}

			//encode image array into gob and send to redis
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			err = enc.Encode(images)
			if err != nil {
				log.Error(err)
				progress <- 1
				return
			}

			log.Info("Sending ", len(images), " images to redis")
			_, err = redisClient.RPush(context.Background(), "paktum:metadata_process", buf.Bytes()).Result()
			if err != nil {
				log.Error("Failed to push data to redis:", err)
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