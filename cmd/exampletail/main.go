package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"golang.org/x/net/context"

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

	logger := myLogger{log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)}
	pollingIntervalAfterRename := time.Duration(50) * time.Millisecond
	t, err := tailfile.NewTailFile(targetPath, pollingIntervalAfterRename, logger)
	if err != nil {
		log.Fatal(err)
	}
	defer t.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	go t.ReadLoop(ctx)
loop:
	for {
		select {
		case line := <-t.Lines:
			fmt.Printf("line=%s", line)
		case err := <-t.Errors:
			fmt.Printf("error from tail. err=%s\n", err)
			break loop
		case s := <-c:
			fmt.Println("got signal:", s)
			cancel()
			break loop
		default:
			// do nothing
		}
	}
	fmt.Println("exiting main")
}
