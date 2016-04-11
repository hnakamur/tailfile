package tailfile

import (
	"bufio"
	"io"
	"os"
	"time"

	"golang.org/x/net/context"
)

type TailFile struct {
	filename string
	file     *os.File
	reader   *bufio.Reader
	logger   Logger

	pollingInterval time.Duration
	recreatedFile   *os.File
	renamedFilename string
	seenEOF         bool
	fileSize        int64

	Lines  chan string // The channel to receiving log lines while reading the log file
	Errors chan error  // The channel to receiving an error while reading the log file
}

// NewTailFile starts watching the directory for the target file and opens the target file if it exists.
// The target file may not exist at the first. In that case, TailFile opens the target file as soon as
// the target file is created and written in the later time.
func NewTailFile(filename string, pollingInterval time.Duration, logger Logger) *TailFile {
	return &TailFile{
		filename:        filename,
		logger:          logger,
		pollingInterval: pollingInterval,
		Lines:           make(chan string),
		Errors:          make(chan error),
	}
}

func (t *TailFile) closeFile() {
	if t.file != nil {
		err := t.file.Close()
		if err != nil {
			if t.logger != nil {
				t.logger.Log(err)
			}
		}

		if t.logger != nil {
			t.logger.Log("closed the file")
		}
		t.file = nil
	}
	t.reader = nil
	t.recreatedFile = nil
	t.renamedFilename = ""
	t.seenEOF = false
	t.fileSize = 0
}

// Run a loop for reading the target file.
func (t *TailFile) Run(ctx context.Context) {
	if t.logger != nil {
		t.logger.Log("Run start")
	}
	defer t.closeFile()
	for s := stateOpening; s != nil; s = s(t) {
		select {
		case <-ctx.Done():
			return
		default:
			// do nothing
		}
	}
}

func (t *TailFile) readLine() error {
	if t.seenEOF {
		time.Sleep(t.pollingInterval)
	}

	line, err := t.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	t.seenEOF = (err == io.EOF)
	if line != "" {
		t.Lines <- line
	}
	return nil
}

type stateFn func(*TailFile) stateFn

func stateOpening(t *TailFile) stateFn {
	time.Sleep(t.pollingInterval)

	file, err := os.Open(t.filename)
	if err != nil && !os.IsNotExist(err) {
		t.Errors <- err
		return nil
	}

	if err == nil {
		t.file = file
		t.reader = bufio.NewReader(file)
		return stateReading
	} else {
		return stateOpening
	}
}

type fileInfo struct {
	Size    int64
	Removed bool
}

func stateReading(t *TailFile) stateFn {
	fi, err := getFileInfo(t.file)
	if err != nil {
		t.Errors <- err
		return nil
	}
	if fi.Size < t.fileSize {
		if t.logger != nil {
			t.logger.Log("transiion to stateShrinked")
		}
		return stateShrinked
	} else if fi.Removed {
		if t.logger != nil {
			t.logger.Log("transiion to stateRemoved")
		}
		return stateRemoved
	}
	t.fileSize = fi.Size

	filename, err := getFilename(t.file)
	if err != nil {
		t.Errors <- err
		return nil
	}
	if filename != t.filename {
		t.renamedFilename = filename
		return stateRenamed
	}

	err = t.readLine()
	if err != nil {
		t.Errors <- err
		return nil
	}
	return stateReading
}

func stateShrinked(t *TailFile) stateFn {
	t.closeFile()
	return stateOpening
}

func stateRemoved(t *TailFile) stateFn {
	return stateReadingOldFileBeforeRecreation
}

func stateRenamed(t *TailFile) stateFn {
	return stateReadingOldFileBeforeRecreation
}

func stateReadingOldFileBeforeRecreation(t *TailFile) stateFn {
	file, err := os.Open(t.filename)
	if err != nil && !os.IsNotExist(err) {
		t.Errors <- err
		return nil
	}
	if err == nil {
		t.recreatedFile = file
	}

	err = t.readLine()
	if err != nil {
		t.Errors <- err
		return nil
	}

	if t.recreatedFile != nil {
		if t.logger != nil {
			t.logger.Log("transition to stateReadingOldFileAfterRecreation")
		}
		return stateReadingOldFileAfterRecreation
	} else {
		return stateReadingOldFileBeforeRecreation
	}
}

func stateReadingOldFileAfterRecreation(t *TailFile) stateFn {
	err := t.readLine()
	if err != nil {
		t.Errors <- err
		return nil
	}

	if t.seenEOF {
		// The file of the target path has been recreated
		// and we had read until the end of the old file.
		// Close the old file and start reading the new file.
		file := t.recreatedFile
		t.closeFile()
		t.file = file
		t.reader = bufio.NewReader(file)
		if t.logger != nil {
			t.logger.Log("transition to stateReading")
		}
		return stateReading
	}

	return stateReadingOldFileAfterRecreation
}
