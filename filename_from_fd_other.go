// +build !linux

package tailfile

import "errors"

func getFilenameFromFd(fd uintptr) (string, error) {
	return "", errors.New("not implemented")
}
