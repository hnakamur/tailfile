package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hnakamur/tailfile"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Println("Usage: exampletail filename")
		os.Exit(1)
	}
	targetPath := flag.Arg(0)

	pollingIntervalAfterRename := time.Duration(50) * time.Millisecond
	t, err := tailfile.NewTailFile(targetPath, pollingIntervalAfterRename)
	if err != nil {
		log.Fatal(err)
	}
	i := 0
	err = t.EachLine(func(line string) (done bool, err error) {
		fmt.Printf("line=%s", line)
		i++
		if i > 130 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("exiting main")
}
