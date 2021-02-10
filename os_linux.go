package main

// References:
//  https://yarchive.net/comp/linux/o_direct.html

import (
	"os"
	"syscall"
)

func OpenFileRd(name string, perm os.FileMode) (*os.File, error) {
	flag := os.O_RDONLY | os.O_SYNC
	if !IsSpecialFile(name) {
		flag |= syscall.O_DIRECT
	}
	return os.OpenFile(name, flag, perm)
}

func OpenFileWr(name string, perm os.FileMode) (*os.File, error) {
	flag := os.O_WRONLY | os.O_CREATE | os.O_SYNC
	if !IsSpecialFile(name) {
		flag |= syscall.O_DIRECT
	}
	return os.OpenFile(name,  flag, perm)
}