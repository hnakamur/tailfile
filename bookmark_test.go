package tailfile

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func ExampleBookmarkSave() {
	file, err := ioutil.TempFile("", "tailfile-bookmark-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	bookmark := Bookmark{
		OriginalPath: "/var/log/example/access.log",
		WatchingPath: "/var/log/example/access.log.old",
		Position:     234,
	}
	err = bookmark.Save(file.Name())
	if err != nil {
		log.Fatal(err)
	}

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(buf))

	// Output:
	//{"originalPath":"/var/log/example/access.log","watchingPath":"/var/log/example/access.log.old","position":"234"}
}

func ExampleBookmarkLoad() {
	file, err := ioutil.TempFile("", "tailfile-bookmark-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	_, err = file.WriteString(`{"originalPath":"/var/log/example/access.log","watchingPath":"/var/log/example/access.log.old","position":"234"}`)
	if err != nil {
		log.Fatal(err)
	}

	var bookmark Bookmark
	err = bookmark.Load(file.Name())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("originalPath:%s\n", bookmark.OriginalPath)
	fmt.Printf("watchingPath:%s\n", bookmark.WatchingPath)
	fmt.Printf("position:%d\n", bookmark.Position)
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(buf))

	// Output:
	//originalPath:/var/log/example/access.log
	//watchingPath:/var/log/example/access.log.old
	//position:234
}
