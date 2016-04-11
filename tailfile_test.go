package tailfile

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/context"
)

func ExampleTailCreateWrite() {
	dir, err := ioutil.TempDir("", "tailfile-example")
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "example.log")

	done := make(chan struct{})

	go func() {
		defer func() {
			done <- struct{}{}
		}()

		interval := time.Duration(9) * time.Millisecond
		time.Sleep(interval)
		file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			panic(err)
		}

		i := 0
		for ; i < 5; i++ {
			_, err := file.WriteString(fmt.Sprintf("line%d\n", i))
			if err != nil {
				panic(err)
			}
			time.Sleep(interval)
		}
	}()

	t, err := NewTailFile(targetPath, nil)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go t.Run(ctx)
loop:
	for {
		select {
		case line := <-t.Lines:
			fmt.Printf("line=%s\n", strings.TrimRight(line, "\n"))
		case err := <-t.Errors:
			fmt.Printf("error from tail. err=%s\n", err)
			break loop
		case <-done:
			fmt.Println("got done")
			cancel()
			break loop
		default:
			// do nothing
		}
	}

	// Output:
	// line=line0
	// line=line1
	// line=line2
	// line=line3
	// line=line4
	// got done
}

func ExampleTailCreateRenameRecreate() {
	dir, err := ioutil.TempDir("", "tailfile-example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "example.log")
	renamedPath := filepath.Join(dir, "example.log.old")

	done := make(chan struct{})

	go func() {
		defer func() {
			done <- struct{}{}
		}()

		interval := time.Duration(9) * time.Millisecond
		file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal(err)
		}

		i := 0
		for ; i < 5; i++ {
			_, err := file.WriteString(fmt.Sprintf("line%d\n", i))
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(interval)
		}

		err = os.Rename(targetPath, renamedPath)
		if err != nil {
			log.Fatal(err)
		}
		for ; i < 10; i++ {
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

		file, err = os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal()
		}
		for ; i < 15; i++ {
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
	}()

	t, err := NewTailFile(targetPath, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go t.Run(ctx)
loop:
	for {
		select {
		case line := <-t.Lines:
			fmt.Printf("line=%s\n", strings.TrimRight(line, "\n"))
		case err := <-t.Errors:
			fmt.Printf("error from tail. err=%s\n", err)
			break loop
		case <-done:
			fmt.Println("got done")
			cancel()
			break loop
		default:
			// do nothing
		}
	}

	// Output:
	// line=line0
	// line=line1
	// line=line2
	// line=line3
	// line=line4
	// line=line5
	// line=line6
	// line=line7
	// line=line8
	// line=line9
	// line=line10
	// line=line11
	// line=line12
	// line=line13
	// line=line14
	// got done
}

func ExampleTailCreateTruncate() {
	dir, err := ioutil.TempDir("", "tailfile-example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "example.log")

	done := make(chan struct{})

	go func() {
		defer func() {
			done <- struct{}{}
		}()

		interval := time.Duration(9) * time.Millisecond
		file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal(err)
		}

		i := 0
		for ; i < 5; i++ {
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

		file, err = os.OpenFile(targetPath, os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			log.Fatal()
		}
		for ; i < 10; i++ {
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
	}()

	t, err := NewTailFile(targetPath, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go t.Run(ctx)
loop:
	for {
		select {
		case line := <-t.Lines:
			fmt.Printf("line=%s\n", strings.TrimRight(line, "\n"))
		case err := <-t.Errors:
			fmt.Printf("error from tail. err=%s\n", err)
			break loop
		case <-done:
			fmt.Println("got done")
			cancel()
			break loop
		default:
			// do nothing
		}
	}

	// Output:
	// line=line0
	// line=line1
	// line=line2
	// line=line3
	// line=line4
	// line=line5
	// line=line6
	// line=line7
	// line=line8
	// line=line9
	// got done
}

func ExampleTailCreateDeleteRecreate() {
	dir, err := ioutil.TempDir("", "tailfile-example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "example.log")

	done := make(chan struct{})

	go func() {
		defer func() {
			done <- struct{}{}
		}()

		interval := time.Duration(9) * time.Millisecond
		file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal(err)
		}

		i := 0
		for ; i < 5; i++ {
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

		err = os.Remove(targetPath)
		if err != nil {
			log.Fatal(err)
		}

		file, err = os.OpenFile(targetPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal()
		}
		for ; i < 10; i++ {
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
	}()

	t, err := NewTailFile(targetPath, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go t.Run(ctx)
loop:
	for {
		select {
		case line := <-t.Lines:
			fmt.Printf("line=%s\n", strings.TrimRight(line, "\n"))
		case err := <-t.Errors:
			fmt.Printf("error from tail. err=%s\n", err)
			break loop
		case <-done:
			fmt.Println("got done")
			cancel()
			break loop
		default:
			// do nothing
		}
	}

	// Output:
	// line=line0
	// line=line1
	// line=line2
	// line=line3
	// line=line4
	// line=line5
	// line=line6
	// line=line7
	// line=line8
	// line=line9
	// got done
}
