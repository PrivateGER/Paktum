package ImageScraper

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type GelbooruPage struct {
	Attributes struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Count  int `json:"count"`
	} `json:"@attributes"`
	GelbooruImage []struct {
		ID            int    `json:"id"`
		CreatedAt     string `json:"created_at"`
		Score         int    `json:"score"`
		Width         int    `json:"width"`
		Height        int    `json:"height"`
		Md5           string `json:"md5"`
		Directory     string `json:"directory"`
		Image         string `json:"image"`
		Rating        string `json:"rating"`
		Source        string `json:"source"`
		Change        int    `json:"change"`
		Owner         string `json:"owner"`
		CreatorID     int    `json:"creator_id"`
		ParentID      int    `json:"parent_id"`
		Sample        int    `json:"sample"`
		PreviewHeight int    `json:"preview_height"`
		PreviewWidth  int    `json:"preview_width"`
		Tags          string `json:"tags"`
		Title         string `json:"title"`
		HasNotes      string `json:"has_notes"`
		HasComments   string `json:"has_comments"`
		FileURL       string `json:"file_url"`
		PreviewURL    string `json:"preview_url"`
		SampleURL     string `json:"sample_url"`
		SampleHeight  int    `json:"sample_height"`
		SampleWidth   int    `json:"sample_width"`
		Status        string `json:"status"`
		PostLocked    int    `json:"post_locked"`
		HasChildren   string `json:"has_children"`
	} `json:"post"`
}

func scrape(tags []string, page uint) (error, []Image) {
	url := "https://gelbooru.com/index.php?page=dapi&s=post&q=index&pid=" + strconv.Itoa(int(page)) + "&json=1&tags=" + strings.Join(tags, "+")
	log.Trace("Requesting Gelbooru page with URL: ", url)

	httpClient := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err, nil
	}
	req.Header.Set("User-Agent", "Paktum Scraper/Importer")

	res, err := httpClient.Do(req)

	if err != nil {
		return err, nil
	}
	if res.Body == nil && res.StatusCode != http.StatusOK {
		return nil, nil
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	var posts GelbooruPage
	jsonErr := json.Unmarshal(body, &posts)
	if jsonErr != nil {
		return jsonErr, nil
	}

	var images []Image
	for _, post := range posts.GelbooruImage {
		images = append(images, Image{
			ID:          string(strconv.Itoa(post.ID)),
			Filename:    post.Image,
			FileURL:     post.FileURL,
			Tags:        strings.Split(post.Tags, " "),
			Description: post.Title,
			Rating:      post.Rating,
		})
	}

	return nil, images
}

func Gelbooru(tags []string) (error, []Image) {
	imageList := make([]Image, 0)

	for i := 0; i < 5; i++ {
		err, images := scrape(tags, uint(i))
		if err != nil {
			log.Error("Failed to scrape Gelbooru: ", err)
			return err, nil
		}
		log.Info("Got ", len(images), " images from Gelbooru tags ", tags, ", page ", i)
		imageList = append(imageList, images...)
		if len(images) == 0 {
			log.Info("Reached end of Gelbooru tag ", tags, " at page ", i)
			break
		}
	}

	return nil, imageList
}
