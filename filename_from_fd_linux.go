// +build linux

package tailfile

import (
	"fmt"
	"os"
)

func getFilenameFromFd(fd uintptr) (string, error) {
	n := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), uint(fd))
	return os.Readlink(n)
}
