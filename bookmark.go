package tailfile

import (
	"encoding/json"
	"os"
)

type Bookmark struct {
	OriginalPath string `json:"originalPath"`
	WatchingPath string `json:"watchingPath"`
	Position     int64  `json:"position,string"`
}

func (b *Bookmark) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(&b)
}

func (b *Bookmark) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(&b)
}
