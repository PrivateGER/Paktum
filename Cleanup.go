package main

import (
	"github.com/corona10/goimagehash"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
)

func CleanupMode(client *meilisearch.Client) {
	var docs meilisearch.DocumentsResult
	err := client.Index("images").GetDocuments(&meilisearch.DocumentsQuery{
		Fields: []string{"ID", "PHash"},
		Limit:  1000,
	}, &docs)
	if err != nil {
		log.Fatal("Failed to get documents from MeiliSearch:", err)
	}

	log.Info("Got ", len(docs.Results), " documents from MeiliSearch")

	// find duplicates using pHash
	for i, doc := range docs.Results {
		needleHash, _ := doc["PHash"].(float64)
		needleID, _ := doc["ID"].(string)

		if uint64(needleHash) == 0 {
			continue
		}

		log.Trace("Processing document ", i, " with ID ", needleID, " and pHash ", needleHash)

		hash := goimagehash.NewImageHash(uint64(needleHash), goimagehash.PHash)

		for j := i + 1; j < len(docs.Results); j++ {
			otherHash, _ := docs.Results[j]["PHash"].(float64)
			otherID, _ := docs.Results[j]["ID"].(string)

			if uint64(otherHash) == 0 {
				continue
			}

			otherImgHash := goimagehash.NewImageHash(uint64(otherHash), goimagehash.PHash)

			distance, err := hash.Distance(otherImgHash)
			if err != nil {
				log.Error("Failed to get distance between hashes: ", err)
				continue
			}

			if distance < 10 {
				log.Info("Found possible image duplicate with IDs ", needleID, " and ", otherID, " with distance ", distance)
			}
		}
	}
}
