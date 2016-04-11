// +build !linux

package tailfile

import "errors"

func getFileInfo(fd uintptr) (*fileInfo, error) {
	return nil, errors.New("not implemented")
}

func getFilenameFromFd(fd uintptr) (string, error) {
	return "", errors.New("not implemented")
}
