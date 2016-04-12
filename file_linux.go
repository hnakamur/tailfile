// +build linux

package tailfile

import (
	"fmt"
	"os"
	"syscall"
)

func getFileInfo(file *os.File) (*fileInfo, error) {
	var st syscall.Stat_t
	err := syscall.Fstat(int(file.Fd()), &st)
	if err != nil {
		return nil, err
	}
	return &fileInfo{
		Size:    st.Size,
		Removed: st.Nlink == 0,
	}, nil
}

func getFilename(file *os.File) (string, error) {
	n := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), uint(file.Fd()))
	return os.Readlink(n)
}
