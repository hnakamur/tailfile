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
	}
	err = t.tryOpenFile()
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TailFile) Close() error {
	var err error
	if t.file != nil {
		err2 := t.file.Close()
		if err2 != nil {
			err = err2
		}
	}
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

func (t *TailFile) readAndPrintLoop() error {
	if t.reader == nil {
		return nil
	}
	for {
		line, err := t.reader.ReadString(byte('\n'))
		if err != nil {
			return err
		}
		fmt.Printf("line=%s", line)
	}
}

func (t *TailFile) Run() error {
	var readingRenamedFile bool
	for {
		select {
		case ev := <-t.dirWatcher.Events:
			//fmt.Printf("ev=%v\n", ev)
			evAbsPath, err := filepath.Abs(ev.Name)
			if err != nil {
				return err
			}
			if evAbsPath != t.targetAbsPath {
				continue
			}
			switch ev.Op {
			case fsnotify.Write:
				if readingRenamedFile {
					fmt.Println("target file is written again after rename.")
					err := t.readAndPrintLoop()
					if err != nil && err != io.EOF {
						return err
					}
					err = t.closeFile()
					if err != nil {
						return err
					}
					fmt.Println("closed renamed file")
					readingRenamedFile = false
				}
				err := t.tryOpenFile()
				if err != nil {
					return err
				}
				err = t.readAndPrintLoop()
				if err != nil {
					if err == io.EOF {
						continue
					}
					return err
				}
			case fsnotify.Rename:
				fmt.Println("File renamed. read until EOF, then close")
				readingRenamedFile = true
				err := t.readAndPrintLoop()
				if err != nil {
					if err == io.EOF {
						continue
					}
					return err
				}
			case fsnotify.Remove:
				fmt.Println("file removed. closing")
				err := t.closeFile()
				if err != nil {
					return err
				}
				fmt.Println("closed removed file")
			default:
				// do nothing
			}
		case <-time.After(t.pollingIntervalAfterRename):
			if readingRenamedFile {
				err := t.readAndPrintLoop()
				if err != nil {
					if err == io.EOF {
						continue
					}
					return err
				}
			}
		case err := <-t.dirWatcher.Errors:
			fmt.Printf("received error from watcher: %s", err)
			return err
		}
	}
}
