package tailfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"gopkg.in/fsnotify.v1"
)

type TailFile struct {
	targetAbsPath              string
	renamedAbsPath             string
	dirWatcher                 *fsnotify.Watcher
	pollingIntervalAfterRename time.Duration
	file                       *os.File
	reader                     *bufio.Reader
	logger                     Logger

	Lines  chan string // The channel to receiving log lines while reading the log file
	Errors chan error  // The channel to receiving an error while reading the log file
}

// NewTailFile starts watching the directory for the target file and opens the target file if it exists.
// The target file may not exist at the first. In that case, TailFile opens the target file as soon as
// the target file is created and written in the later time.
// pollingIntervalAfterRename will be used in reading logs which is written after the target file is renamed.
// The debug logs are written with logger. You can pass nil if you don't want the debug logs to be printed.
func NewTailFile(filename string, pollingIntervalAfterRename time.Duration, logger Logger) (*TailFile, error) {
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
		logger: logger,
		Lines:  make(chan string),
		Errors: make(chan error),
	}
	err = t.tryOpenFile()
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Close closes the target file and the directory watcher.
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

// ReadLoop runs a loop for reading the target file. Please use this with a goroutine like:
//  go t.ReadLoop()
func (t *TailFile) ReadLoop(ctx context.Context) {
	var readingRenamedFile bool
	for {
		select {
		case ev := <-t.dirWatcher.Events:
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
					if t.logger != nil {
						t.logger.Log("target file is written again after rename.")
					}
					t.readLines()
					err = t.closeFile()
					if err != nil {
						t.Errors <- err
						return
					}
					if t.logger != nil {
						t.logger.Log("closed renamed file")
					}
					readingRenamedFile = false
				}
				err := t.tryOpenFile()
				if err != nil {
					t.Errors <- err
					return
				}
				t.readLines()
			case fsnotify.Rename:
				if t.logger != nil {
					t.logger.Log("File renamed. read until EOF, then close")
				}
				filename, err := getFilenameFromFd(t.file.Fd())
				if err != nil {
					t.Errors <- err
					return
				}
				if t.logger != nil {
					t.logger.Log(fmt.Sprintf("filename after rename: %s", filename))
				}
				t.renamedAbsPath = filename
				readingRenamedFile = true
				t.readLines()
			case fsnotify.Remove:
				if t.logger != nil {
					t.logger.Log("file removed. closing")
				}
				err := t.closeFile()
				if err != nil {
					t.Errors <- err
					return
				}
				if t.logger != nil {
					t.logger.Log("closed removed file")
				}
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
		case <-ctx.Done():
			if t.logger != nil {
				t.logger.Log("received ctx.Done. exiting ReadLoop")
			}
			return
		}
	}
}
