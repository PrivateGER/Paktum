package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"Paktum/Database"
	"Paktum/graph/generated"
	"Paktum/graph/model"
	"context"

	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
)

// Related is the resolver for the Related field.
func (r *imageResolver) Related(ctx context.Context, obj *model.Image) ([]*model.NestedImage, error) {
	relatedImages := make([]*model.NestedImage, 0)

	log.Println("Fetching related images for image with ID ", obj.ID)
	related, err := Database.GetRelatedImages(obj.ID)
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

	return relatedImages, nil
}

// Image is the resolver for the image field.
func (r *queryResolver) Image(ctx context.Context, id string) (*model.Image, error) {
	log.Info("Querying image with id ", id)
	image, err := Database.GetImageEntryFromID(id)
	if err != nil {
		return nil, err
	}

	return Database.DBImageToGraphImage(image), nil
}

// RandomImage is the resolver for the randomImage field.
func (r *queryResolver) RandomImage(ctx context.Context) (*model.Image, error) {
	log.Info("Querying random image")
	image, err := Database.GetRandomImage()
	if err != nil {
		return nil, err
	}

	return Database.DBImageToGraphImage(image), nil
}

// SearchImages is the resolver for the searchImages field.
func (r *queryResolver) SearchImages(ctx context.Context, query string, limit int, shuffle *bool) ([]*model.Image, error) {
	log.Info("Querying images with query ", query)

	if shuffle == nil {
		shuffle = new(bool)
		*shuffle = true
	}

	if limit == 0 || limit > 100 {
		limit = 100
	}

	images, _, err := Database.SearchImages(query, limit, *shuffle)
	if err != nil {
		return nil, err
	}

	var convertedImages []*model.Image
	for _, image := range images {
		var convertedImage model.Image
		err := copier.Copy(&convertedImage, &image)
		if err != nil {
			return nil, err
		}

		convertedImages = append(convertedImages, &convertedImage)
	}

	return convertedImages, nil
}

// Image returns generated.ImageResolver implementation.
func (r *Resolver) Image() generated.ImageResolver { return &imageResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type imageResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
