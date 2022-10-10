package Database

type ImageEntry struct {
	ID        string   `json:"ID"`
	URL       string   `json:"URL"`
	Tags      []string `json:"Tags"`
	Tagstring string   `json:"Tagstring"`
	Rating    string   `json:"Rating"`
	Added     string   `json:"Added"`
	PHash     uint64   `json:"PHash"`
	Size      int      `json:"Size"`
	Width     int      `json:"Width"`
	Height    int      `json:"Height"`
	Filename  string   `json:"Filename"`
}
