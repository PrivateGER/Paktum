package main

import (
	"github.com/corona10/goimagehash"
	log "github.com/sirupsen/logrus"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

func DecodeImage(r io.Reader) image.Image {
	decodedImg, format, err := image.Decode(r)
	if err != nil {
		log.Error("Failed to decode image:", err.Error())
		return nil
	}
	log.Trace("Decoded image of format ", format)
	return decodedImg
}

func GeneratePHash(image image.Image) uint64 {
	log.Trace("Generating PHash for image with size ", image.Bounds().Size())
	hash, err := goimagehash.PerceptionHash(image)

	if err != nil {
		log.Error("Failed to generate pHash:", err.Error())
		return 0
	}

	log.Trace("Generated PHash ", hash, " for image with size ", image.Bounds().Size())

	return hash.GetHash()
}
