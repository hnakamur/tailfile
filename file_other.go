// +build !linux

package tailfile

import (
	"errors"
	"os"
)

func getFileInfo(file *os.File) (*fileInfo, error) {
	return nil, errors.New("not implemented")
}

func getFilename(file *os.File) (string, error) {
	return "", errors.New("not implemented")
}
