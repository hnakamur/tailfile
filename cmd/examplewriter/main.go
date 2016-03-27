package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Println("Usage: logwriter filename rename")
		os.Exit(1)
	}
	filename := flag.Arg(0)
	rename := flag.Arg(1)

	interval := time.Duration(9) * time.Millisecond
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal()
	}

	i := 0
	for ; i < 50; i++ {
		_, err := file.WriteString(fmt.Sprintf("line%d\n", i))
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(interval)
	}

	err = os.Rename(filename, rename)
	if err != nil {
		log.Fatal(err)
	}
	for ; i < 100; i++ {
		_, err := file.WriteString(fmt.Sprintf("line%d\n", i))
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(interval)
	}
	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}

	file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal()
	}
	for ; i < 150; i++ {
		_, err := file.WriteString(fmt.Sprintf("line%d\n", i))
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(interval)
	}
	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
}
