package ImageScraper

import log "github.com/sirupsen/logrus"

type Image struct {
	ID          string
	Filename    string
	FileURL     string
	Tags        []string
	Description string
	Rating      string
}

func Scrape(tags []string) (error, [][]Image) {
	err, images := Gelbooru(tags)
	batchSize := 100
	var batches [][]Image
	if err != nil {
		log.Error("Failed to scrape Gelbooru: ", err)
		return err, nil
	}

	// go over images and split into batches of 50
	for i := 0; i < len(images); i += batchSize {
		end := i + batchSize
		if end > len(images) {
			end = len(images)
		}
		batches = append(batches, images[i:end])
	}

	return err, batches
}
