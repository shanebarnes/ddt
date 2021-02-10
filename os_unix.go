// +build darwin linux

package main

import (
	"os"
	"path/filepath"
)

var special = []string{
	os.DevNull,
	"/dev/random",
	"/dev/urandom",
	"/dev/zero",
}

func IsSpecialFile(name string) bool {
	if abs, err := filepath.Abs(name); err == nil {
		for _, f := range special {
			if abs == f {
				return true
			}
		}
	}
	return false
}