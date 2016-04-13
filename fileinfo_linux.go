// +build linux

package tailfile

import (
	"fmt"
	"os"
	"syscall"
)

func (f *fileAndStat) currentFilename() (string, error) {
	n := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), uint(f.file.Fd()))
	return os.Readlink(n)
}

func (f *fileAndStat) removed() (bool, error) {
	return f.fi.Sys().(*syscall.Stat_t).Nlink == 0, nil
}
