package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"Paktum/Database"
	"Paktum/graph/generated"
	"Paktum/graph/model"
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
)

// Image is the resolver for the image field.
func (r *queryResolver) Image(ctx context.Context, id string) (*model.Image, error) {
	log.Info("Querying image with id ", id)
	image, err := Database.GetImageEntryFromID(id)
	if err != nil {
		return nil, err
	}

	allFields := graphql.CollectAllFields(ctx)
	shouldFetchRelated := false
	for _, field := range allFields {
		if field == "Related" {
			shouldFetchRelated = true
		}
	}

	relatedImages := make([]*model.NestedImage, 0)
	if shouldFetchRelated {
		log.Println("Fetching related images for image with ID ", id)
		related, err := Database.GetRelatedImages(image.ID)
		if err != nil {
			return nil, err
		}

		for _, relatedImage := range related {
			var convertedImage model.NestedImage
			err := copier.Copy(&convertedImage, &relatedImage)
			if err != nil {
				return nil, err
			}
			relatedImages = append(relatedImages, &convertedImage)
		}
	}

	var returnedImage = Database.DBImageToGraphImage(image)
	returnedImage.Related = relatedImages

	return returnedImage, nil
}

// RandomImage is the resolver for the randomImage field.
func (r *queryResolver) RandomImage(ctx context.Context) (*model.Image, error) {
	log.Info("Querying random image")
	image, err := Database.GetRandomImage()
	if err != nil {
		return nil, err
	}

	allFields := graphql.CollectAllFields(ctx)
	shouldFetchRelated := false
	for _, field := range allFields {
		if field == "Related" {
			shouldFetchRelated = true
		}
	}

	relatedImages := make([]*model.NestedImage, 0)
	if shouldFetchRelated {
		log.Println("Fetching related images for image with ID ", image.ID)
		related, err := Database.GetRelatedImages(image.ID)
		if err != nil {
			return nil, err
		}

		for _, relatedImage := range related {
			var convertedImage model.NestedImage
			err := copier.Copy(&convertedImage, &relatedImage)
			if err != nil {
				return nil, err
			}
			relatedImages = append(relatedImages, &convertedImage)
		}
	}

	convertedImage := Database.DBImageToGraphImage(image)
	convertedImage.Related = relatedImages

	return convertedImage, nil
}

// SearchImages is the resolver for the searchImages field.
func (r *queryResolver) SearchImages(ctx context.Context, query string) ([]*model.Image, error) {
	log.Info("Querying images with query ", query)
	images, _, err := Database.SearchImages(query, 10, true)
	if err != nil {
		return nil, err
	}

	allFields := graphql.CollectAllFields(ctx)
	shouldFetchRelated := false
	for _, field := range allFields {
		if field == "Related" {
			shouldFetchRelated = true
		}
	}

	var convertedImages []*model.Image
	for _, image := range images {
		var convertedImage model.Image
		err := copier.Copy(&convertedImage, &image)
		if err != nil {
			return nil, err
		}

		relatedImages := make([]*model.NestedImage, 0)
		if shouldFetchRelated {
			log.Println("Fetching related images for image with ID ", image.ID)
			related, err := Database.GetRelatedImages(image.ID)
			if err != nil {
				return nil, err
			}

			for _, relatedImage := range related {
				var nestedConvertedImage model.NestedImage
				err := copier.Copy(&nestedConvertedImage, &relatedImage)
				if err != nil {
					return nil, err
				}
				relatedImages = append(relatedImages, &nestedConvertedImage)
			}
		}
		convertedImage.Related = relatedImages

		convertedImages = append(convertedImages, &convertedImage)
	}

	return convertedImages, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
