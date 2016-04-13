// +build !linux
// +build !darwin

package tailfile

import "errors"

func (f *fileAndStat) currentFilename() (string, error) {
	return "", errors.New("not implemented")
}

func (f *fileAndStat) removed() (bool, error) {
	return "", errors.New("not implemented")
}
