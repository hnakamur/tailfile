package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hnakamur/tailfile"
)

type myLogger struct {
	*log.Logger
}

func (l myLogger) Log(v interface{}) {
	l.Print(v)
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Println("Usage: exampletail filename")
		os.Exit(1)
	}
	targetPath := flag.Arg(0)

	//logger := myLogger{log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)}
	pollingIntervalAfterRename := time.Duration(50) * time.Millisecond
	t, err := tailfile.NewTailFile(targetPath, pollingIntervalAfterRename, nil)
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
