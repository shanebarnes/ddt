package main

import (
	"os"
)

const (
	FILE_FLAG_NO_BUFFERING  = 0x20000000
	FILE_FLAG_WRITE_THROUGH = 0x80000000
)

func OpenFileRd(name string, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, os.O_RDONLY | os.O_SYNC | FILE_FLAG_NO_BUFFERING, perm)
}

func OpenFileWr(name string, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, os.O_WRONLY | os.O_CREATE | os.O_SYNC | os.O_TRUNC | FILE_FLAG_NO_BUFFERING | FILE_FLAG_WRITE_THROUGH, perm)
}