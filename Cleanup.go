package main

import (
	"Paktum/Database"
	"bytes"
	"context"
	"encoding/gob"
	"github.com/corona10/goimagehash"
	"github.com/go-redis/redis/v8"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
	"time"
)

func CleanupMode(client *meilisearch.Client, redis *redis.Client) {
	var docs meilisearch.DocumentsResult
	err := client.Index("images").GetDocuments(&meilisearch.DocumentsQuery{
		Fields: []string{"ID", "PHash"},
		Limit:  1000,
	}, &docs)
	if err != nil {
		log.Fatal("Failed to get documents from MeiliSearch:", err)
	}

	log.Info("Got ", len(docs.Results), " documents from MeiliSearch")

	startTime := time.Now()

	duplicates := make(map[string][]Database.PHashEntry)

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
				duplicates[needleID] = append(duplicates[needleID], Database.PHashEntry{
					ID:       otherID,
					Hash:     uint64(otherHash),
					Distance: distance,
				})
			}
		}
	}

	var duplicateGroups [][]Database.PHashEntry

	// detect duplicate groups and add them to a nested array to have a simplified way to create graphs
	for originalKey, original := range duplicates {
		// check whether the key of this group exists in the groups array
		// if not, we create a new group and throw all sub-keys into it along with the main key
		// if it does, we add all sub-keys to the group, including the main key

		// find the index of the group where the original key is a member
		// if it is not a member of any group, it returns -1
		groupIndex := FindInside([]Database.PHashEntry{FindPHashFromID(originalKey, docs)}, duplicateGroups)
		if groupIndex == -1 {
			// create a new group
			duplicateGroups = append(duplicateGroups, append(original, FindPHashFromID(originalKey, docs)))
		} else {
			// add the original key to the group
			duplicateGroups[groupIndex] = append(duplicateGroups[groupIndex], FindPHashFromID(originalKey, docs))

			// add all sub-keys to the groups
			duplicateGroups[groupIndex] = MergeGroups(duplicateGroups[groupIndex], original)
		}
	}

	log.Info("Found ", len(duplicateGroups), " duplicate groups")
	for i, group := range duplicateGroups {
		log.Info("Group ", i, " contains ", len(group), " members")
		for _, member := range group {
			log.Info("Member ", member)
		}
	}

	log.Info("Finished in ", time.Since(startTime))

	// encode duplicateGroups with gob and store it in redis
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(duplicateGroups)
	if err != nil {
		log.Error(err)
		return
	}

	redis.Set(context.Background(), "paktum:image_alts", buf, 0)
}

func PHashExistsInGroup(hash uint64, group []Database.PHashEntry) bool {
	for _, member := range group {
		if member.Hash == hash {
			return true
		}
	}

	return false
}

func FindInside(haystack []Database.PHashEntry, groups [][]Database.PHashEntry) int {
	// goes through all groups and detects group where any in haystack is a member, if one is found, it returns the index of the group
	// if none is found, it returns -1
	for i, group := range groups {
		for _, member := range group {
			for _, needle := range haystack {
				if member.Hash == needle.Hash {
					return i
				}
			}
		}
	}

	return -1
}

func MergeGroups(originalGroup []Database.PHashEntry, newGroup []Database.PHashEntry) []Database.PHashEntry {
	for _, member := range newGroup {
		if !PHashExistsInGroup(member.Hash, originalGroup) {
			originalGroup = append(originalGroup, member)
		}
	}

	return originalGroup
}

func FindPHashFromID(id string, docs meilisearch.DocumentsResult) Database.PHashEntry {
	for _, doc := range docs.Results {
		if doc["ID"] == id {
			return Database.PHashEntry{
				ID:       id,
				Hash:     uint64(doc["PHash"].(float64)),
				Distance: 0,
			}
		}
	}

	return Database.PHashEntry{}
}
