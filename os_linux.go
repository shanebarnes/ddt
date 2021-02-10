package main

// References:
//  https://yarchive.net/comp/linux/o_direct.html

import (
	"os"
	"syscall"
)

func OpenFileRd(name string, perm os.FileMode) (*os.File, error) {
	flag := os.O_RDONLY | os.O_SYNC

	// O_DIRECT may fail on some kernels and filesystems
	f, err := os.OpenFile(name, flag | syscall.O_DIRECT, perm)
	if err == syscall.EINVAL {
		f, err = os.OpenFile(name, flag, perm)
	}
	return f, err
}

func OpenFileWr(name string, perm os.FileMode) (*os.File, error) {
	// syscall.O_SYNC - wait for file data and meta data to be written to disk
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC | syscall.O_DSYNC

	// O_DIRECT may fail on some kernels and filesystems
	f, err := os.OpenFile(name,  flag | syscall.O_DIRECT, perm)
	if err == syscall.EINVAL {
		f, err = os.OpenFile(name, flag, perm)
	}
	return f, err
}