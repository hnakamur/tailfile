package tailfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/fsnotify.v1"
)

type TailFile struct {
	targetAbsPath              string
	dirWatcher                 *fsnotify.Watcher
	pollingIntervalAfterRename time.Duration
	file                       *os.File
	reader                     *bufio.Reader

	Lines  chan string
	Errors chan error
}

func NewTailFile(filename string, pollingIntervalAfterRename time.Duration) (*TailFile, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	dirWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(absPath)
	err = dirWatcher.Add(dir)
	if err != nil {
		return nil, err
	}

	t := &TailFile{
		targetAbsPath:              absPath,
		dirWatcher:                 dirWatcher,
		pollingIntervalAfterRename: pollingIntervalAfterRename,
		Lines:  make(chan string),
		Errors: make(chan error),
	}
	err = t.tryOpenFile()
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TailFile) Close() error {
	err := t.closeFile()
	if t.dirWatcher != nil {
		err2 := t.dirWatcher.Close()
		if err == nil && err2 != nil {
			err = err2
		}
	}
	return err
}

func (t *TailFile) tryOpenFile() error {
	if t.file == nil {
		file, err := os.Open(t.targetAbsPath)
		if err != nil {
			if err2, ok := err.(*os.PathError); ok && err2.Err == syscall.ENOENT {
				return nil
			} else {
				return err
			}
		}

		t.file = file
		t.reader = bufio.NewReader(file)
	}
	return nil
}

func (t *TailFile) closeFile() error {
	if t.file != nil {
		err := t.file.Close()
		if err != nil {
			return err
		}
		t.reader = nil
		t.file = nil
	}
	return nil
}

func (t *TailFile) readLines() {
	if t.reader == nil {
		return
	}
	for {
		line, err := t.reader.ReadString(byte('\n'))
		if err != nil {
			if err == io.EOF {
				return
			}
			t.Errors <- err
			return
		}
		t.Lines <- line
	}
}

func (t *TailFile) ReadLoop() {
	var readingRenamedFile bool
	for {
		select {
		case ev := <-t.dirWatcher.Events:
			//fmt.Printf("ev=%v\n", ev)
			evAbsPath, err := filepath.Abs(ev.Name)
			if err != nil {
				t.Errors <- err
				return
			}
			if evAbsPath != t.targetAbsPath {
				continue
			}
			switch ev.Op {
			case fsnotify.Write:
				if readingRenamedFile {
					fmt.Println("target file is written again after rename.")
					t.readLines()
					err = t.closeFile()
					if err != nil {
						t.Errors <- err
						return
					}
					fmt.Println("closed renamed file")
					readingRenamedFile = false
				}
				err := t.tryOpenFile()
				if err != nil {
					t.Errors <- err
					return
				}
				t.readLines()
			case fsnotify.Rename:
				fmt.Println("File renamed. read until EOF, then close")
				readingRenamedFile = true
				t.readLines()
			case fsnotify.Remove:
				fmt.Println("file removed. closing")
				err := t.closeFile()
				if err != nil {
					t.Errors <- err
					return
				}
				fmt.Println("closed removed file")
			default:
				// do nothing
			}
		case <-time.After(t.pollingIntervalAfterRename):
			if readingRenamedFile {
				t.readLines()
			}
		case err := <-t.dirWatcher.Errors:
			t.Errors <- err
			return
		}
	}
}
