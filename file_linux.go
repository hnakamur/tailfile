// +build linux

package tailfile

import (
	"fmt"
	"os"
	"syscall"
)

func getFileInfo(fd uintptr) (*fileInfo, error) {
	var st syscall.Stat_t
	err := syscall.Fstat(int(fd), &st)
	if err != nil {
		return nil, err
	}
	return &fileInfo{
		Size:    st.Size,
		Removed: st.Nlink == 0,
	}, nil
}

func getFilenameFromFd(fd uintptr) (string, error) {
	n := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), uint(fd))
	return os.Readlink(n)
}
