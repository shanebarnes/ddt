package os

import (
	"os"
)

func OpenFileRd(name string, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, os.O_RDONLY /*| os.O_SYNC | syscall.F_NOCACHE*/, perm)
}

func OpenFileWr(name string, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, os.O_WRONLY | os.O_CREATE | os.O_TRUNC /*| os.O_SYNC | syscall.F_NOCACHE*/, perm)
}
