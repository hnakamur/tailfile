package tailfile

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
)

//type myLogger struct {
//	*log.Logger
//}
//
//func (l myLogger) Log(v interface{}) {
//	l.Print(v)
//}

func ExampleTailCreateRenameRecreate() {
	dir, err := ioutil.TempDir("", "tailfile-example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "example.log")
	renamedPath := filepath.Join(dir, "example.log.old")
	bookmarkPath := filepath.Join(dir, "example.log.bookmark")

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

	//file, err := os.Create("/tmp/tailfile_test.log")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer file.Close()
	//logger := myLogger{log.New(file, "", log.LstdFlags|log.Lmicroseconds)}
	//t, err := NewTailFile(targetPath, bookmarkPath, logger)
	t, err := NewTailFile(targetPath, bookmarkPath, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer t.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go t.Run(ctx)
loop:
	for {
		select {
		case line := <-t.Lines:
			fmt.Printf("line=%s", line)
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
	//line=line0
	//line=line1
	//line=line2
	//line=line3
	//line=line4
	//line=line5
	//line=line6
	//line=line7
	//line=line8
	//line=line9
	//line=line10
	//line=line11
	//line=line12
	//line=line13
	//line=line14
	//got done
}

func ExampleTailCreateTruncate() {
	dir, err := ioutil.TempDir("", "tailfile-example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "example.log")
	bookmarkPath := filepath.Join(dir, "example.log.bookmark")

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

	t, err := NewTailFile(targetPath, bookmarkPath, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer t.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go t.Run(ctx)
loop:
	for {
		select {
		case line := <-t.Lines:
			fmt.Printf("line=%s", line)
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
	//line=line0
	//line=line1
	//line=line2
	//line=line3
	//line=line4
	//line=line5
	//line=line6
	//line=line7
	//line=line8
	//line=line9
	//got done
}
