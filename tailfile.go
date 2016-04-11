package tailfile

import (
	"bufio"
	"io"
	"os"
	"strings"

	"golang.org/x/net/context"
)

type TailFile struct {
	filename string
	file     *os.File
	reader   *bufio.Reader
	logger   Logger

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
func NewTailFile(filename string, logger Logger) (*TailFile, error) {
	return &TailFile{
		filename: filename,
		logger:   logger,
		Lines:    make(chan string),
		Errors:   make(chan error),
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
		select {
		case <-ctx.Done():
			return
		default: //TODO: Use timer to avoid the busy loop
		}
	}
}

func (t *TailFile) readLine() error {
	line, err := t.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	t.seenEOF = (err == io.EOF)
	if !t.seenEOF {
		line = strings.TrimRight(line, "\n")
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
	err := t.readLine()
	if err != nil {
		t.Errors <- err
		return nil
	}
	return stateReading
}
