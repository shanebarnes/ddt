package path

import (
	"os"
	"path/filepath"
)

const (
	ExtendedLengthPathUncPrefix = `\\?\UNC`
)

func FixLongUncPath (path string) string {
	extPath := path
	volName := filepath.VolumeName(path)

	if len(volName) >= 2 && volName[1] != ':' {
		// Taken from https://github.com/golang/go/blob/master/src/os/path_windows.go
		if l := len(path); l >= 5 &&
			os.IsPathSeparator(path[0]) &&
			os.IsPathSeparator(path[1]) &&
			!os.IsPathSeparator(path[2]) && path[2] != '.' {
			// first, leading `\\` and next shouldn't be `\`. its server name.
			for n := 3; n < l-1; n++ {
				// second, next '\' shouldn't be repeated.
				if os.IsPathSeparator(path[n]) {
					n++
					// third, following something characters. its share name.
					if !os.IsPathSeparator(path[n]) {
						if path[n] == '.' {
							break
						}
						for ; n < l; n++ {
							if os.IsPathSeparator(path[n]) {
								break
							}
						}
						extPath = ExtendedLengthPathUncPrefix + path[1:]
					}
					break
				}
			}
		}
	}

	return extPath
}
