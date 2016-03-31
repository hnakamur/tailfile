package tailfile

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestBookmarkSaveAndLoad(t *testing.T) {
	file, err := ioutil.TempFile("", "tailfile-bookmark-example")
	if err != nil {
		t.Fatal("failed to create a temporary file: %s", err)
	}
	defer os.Remove(file.Name())

	b := Bookmark{
		OriginalPath: "/var/log/example/access.log",
		WatchingPath: "/var/log/example/access.log.old",
		Position:     234,
	}
	err = b.Save(file.Name())
	if err != nil {
		t.Fatal("failed to save a bookmark to file: %s", err)
	}

	var b2 Bookmark
	err = b2.Load(file.Name())
	if err != nil {
		t.Fatal("failed to load a saved bookmark: %s", err)
	}

	if b2.OriginalPath != b.OriginalPath {
		t.Errorf("Loaded OriginalPath %s; want %s", b2.OriginalPath, b.OriginalPath)
	}
	if b2.WatchingPath != b.WatchingPath {
		t.Errorf("Loaded WatchingPath %s; want %s", b2.WatchingPath, b.WatchingPath)
	}
	if b2.Position != b.Position {
		t.Errorf("Loaded Position %d; want %d", b2.Position, b.Position)
	}
}
