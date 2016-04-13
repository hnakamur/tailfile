// +build linux

package tailfile

import (
	"fmt"
	"os"
)

func getFilename(file *os.File) (string, error) {
	n := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), uint(file.Fd()))
	return os.Readlink(n)
}
