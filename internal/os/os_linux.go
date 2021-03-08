package os

// References:
//  https://yarchive.net/comp/linux/o_direct.html

import (
	"os"
)

func OpenFileRd(name string, perm os.FileMode) (*os.File, error) {
	// Notes:
	//   Ignore O_DIRECT as it may fail on some kernels and filesystems
	return os.OpenFile(name, os.O_RDONLY /*| os.O_SYNC*/, perm)
}

func OpenFileWr(name string, perm os.FileMode) (*os.File, error) {
	// Notes:
	//   Ignore O_DIRECT as it may fail on some kernels and filesystems
	//   Use syscall.O_DSYNC instead of syscall.O_SYNC for more optimistic results
	return os.OpenFile(name,  os.O_WRONLY | os.O_CREATE | os.O_TRUNC /*| syscall.O_DSYNC*/, perm)
}
