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
	bookmarkFilename    string
	bookmark            *Bookmark
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
func NewTailFile(filename, bookmarkFilename string, logger Logger) (*TailFile, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	t := &TailFile{
		targetAbsPath:       absPath,
		watchingFileAbsPath: absPath,
		bookmarkFilename:    bookmarkFilename,
		logger:              logger,
		Lines:               make(chan string),
		Errors:              make(chan error),
	}

	if bookmarkFilename != "" {
		bookmark := new(Bookmark)
		err := bookmark.Load(bookmarkFilename)
		if err != nil {
			if err2, ok := err.(*os.PathError); ok && err2.Err == syscall.ENOENT {
				if logger != nil {
					logger.Log(fmt.Sprintf("Bookmark file \"%s\" does not exist, ignoring.", bookmarkFilename))
				}
			} else {
				return nil, err
			}
		} else {
			if bookmark.OriginalPath != absPath {
				return nil, fmt.Errorf("Bookmark's originalPath \"%s\" does not match \"%s\"", bookmark.OriginalPath, filename)
			}
			t.bookmark = bookmark
			if logger != nil {
				logger.Log("Loaded bookmark")
			}
		}
	} else {
		if logger != nil {
			logger.Log("Bookmark filename specified was empty. We recommend you to specify a non-empty path")
		}
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	t.fsWatcher = fsWatcher

	dirPath := filepath.Dir(absPath)
	err = fsWatcher.WatchFlags(dirPath, fsnotify.FSN_CREATE|fsnotify.FSN_DELETE)
	if err != nil {
		return nil, err
	}
	t.dirPath = dirPath

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

	var file *os.File
	var exists bool
	var fileSize int64
	var err error
	if t.bookmark != nil {
		file, exists, err = openFileForReadingIfExist(t.bookmark.WatchingPath)
		if err != nil {
			return err
		}

		if exists {
			// We successfully opened the file we were watching when we save the bookmark before.
			if t.logger != nil {
				t.logger.Log(fmt.Sprintf("tryOpenFile before updating t.watchingFileAbsPath=\"%s\" with t.bookmark.WathingPath=\"%s\"", t.watchingFileAbsPath, t.bookmark.WatchingPath))
			}
			t.watchingFileAbsPath = t.bookmark.WatchingPath

			fileSize, err = getOpenedFileSize(file)
			if err != nil {
				return err
			}

			if t.bookmark.Position <= fileSize {
				if t.logger != nil {
					t.logger.Log(fmt.Sprintf("tryOpenFile: before seeeking \"%s\":%d fileSize=%d, fd=%d", t.watchingFileAbsPath, t.bookmark.Position, fileSize, file.Fd()))
				}
				pos, err := file.Seek(t.bookmark.Position, os.SEEK_SET)
				if err != nil {
					return err
				}

				if t.logger != nil {
					t.logger.Log(fmt.Sprintf("Opened \"%s\" and seeked to position %d (position in Bookmark: %d)", t.bookmark.WatchingPath, pos, t.bookmark.Position))
				}
			}

			t.bookmark = nil
		} else {
			// Copy to temporary variable for later use
			bookmarkWatchingPath := t.bookmark.WatchingPath

			// NOTE: Discard bookmark since the file we were watching does not exist.
			t.bookmark = nil

			if t.targetAbsPath == bookmarkWatchingPath {
				return nil
			}

			file, exists, err = openFileForReadingIfExist(t.targetAbsPath)
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}

			fileSize, err = getOpenedFileSize(file)
			if err != nil {
				return err
			}

			// NOTE: Start reading from the head without seeking
			//       since the file we were watching does not exist.
		}
	} else {
		if t.logger != nil {
			t.logger.Log(fmt.Sprintf("tryOpenFile opening t.targetAbsPath=%s (t.bookmark == nil)", t.targetAbsPath))
		}
		file, exists, err = openFileForReadingIfExist(t.targetAbsPath)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}

		fileSize, err = getOpenedFileSize(file)
		if err != nil {
			return err
		}
	}

	if t.logger != nil {
		t.logger.Log(fmt.Sprintf("tryOpenFile before watch file. t.watchingFileAbsPath=\"%s\"", t.watchingFileAbsPath))
	}
	err = t.watchFile(t.watchingFileAbsPath)
	if err != nil {
		return err
	}
	if t.logger != nil {
		t.logger.Log("tryOpenFile after watch file")
	}

	t.file = file
	t.fileSize = fileSize
	t.reader = bufio.NewReader(file)
	return nil
}

func openFileForReadingIfExist(path string) (file *os.File, exists bool, err error) {
	file, err = os.Open(path)
	if err != nil {
		if err2, ok := err.(*os.PathError); ok && err2.Err == syscall.ENOENT {
			err = nil
		}
	} else {
		exists = true
	}
	return
}

func getOpenedFileSize(file *os.File) (int64, error) {
	fi, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
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

func (t *TailFile) saveBookmark() error {
	pos, err := t.file.Seek(0, os.SEEK_CUR)
	if err != nil {
		return err
	}
	b := Bookmark{
		OriginalPath: t.targetAbsPath,
		WatchingPath: t.watchingFileAbsPath,
		Position:     pos,
	}
	return b.Save(t.bookmarkFilename)
}

// ReadLoop runs a loop for reading the target file. Please use this with a goroutine like:
//  go t.ReadLoop()
func (t *TailFile) ReadLoop(ctx context.Context) {
	t.mu.Lock() // synchronize with Close
	defer t.mu.Unlock()

	t.readLines()
	err = t.saveBookmark()
	if err != nil {
		t.Errors <- err
		return
	}

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
				err = t.saveBookmark()
				if err != nil {
					t.Errors <- err
					return
				}

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
				err = t.saveBookmark()
				if err != nil {
					t.Errors <- err
					return
				}

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
