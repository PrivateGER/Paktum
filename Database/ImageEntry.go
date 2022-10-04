package Database

type ImageEntry struct {
	ID        string   `json:"ID"`
	URL       string   `json:"URL"`
	Tags      []string `json:"Tags"`
	Tagstring string   `json:"Tagstring"`
	Rating    string   `json:"Rating"`
	Added     string   `json:"Added"`
	PHash     uint64   `json:"PHash"`
}
