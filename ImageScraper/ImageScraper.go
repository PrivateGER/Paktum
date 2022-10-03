package ImageScraper

type Image struct {
	ID          string
	Filename    string
	FileURL     string
	Tags        []string
	Description string
}

func Scrape(tags []string, taskuid string) (error, []Image) {
	err, images := Gelbooru(tags, taskuid)
	return err, images
}
