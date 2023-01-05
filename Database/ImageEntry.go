package Database

import (
	"Paktum/ImageScraper"
	"Paktum/graph/model"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type ImageEntry struct {
	ID           string   `json:"ID"`
	URL          string   `json:"URL"`
	ThumbnailURL string   `json:"ThumbnailURL"`
	Tags         []string `json:"Tags"`
	Tagstring    string   `json:"Tagstring"`
	Rating       Rating   `json:"Rating"`
	Added        string   `json:"Added"`
	PHash        uint64   `json:"PHash"`
	Size         int      `json:"Size"`
	Width        int      `json:"Width"`
	Height       int      `json:"Height"`
	Filename     string   `json:"Filename"`
}

type Rating string

const (
	RatingExplicit     Rating = "explicit"
	RatingQuestionable Rating = "questionable"
	RatingSafe         Rating = "safe"
	RatingGeneral      Rating = "general"
)

var lastPHashFetch uint64
var phashGroupMap [][]PHashEntry

/* GetPHashes returns a list of all phash groups in the database
 * @return A list of phash groups
 */
func GetPHashGroups() ([][]PHashEntry, error) {
	// If the last fetch of this was longer than 5 minutes ago, fetch a fresh copy from the redis db
	if lastPHashFetch+300 < uint64(time.Now().Unix()) {
		groupGob, err := GetRedis().Get(context.TODO(), "paktum:image_alts").Result()
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
 * @param rating Return only images with this rating [if nil, accepts all]
 * @return A list of ImageEntry objects, the total number of results, and a possible error
 */
func SearchImages(query string, limit int, shuffle bool, rating string) ([]ImageEntry, int, error) {
	imageIndex := GetMeiliClient().Index("images")

	// We first run a search to get the total results for this query
	// This way we can run the "proper" search with a randomized offset, giving unique results every time
	var resultCountSearch *meilisearch.SearchResponse
	var err error
	if rating == "" {
		resultCountSearch, err = imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit: 1,
			Sort:  []string{"Added:desc"},
		})
	} else {
		log.Info("Searching with rating", rating)
		resultCountSearch, err = imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit:  1,
			Filter: "Rating = '" + rating + "'",
		})
	}

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

	// Offset is now randomized between 0 and result count - limit (if shuffle disabled), so we can always get unique results
	// and return enough results to fulfill the limit
	var search *meilisearch.SearchResponse
	if rating == "" && !shuffle {
		search, err = imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit: int64(limit),
			Sort:  []string{"Added:desc"},
		})
	} else if rating == "" && shuffle {
		search, err = imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit:  int64(limit),
			Offset: int64(offset),
			Sort:   []string{"Added:desc"},
		})
	} else if rating != "" && !shuffle {
		search, err = imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit:  int64(limit),
			Offset: int64(offset),
			Filter: "Rating = '" + rating + "'",
		})
	} else {
		search, err = imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit:  int64(limit),
			Offset: int64(offset),
			Filter: "Rating = '" + rating + "'",
			Sort:   []string{"Added:desc"},
		})
	}

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

		thumbnail := GetImgproxyBaseUrl() + SignImgproxyURL("rs:fill:480/g:sm/plain/local:///"+value["Filename"].(string))
		if strings.HasSuffix(thumbnail, ".webm") {
			thumbnail = ""
		}

		results = append(results, ImageEntry{
			ID:           value["ID"].(string),
			URL:          GetBaseURL() + "/images/" + value["Filename"].(string),
			ThumbnailURL: thumbnail,
			Tags:         tags,
			Tagstring:    value["Tagstring"].(string),
			Rating:       Rating(value["Rating"].(string)),
			Added:        value["Added"].(string),
			PHash:        uint64(value["PHash"].(float64)),
			Size:         int(value["Size"].(float64)),
			Width:        int(value["Width"].(float64)),
			Height:       int(value["Height"].(float64)),
			Filename:     value["Filename"].(string),
		})
	}
	if shuffle {
		rand.Shuffle(len(results), func(i, j int) {
			results[i], results[j] = results[j], results[i]
		})
	}

	return results, int(search.EstimatedTotalHits), nil
}

/* SearchImagesPaginated runs a search like SearchImages, but returns a paginated result
 * @param query The tagstring to search for
 * @param limit The number of results to return per page
 * @param page The page to return (1-indexed)
 * @param rating Return only images with this rating [if nil, accepts all]
 * @return A list of ImageEntry objects, the total number of results, and a possible error
 */
func SearchImagesPaginated(query string, limit int, page int, rating string) ([]ImageEntry, int, error) {
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "search",
		Message:  "Searching for " + query,
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"query":  query,
			"limit":  limit,
			"page":   page,
			"rating": rating,
		},
	})
	imageIndex := GetMeiliClient().Index("images")

	// limit is from 0 to 100
	if limit > 100 {
		limit = 100
	} else if limit < 0 {
		return []ImageEntry{}, 0, errors.New("limit must be greater than 0")
	}

	// page has to be greater than 0
	if page < 1 {
		return []ImageEntry{}, 0, errors.New("page must be greater than 0")
	}

	var search *meilisearch.SearchResponse
	var err error
	if rating == "" {
		search, err = imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit:  int64(limit),
			Offset: int64((page + 1) * limit),
			Sort:   []string{"Added:desc"},
		})
	} else {
		search, err = imageIndex.Search(query, &meilisearch.SearchRequest{
			Limit:  int64(limit),
			Offset: int64((page + 1) * limit),
			Filter: "Rating = '" + rating + "'",
			Sort:   []string{"Added:desc"},
		})
	}

	if err != nil {
		sentry.CaptureException(err)
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

		thumbnail := GetImgproxyBaseUrl() + SignImgproxyURL("rs:fill:480/g:sm/plain/local:///"+value["Filename"].(string))
		if strings.HasSuffix(thumbnail, ".webm") {
			thumbnail = ""
		}

		results = append(results, ImageEntry{
			ID:           value["ID"].(string),
			URL:          GetBaseURL() + "/images/" + value["Filename"].(string),
			ThumbnailURL: thumbnail,
			Tags:         tags,
			Tagstring:    value["Tagstring"].(string),
			Rating:       Rating(value["Rating"].(string)),
			Added:        value["Added"].(string),
			PHash:        uint64(value["PHash"].(float64)),
			Size:         int(value["Size"].(float64)),
			Width:        int(value["Width"].(float64)),
			Height:       int(value["Height"].(float64)),
			Filename:     value["Filename"].(string),
		})
	}

	return results, int(search.EstimatedTotalHits), nil
}

/* GetImageByID returns an image matching the given ID
 * @param id The ID of the image to return
 * @return The image entry, or nil if no image was found
 */
func GetImageEntryFromID(id string) (ImageEntry, error) {
	var image ImageEntry

	err := GetMeiliClient().Index("images").GetDocument(id, &meilisearch.DocumentQuery{
		Fields: nil,
	}, &image)
	if err != nil {
		return ImageEntry{}, err
	}

	image.URL = GetBaseURL() + "/images/" + image.Filename

	thumbnail := GetImgproxyBaseUrl() + SignImgproxyURL("rs:fill:480/g:sm/plain/local:///"+image.Filename)
	if strings.HasSuffix(thumbnail, ".webm") {
		thumbnail = ""
	}

	image.ThumbnailURL = thumbnail

	return image, nil
}

/* GetRelatedImages returns a list of images that are similar to the given image
 * Do not call this recursively, it will run infinitely
 * @param image The image to find similar images for
 * @return A list of similar images
 */
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

/* GetRelatedImageIDs returns a list of image IDs that are similar to the given image
 * Do not call this recursively, it will run infinitely
 * @param image The image to find similar images for
 * @return A list of similar image IDs
 */
func GetRelatedImageIDs(id string) ([]string, error) {
	// Get the phash group for this image
	phashGroups, err := GetPHashGroups()
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "phash",
		Message:  fmt.Sprintf("Got phash groups, count: %d", len(phashGroups)),
	})

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

/* GetRandomImage returns a random image from the database
 * @return The image entry, or nil if no image was found
 */
func GetRandomImage() (ImageEntry, error) {
	imageIndex := GetMeiliClient().Index("images")

	totalImageCount, err := GetTotalImageCount()
	if err != nil {
		sentry.CaptureException(err)
		return ImageEntry{}, err
	}

	maxOffset := totalImageCount - 1
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

		thumbnail := GetImgproxyBaseUrl() + SignImgproxyURL("rs:fill:480/g:sm/plain/local:///"+value["Filename"].(string))
		if strings.HasSuffix(thumbnail, ".webm") {
			thumbnail = ""
		}

		image = ImageEntry{
			ID:           value["ID"].(string),
			URL:          GetBaseURL() + "/images/" + value["Filename"].(string),
			ThumbnailURL: thumbnail,
			Tags:         tags,
			Tagstring:    value["Tagstring"].(string),
			Rating:       Rating(value["Rating"].(string)),
			Added:        value["Added"].(string),
			PHash:        uint64(value["PHash"].(float64)),
			Size:         int(value["Size"].(float64)),
			Width:        int(value["Width"].(float64)),
			Height:       int(value["Height"].(float64)),
			Filename:     value["Filename"].(string),
		}
	}

	return image, nil
}

/* GetTotalImageCount returns the total number of images in the database
 * @return The total number of images
 */
func GetTotalImageCount() (int, error) {
	imageIndex := GetMeiliClient().Index("images")

	// We first run a search to get the total count of documents
	// This gives us an offset we can use to get a random image
	resultCountSearch, err := imageIndex.Search("", &meilisearch.SearchRequest{
		Limit: 1,
	})
	if err != nil {
		return 0, err
	}
	if resultCountSearch.EstimatedTotalHits == 0 {
		return 0, nil
	}

	return int(resultCountSearch.EstimatedTotalHits), nil
}

func DBImageToGraphImage(image ImageEntry) *model.Image {
	return &model.Image{
		ID:           image.ID,
		URL:          image.URL,
		ThumbnailURL: image.ThumbnailURL,
		Tags:         image.Tags,
		Tagstring:    image.Tagstring,
		Rating:       model.Rating(image.Rating),
		Added:        image.Added,
		PHash:        strconv.FormatUint(image.PHash, 10),
		Size:         image.Size,
		Width:        image.Width,
		Height:       image.Height,
		Filename:     image.Filename,
	}
}

func SignImgproxyURL(path string) string {
	mac := hmac.New(sha256.New, GetImgproxyKey())
	mac.Write(GetImgproxySalt())
	mac.Write([]byte("/" + path))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return "/" + signature + "/" + path
}
