package main

import (
	"Paktum/Database"
	"bytes"
	"encoding/gob"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"
)

func ServerMode(meili *meilisearch.Client, redis *redis.Client, imageDir string, baseURL string) {
	rand.Seed(time.Now().UnixNano())
	r := gin.Default()
	imageIndex := meili.Index("images")

	r.GET("/api/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(400, gin.H{
				"error": "No query provided",
			})
			return
		}
		limitString := c.Query("limit")
		if limitString == "" {
			limitString = "10"
		}
		limit, err := strconv.Atoi(limitString)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "Invalid limit provided (0 < limit <= 50)",
			})
			return
		}

		// We first run a search to get the total results for this query
		// This way we can run the "proper" search with a randomized offset, giving unique results every time
		resultCountSearch, err := imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit: 1,
		})
		if err != nil {
			c.JSON(500, gin.H{
				"error": "Failed to search for query",
			})
		}
		if resultCountSearch.EstimatedTotalHits == 0 {
			c.JSON(200, gin.H{
				"results": []string{},
				"error":   "",
			})
			return
		}

		maxOffset := int(resultCountSearch.EstimatedTotalHits) - limit
		if maxOffset < 0 {
			maxOffset = int(resultCountSearch.EstimatedTotalHits)
		}
		offset := rand.Intn(maxOffset)

		// Offset is now randomized between 0 and result count - limit, so we can always get unique results
		// and return enough results to fulfill the limit
		search, err := imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit:  int64(limit),
			Offset: int64(offset),
		})

		if err != nil {
			c.JSON(500, gin.H{
				"error": "Failed to search for query",
			})
		}

		var results []Database.ImageEntry
		for _, hit := range search.Hits {
			value := hit.(map[string]interface{})
			var tags []string
			for _, tag := range value["Tags"].([]interface{}) {
				tags = append(tags, tag.(string))
			}

			results = append(results, Database.ImageEntry{
				ID:        value["ID"].(string),
				URL:       baseURL + "/images/" + value["Filename"].(string),
				Tags:      tags,
				Tagstring: value["Tagstring"].(string),
				Rating:    value["Rating"].(string),
				Added:     value["Added"].(string),
				PHash:     uint64(value["PHash"].(float64)),
				Size:      int(value["Size"].(float64)),
				Width:     int(value["Width"].(float64)),
				Height:    int(value["Height"].(float64)),
				Filename:  value["Filename"].(string),
			})
		}
		rand.Shuffle(len(results), func(i, j int) {
			results[i], results[j] = results[j], results[i]
		})

		c.JSON(200, gin.H{
			"results":    results,
			"error":      "",
			"total_hits": resultCountSearch.EstimatedTotalHits,
		})
	})

	r.GET("/api/image/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(400, gin.H{
				"error": "No ID provided",
			})
			return
		}

		var image Database.ImageEntry
		err := imageIndex.GetDocument(id, &meilisearch.DocumentQuery{
			Fields: nil,
		}, &image)
		if err != nil {
			c.JSON(404, gin.H{
				"error": "image not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"image": image,
			"error": "",
		})
	})

	r.GET("/api/image/:id/related", func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(400, gin.H{
				"error": "No ID provided",
			})
			return
		}

		var image Database.ImageEntry
		err := imageIndex.GetDocument(id, &meilisearch.DocumentQuery{
			Fields: nil,
		}, &image)
		if err != nil {
			c.JSON(404, gin.H{
				"error": "image not found",
			})
			return
		}

		groupGob, err := redis.Get(c, "paktum:image_alts").Result()
		if err != nil {
			c.JSON(500, gin.H{
				"error": "Failed to get image alts from redis",
			})
			return
		}

		var groupMap [][]Database.PHashEntry
		dec := gob.NewDecoder(bytes.NewBuffer([]byte(groupGob)))
		err = dec.Decode(&groupMap)
		if err != nil {
			log.Error("Failed to decode image gob:", err.Error())
			c.JSON(500, gin.H{
				"error": "Failed to decode variants gob",
			})
		}

		// find the phash of the image in a group
		for _, group := range groupMap {
			for _, entry := range group {
				if entry.ID == id {
					// we found the group, generate slice with IDs, minus the given image
					var ids []string
					for _, entry := range group {
						if entry.ID != id {
							ids = append(ids, entry.ID)
						}
					}

					c.JSON(200, gin.H{
						"results": ids,
						"error":   "",
					})
					return
				}
			}
		}

		c.JSON(200, gin.H{
			"results": []string{},
			"error":   "",
		})
	})

	r.Static("/images/", imageDir)

	err := r.Run()
	if err != nil {
		log.Fatal("Failed to start server:", err)
	} // listen and serve on
}
