// +build linux

package tailfile

import (
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
