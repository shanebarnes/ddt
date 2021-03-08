package ddt

import (
	"os"
	"strings"
	"sync"
	"syscall"

	ddtos "github.com/shanebarnes/ddt/internal/os"
)

const FilePerm = 0755

type Copier struct {
	blocks    [][]byte
	blockSize int64
	epRd      Endpoint
	epWr      Endpoint
	open      bool
	patterns  []string
}

type Endpoint struct {
	file []*os.File
	mu   *sync.Mutex
	name string
	ref  int
}

func Create(blockSize int64, fnameRd, fnameWr string, patterns []string) *Copier {
	 return &Copier{
		blockSize: blockSize,
		epRd: Endpoint {
			mu: &sync.Mutex{},
			name: fnameRd,
		},
		epWr: Endpoint {
			mu: &sync.Mutex{},
			name: fnameWr,
		},
		patterns: patterns,
	}
}

func (c *Copier) BlockSize() int64 {
	return c.blockSize
}

func (c *Copier) Open(idx int, share bool) error {
	var err error
	var fileRd, fileWr *os.File

	if share && len(c.epRd.file) == 1 && len(c.epWr.file) == 1 {
		c.epRd.mu.Lock()
		c.epRd.ref++
		c.epRd.mu.Unlock()
		c.epWr.mu.Lock()
		c.epWr.ref++
		c.epWr.mu.Unlock()
	} else if idx == len(c.epRd.file) && idx == len(c.epWr.file) {
		if len(c.patterns) > 0 {
			for _, pattern := range c.patterns {
				count := c.blockSize/int64(len(pattern)) + 1
				block := []byte(strings.Repeat(pattern, int(count)))
				c.blocks = append(c.blocks, block[0:c.blockSize])
			}
		} else {
			fileRd, err = ddtos.OpenFileRd(c.epRd.name, FilePerm)
		}

		if err != nil {
			// Do nothing
		} else if fileWr, err = ddtos.OpenFileWr(c.epWr.name, FilePerm); err != nil {
			fileRd.Close()
		} else {
			c.epRd.mu.Lock()
			c.epRd.ref++
			c.epRd.mu.Unlock()
			c.epWr.mu.Lock()
			c.epWr.ref++
			c.epWr.mu.Unlock()
			c.epRd.file = append(c.epRd.file, fileRd)
			c.epWr.file = append(c.epWr.file, fileWr)
		}
	} else {
		err = syscall.EINVAL
	}

	return err
}

func (c *Copier) ReadAt(idx int, buf []byte, off int64) (int, error) {
	if len(c.patterns) > 0 {
		blockIndex := (off / c.blockSize) % int64(len(c.blocks))
		patternSize := len(c.patterns[blockIndex])
		n := 0

		for n < len(buf) {
			r := len(buf) - n
			if r > patternSize {
				r = patternSize
			}
			copy(buf[n:n+r], c.blocks[blockIndex][0:r])
			n = n + r
		}
		return n, nil
	} else if len(c.epRd.file) == 0 {
		return -1, syscall.EBADF
	} else if idx < len(c.epRd.file) {
		c.epRd.file[idx].ReadAt(buf, off)
	}
	return c.epRd.file[0].ReadAt(buf, off) // share
}

func (c *Copier) ReadClose(idx int) error {
	var err error
	if len(c.epRd.file) == 0 && len(c.patterns) == 0 {
		err = syscall.EBADF
	} else if len(c.epRd.file) == 1 { // share
		c.epRd.mu.Lock()
		if c.epRd.ref > 0 {
			c.epRd.ref--
		}
		ref := c.epRd.ref
		c.epRd.mu.Unlock()
		if ref == 0 {
			err = c.epRd.file[0].Close()
		}
	} else if idx < len(c.epRd.file) {
		err = c.epRd.file[idx].Close()
	}
	return err
}

func (c *Copier) WriteAt(idx int, buf []byte, off int64) (int, error) {
	if len(c.epWr.file) == 0 {
		return -1, syscall.EBADF
	} else if idx < len(c.epWr.file) {
		return c.epWr.file[idx].WriteAt(buf, off)
	}
	return c.epWr.file[0].WriteAt(buf, off) // share
}

func (c *Copier) WriteClose(idx int) error {
	var err error
	if len(c.epWr.file) == 0 {
		err = syscall.EBADF
	} else if len(c.epWr.file) == 1 { // share
		c.epWr.mu.Lock()
		if c.epWr.ref > 0 {
			c.epWr.ref--
		}
		ref := c.epWr.ref
		c.epWr.mu.Unlock()
		if ref == 0 {
			err = c.epWr.file[0].Close()
		}
	} else if idx < len(c.epWr.file) {
		err = c.epWr.file[idx].Close()
	}
	return err
}
