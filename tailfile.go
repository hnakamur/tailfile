package tailfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"golang.org/x/net/context"

	"gopkg.in/fsnotify.v0"
)

type TailFile struct {
	targetAbsPath       string
	dirPath             string
	watchingFileAbsPath string
	fileSize            int64
	fsWatcher           *fsnotify.Watcher
	file                *os.File
	reader              *bufio.Reader
	logger              Logger
	mu                  sync.Mutex

	Lines  chan string // The channel to receiving log lines while reading the log file
	Errors chan error  // The channel to receiving an error while reading the log file
}

// NewTailFile starts watching the directory for the target file and opens the target file if it exists.
// The target file may not exist at the first. In that case, TailFile opens the target file as soon as
// the target file is created and written in the later time.
func NewTailFile(filename string, logger Logger) (*TailFile, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dirPath := filepath.Dir(absPath)
	err = fsWatcher.WatchFlags(dirPath, fsnotify.FSN_CREATE|fsnotify.FSN_DELETE)
	if err != nil {
		return nil, err
	}

	t := &TailFile{
		targetAbsPath:       absPath,
		watchingFileAbsPath: absPath,
		dirPath:             dirPath,
		fsWatcher:           fsWatcher,
		logger:              logger,
		Lines:               make(chan string),
		Errors:              make(chan error),
	}
	err = t.tryOpenFile()
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Close closes the target file and the directory watcher.
func (t *TailFile) Close() error {
	t.mu.Lock() // synchronize with ReadLoop goroutine
	defer t.mu.Unlock()

	err := t.closeFile()
	err2 := t.fsWatcher.Close()
	if err != nil {
		return err
	} else if err2 != nil {
		return err2
	}
	return nil
}

func (t *TailFile) tryOpenFile() error {
	if t.file != nil {
		return nil
	}

	file, err := os.Open(t.targetAbsPath)
	if err != nil {
		if err2, ok := err.(*os.PathError); ok && err2.Err == syscall.ENOENT {
			return nil
		} else {
			return err
		}
	}

	fi, err := file.Stat()
	if err != nil {
		return err
	}
	t.fileSize = fi.Size()

	err = t.watchFile(t.watchingFileAbsPath)
	if err != nil {
		return err
	}

	t.file = file
	t.reader = bufio.NewReader(file)
	return nil
}

func (t *TailFile) closeFile() error {
	if t.file == nil {
		return nil
	}

	err2 := t.unwatchFile(t.watchingFileAbsPath)
	err := t.file.Close()
	if err != nil {
		return err
	} else if err2 != nil {
		return err2
	}

	t.reader = nil
	t.file = nil
	return nil
}

func (t *TailFile) watchFile(absPath string) error {
	return t.fsWatcher.WatchFlags(absPath, fsnotify.FSN_MODIFY|fsnotify.FSN_RENAME|fsnotify.FSN_DELETE)
}

func (t *TailFile) unwatchFile(absPath string) error {
	return t.fsWatcher.RemoveWatch(absPath)
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
	t.mu.Lock() // synchronize with Close
	defer t.mu.Unlock()

	for {
		select {
		case ev := <-t.fsWatcher.Event:
			//if t.logger != nil {
			//	t.logger.Log(fmt.Sprintf("event=%v", ev))
			//}
			switch {
			case ev.IsCreate() && ev.Name == t.targetAbsPath:
				if t.watchingFileAbsPath != t.targetAbsPath {
					if t.logger != nil {
						t.logger.Log("file recreated. reading rest of renamed file")
					}
					t.readLines()
					if t.logger != nil {
						t.logger.Log("file recreated. closing renamed file")
					}
					err := t.closeFile()
					if err != nil {
						t.Errors <- err
						return
					}
					if t.logger != nil {
						t.logger.Log("closed renamed file")
					}
					t.watchingFileAbsPath = t.targetAbsPath
				}
				err := t.tryOpenFile()
				if err != nil {
					t.Errors <- err
					return
				}
				t.readLines()
			case ev.IsModify() && ev.Name == t.watchingFileAbsPath:
				fi, err := t.file.Stat()
				if err != nil {
					t.Errors <- err
					return
				}
				if fi.Size() < t.fileSize {
					if t.logger != nil {
						t.logger.Log("file size got smaller. closing file")
					}
					err := t.closeFile()
					if err != nil {
						t.Errors <- err
						return
					}
				}
				t.fileSize = fi.Size()

				err = t.tryOpenFile()
				if err != nil {
					t.Errors <- err
					return
				}
				t.readLines()
			case ev.IsRename() && ev.Name == t.watchingFileAbsPath:
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
				err = t.unwatchFile(t.watchingFileAbsPath)
				if err != nil {
					t.Errors <- err
					return
				}
				err = t.watchFile(filename)
				if err != nil {
					t.Errors <- err
					return
				}
				t.watchingFileAbsPath = filename
				t.readLines()
			case ev.IsDelete():
				if ev.Name == t.watchingFileAbsPath {
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
				} else if ev.Name == t.dirPath {
					t.Errors <- fmt.Errorf("directory of watching file was deleted unexpectedly. filePath=%s", t.targetAbsPath)
				}
			}
		case err := <-t.fsWatcher.Error:
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
