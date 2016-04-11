package tailfile

import (
	"bufio"
	"io"
	"os"
	"syscall"
	"time"

	"golang.org/x/net/context"
)

type TailFile struct {
	filename string
	file     *os.File
	reader   *bufio.Reader
	logger   Logger

	pollingInterval time.Duration
	pollingTimer    *time.Timer
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
func NewTailFile(filename string, pollingInterval time.Duration, logger Logger) (*TailFile, error) {
	timer := time.NewTimer(0)
	timer.Stop()
	return &TailFile{
		filename:        filename,
		logger:          logger,
		pollingInterval: pollingInterval,
		pollingTimer:    timer,
		Lines:           make(chan string),
		Errors:          make(chan error),
	}, nil
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
	defer t.closeFile()
	for s := stateOpening; s != nil; s = s(ctx, t) {
		if t.file == nil || t.seenEOF {
			t.pollingTimer.Reset(t.pollingInterval)
		} else {
			t.pollingTimer.Stop()
		}

		select {
		case <-ctx.Done():
			return
		case <-t.pollingTimer.C:
			// do nothing
		default:
			// do nothing
		}
	}
}

func (t *TailFile) readLine() error {
	line, err := t.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	t.seenEOF = (err == io.EOF)
	if !t.seenEOF || (t.seenEOF && line != "") {
		t.Lines <- line
	}
	return nil
}

type stateFn func(context.Context, *TailFile) stateFn

func stateOpening(ctx context.Context, t *TailFile) stateFn {
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

func stateReading(ctx context.Context, t *TailFile) stateFn {
	var st syscall.Stat_t
	err := syscall.Fstat(int(t.file.Fd()), &st)
	if err != nil {
		t.Errors <- err
		return nil
	}
	if st.Size < t.fileSize {
		return stateShrinked
	} else if st.Nlink == 0 {
		return stateDeleted
	}
	t.fileSize = st.Size

	filename, err := getFilenameFromFd(t.file.Fd())
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

func stateShrinked(ctx context.Context, t *TailFile) stateFn {
	t.closeFile()
	return stateOpening
}

func stateDeleted(ctx context.Context, t *TailFile) stateFn {
	t.closeFile()
	return stateOpening
}

func stateRenamed(ctx context.Context, t *TailFile) stateFn {
	return stateReadingOldFileBeforeRecreation
}

func stateReadingOldFileBeforeRecreation(ctx context.Context, t *TailFile) stateFn {
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
		return stateReadingOldFileAfterRecreation
	} else {
		return stateReadingOldFileBeforeRecreation
	}
}

func stateReadingOldFileAfterRecreation(ctx context.Context, t *TailFile) stateFn {
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
		return stateReading
	}

	return stateReadingOldFileAfterRecreation
}
