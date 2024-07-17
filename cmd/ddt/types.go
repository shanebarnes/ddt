package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/shanebarnes/ddt"
)

type ddInfo struct {
	read  ddt.OpInfo
	write ddt.OpInfo
}

func exitIf(msg string, err error, ignoreErrs ...error) {
	if err != nil {
		for _, errIgnore := range ignoreErrs {
			if errors.Is(err, errIgnore) {
				return
			}
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
		os.Exit(1)
	}
}
