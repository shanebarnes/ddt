package main

// References:
//  https://yarchive.net/comp/linux/o_direct.html

import (
	"os"
	"syscall"
)

func OpenFileRd(name string, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, os.O_RDONLY | os.O_SYNC | syscall.O_DIRECT, perm)
}

func OpenFileWr(name string, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, os.O_WRONLY | os.O_CREATE | os.O_SYNC | syscall.O_DIRECT, perm)
}