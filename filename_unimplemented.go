// +build !linux

package tailfile

import (
	"errors"
	"os"
)

func getFilename(file *os.File) (string, error) {
	return "", errors.New("not implemented")
}
