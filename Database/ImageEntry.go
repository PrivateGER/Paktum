package Database

import (
	"Paktum/ImageScraper"
	"Paktum/graph/model"
	"bytes"
	"context"
	"encoding/gob"
	"github.com/go-redis/redis/v8"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"
)

type ImageEntry struct {
	ID        string   `json:"ID"`
	URL       string   `json:"URL"`
	Tags      []string `json:"Tags"`
	Tagstring string   `json:"Tagstring"`
	Rating    string   `json:"Rating"`
	Added     string   `json:"Added"`
	PHash     uint64   `json:"PHash"`
	Size      int      `json:"Size"`
	Width     int      `json:"Width"`
	Height    int      `json:"Height"`
	Filename  string   `json:"Filename"`
}

var meiliClient *meilisearch.Client
var redisClient *redis.Client

func ConnectMeilisearch(host string, apiKey string) *meilisearch.Client {
	meiliClient = meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   host,
		APIKey: apiKey,
	})

	return meiliClient
}

func ConnectRedis(host string, password string, db int) *redis.Client {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       db,
	})

	return redisClient
}

var baseURL string

func SetBaseURL(url string) {
	baseURL = url
}

func GetBaseURL() string {
	if baseURL == "" {
		panic("Base URL not set")
	}

	return baseURL
}

func GetMeiliClient() *meilisearch.Client {
	if meiliClient == nil {
		panic("Meili client not initialized")
	}

	if !meiliClient.IsHealthy() {
		panic("Meili client is not healthy")
	}

	log.Debug("Meili client is healthy and initialized, returning instance")

	return meiliClient
}

func GetRedis() *redis.Client {
	if redisClient == nil {
		panic("Redis client not initialized")
	}

	// check if redis is healthy
	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		panic("Redis client is not healthy")
	}

	log.Debug("Redis client is healthy and initialized, returning instance")

	return redisClient
}

var lastPHashFetch uint64
var phashGroupMap [][]PHashEntry

func GetPHashGroup(pHash uint64) ([][]PHashEntry, error) {
	// If the last fetch of this was longer than 5 minutes ago, fetch a fresh copy from the redis db
	if lastPHashFetch+300 < uint64(time.Now().Unix()) {
		groupGob, err := redisClient.Get(context.TODO(), "paktum:image_alts").Result()
		if err != nil {
			return nil, err
		}

		var groupMap [][]PHashEntry
		dec := gob.NewDecoder(bytes.NewBuffer([]byte(groupGob)))
		err = dec.Decode(&groupMap)
		if err != nil {
			log.Error("Failed to decode image gob:", err.Error())
			return nil, err
		}

		phashGroupMap = groupMap
		lastPHashFetch = uint64(time.Now().Unix())
	}

	return phashGroupMap, nil
}

/* SearchImages searches an image in the database by the tagstring
 * @param query The tagstring to search for
 * @param limit The maximum number of results to return
 * @param shuffle Whether to return the results in a random order
 * @return A list of ImageEntry objects, the total number of results, and an possible error
 */
func SearchImages(query string, limit int, shuffle bool) ([]ImageEntry, int, error) {
	imageIndex := meiliClient.Index("images")

	// We first run a search to get the total results for this query
	// This way we can run the "proper" search with a randomized offset, giving unique results every time
	resultCountSearch, err := imageIndex.Search(query, &meilisearch.SearchRequest{
		Limit: 1,
	})
	if err != nil {
		return []ImageEntry{}, 0, err
	}
	if resultCountSearch.EstimatedTotalHits == 0 {
		return []ImageEntry{}, 0, nil
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
		return nil, 0, err
	}

	var results []ImageEntry
	for _, hit := range search.Hits {
		value := hit.(map[string]interface{})
		var tags []string
		for _, tag := range value["Tags"].([]interface{}) {
			// check if tag is a banned tag, if so don't include image
			if ImageScraper.TagIsBanned(tag.(string)) {
				continue
			}
			tags = append(tags, tag.(string))
		}

		results = append(results, ImageEntry{
			ID:        value["ID"].(string),
			URL:       GetBaseURL() + "/images/" + value["Filename"].(string),
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
	if shuffle {
		rand.Shuffle(len(results), func(i, j int) {
			results[i], results[j] = results[j], results[i]
		})
	}

	return results, int(search.EstimatedTotalHits), nil
}

func GetImageEntryFromID(id string) (ImageEntry, error) {
	var image ImageEntry

	err := meiliClient.Index("images").GetDocument(id, &meilisearch.DocumentQuery{
		Fields: nil,
	}, &image)
	if err != nil {
		return ImageEntry{}, err
	}

	image.URL = GetBaseURL() + "/images/" + image.Filename
	return image, nil
}

func GetRelatedImages(id string) ([]ImageEntry, error) {
	images, err := GetRelatedImageIDs(id)
	if err != nil {
		return nil, err
	}

	var imageEntries []ImageEntry
	for _, image := range images {
		entry, err := GetImageEntryFromID(image)
		if err != nil {
			return nil, err
		}
		imageEntries = append(imageEntries, entry)
	}

	return imageEntries, nil
}

func GetRelatedImageIDs(id string) ([]string, error) {
	image, err := GetImageEntryFromID(id)
	if err != nil {
		return nil, err
	}

	// Get the phash group for this image
	phashGroups, err := GetPHashGroup(image.PHash)
	if err != nil {
		return nil, err
	}

	for _, group := range phashGroups {
		for _, entry := range group {
			if entry.ID == id {
				// we found the group, generate slice with IDs, minus the given image
				var ids []string
				for _, entry := range group {
					if entry.ID != id {
						ids = append(ids, entry.ID)
					}
				}

				return ids, nil
			}
		}
	}

	return nil, nil
}

func GetRandomImage() (ImageEntry, error) {
	imageIndex := meiliClient.Index("images")

	// We first run a search to get the total count of documents
	// This gives us an offset we can use to get a random image
	resultCountSearch, err := imageIndex.Search("", &meilisearch.SearchRequest{
		Limit: 1,
	})
	if err != nil {
		return ImageEntry{}, err
	}
	if resultCountSearch.EstimatedTotalHits == 0 {
		return ImageEntry{}, nil
	}

	maxOffset := int(resultCountSearch.EstimatedTotalHits) - 1
	offset := rand.Intn(maxOffset)

	// Offset is now randomized between 0 and result count - limit, so we can always get unique results
	// and return one, which is random
	var res meilisearch.DocumentsResult
	err = imageIndex.GetDocuments(&meilisearch.DocumentsQuery{
		Fields: []string{"ID", "PHash", "Filename", "Tagstring", "Tags", "Rating", "Added", "Size", "Width", "Height"},
		Limit:  1,
		Offset: int64(offset),
	}, &res)

	if err != nil {
		return ImageEntry{}, err
	}

	var image ImageEntry
	for _, hit := range res.Results {
		value := hit
		var tags []string
		for _, tag := range value["Tags"].([]interface{}) {
			// check if tag is a banned tag, if so don't include image
			if ImageScraper.TagIsBanned(tag.(string)) {
				continue
			}
			tags = append(tags, tag.(string))
		}

		image = ImageEntry{
			ID:        value["ID"].(string),
			URL:       GetBaseURL() + "/images/" + value["Filename"].(string),
			Tags:      tags,
			Tagstring: value["Tagstring"].(string),
			Rating:    value["Rating"].(string),
			Added:     value["Added"].(string),
			PHash:     uint64(value["PHash"].(float64)),
			Size:      int(value["Size"].(float64)),
			Width:     int(value["Width"].(float64)),
			Height:    int(value["Height"].(float64)),
			Filename:  value["Filename"].(string),
		}
	}

	return image, nil
}

func DBImageToGraphImage(image ImageEntry) *model.Image {
	return &model.Image{
		ID:        image.ID,
		URL:       image.URL,
		Tags:      image.Tags,
		Tagstring: image.Tagstring,
		Rating:    model.Rating(image.Rating),
		Added:     image.Added,
		PHash:     strconv.FormatUint(image.PHash, 10),
		Size:      image.Size,
		Width:     image.Width,
		Height:    image.Height,
		Filename:  image.Filename,
	}
}
