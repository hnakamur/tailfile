// +build darwin

package tailfile

import (
	"strings"
	"syscall"
	"unsafe"
)

const pathMaxLen = 1024

func (f *fileAndStat) currentFilename() (string, error) {
	buf := make([]byte, pathMaxLen)
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, f.file.Fd(), syscall.F_GETPATH, uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return "", errno
	}
	return strings.TrimRight(string(buf), "\x00"), nil
}

func (f *fileAndStat) removed() (bool, error) {
	return f.fi.Sys().(*syscall.Stat_t).Nlink == 0, nil
}
