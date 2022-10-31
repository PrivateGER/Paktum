// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"
)

// A full image with all available metadata.
type Image struct {
	ID        string   `json:"ID"`
	URL       string   `json:"Url"`
	Tags      []string `json:"Tags"`
	Tagstring string   `json:"Tagstring"`
	Rating    Rating   `json:"Rating"`
	Added     string   `json:"Added"`
	// uint64 perception hash encoded as String. They can be compared using Hamming distance.
	PHash string `json:"PHash"`
	// Size in bytes.
	Size     int    `json:"Size"`
	Width    int    `json:"Width"`
	Height   int    `json:"Height"`
	Filename string `json:"Filename"`
	// Images that are similar to this one, based on perception-hashing. By default a distance of 10 is considered related.
	Related []*NestedImage `json:"Related"`
}

// An image that is nested in some way. This does not contain the Related field, but is otherwise identical to Image.
type NestedImage struct {
	ID        string   `json:"ID"`
	URL       string   `json:"Url"`
	Tags      []string `json:"Tags"`
	Tagstring string   `json:"Tagstring"`
	Rating    Rating   `json:"Rating"`
	Added     string   `json:"Added"`
	PHash     string   `json:"PHash"`
	Size      int      `json:"Size"`
	Width     int      `json:"Width"`
	Height    int      `json:"Height"`
	Filename  string   `json:"Filename"`
}

// The safety rating.
// General is SFW, Safe is SFW but may contain some adult content, and questionable up should be considered NSFW.
type Rating string

const (
	RatingExplicit     Rating = "explicit"
	RatingQuestionable Rating = "questionable"
	RatingSafe         Rating = "safe"
	RatingGeneral      Rating = "general"
)

var AllRating = []Rating{
	RatingExplicit,
	RatingQuestionable,
	RatingSafe,
	RatingGeneral,
}

func (e Rating) IsValid() bool {
	switch e {
	case RatingExplicit, RatingQuestionable, RatingSafe, RatingGeneral:
		return true
	}
	return false
}

func (e Rating) String() string {
	return string(e)
}

func (e *Rating) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = Rating(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid Rating", str)
	}
	return nil
}

func (e Rating) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
