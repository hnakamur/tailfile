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
	go t.ReadLoop()
	for {
		select {
		case line := <-t.Lines:
			fmt.Printf("line=%s", line)
		case err := <-t.Errors:
			fmt.Printf("error from tail. err=%s\n", err)
			break
		default:
			// do nothing
		}
	}
	fmt.Println("exiting main")
}
