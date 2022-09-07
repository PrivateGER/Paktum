package main

import (
	"Paktum/ImageScraper"
	"fmt"
)

func main() {
	err, images := ImageScraper.Gelbooru([]string{"cat"}, "123")
	if err != nil {
		fmt.Println(err)
		return
	}

    for _, image := range images {
		fmt.Println(image.ID)
		fmt.Println(image.Filename)
		fmt.Println(image.Description)
		fmt.Println(image.Tags)
	}
}
