package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
)

func main() {
	blockSize  := flag.Int64("bs", 4096, "Set both input and output block size to n bytes")
	count := flag.Int64("count", 1, "Copy only n input blocks")
	fileRd  := flag.String("if", "", "Read input from file instead of the standard input")
	fileWr := flag.String("of", "", "Write output to file instead of the standard output")
	threads := flag.Int("n", runtime.NumCPU(), "")

	flag.Parse()
	validateFlags(*count, *fileRd, *fileWr, *threads)

	fileSize := int64((*count) * (*blockSize))
	req := make(chan int64, *threads)
	res := make(chan int64, *threads)

	for i := 0; i < *threads; i++ {
		go worker(i+1, *fileRd, *fileWr, *blockSize, req, res)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	blocks := int64(0)
	go func() {
		defer wg.Done()
		for i := int64(0); i < *count; i++ {
			n := <-res

			if n >= 0 {
				atomic.StoreInt64(&blocks, atomic.LoadInt64(&blocks) + 1)
			}
		}
		close(res)
		if writer, err := os.OpenFile(*fileWr, os.O_WRONLY|os.O_CREATE, 0755); err == nil {
			defer writer.Close()
			writer.Truncate(fileSize)
			writer.Sync()
		}
	} ()

	start := time.Now()
	ticker := time.NewTicker(time.Millisecond * 1000)
	go func() {
		for range ticker.C {
			printStats(atomic.LoadInt64(&blocks) * (*blockSize), time.Since(start))
		}
	} ()

	for i := int64(0); i < *count; i++ {
		req <- i
	}
	close(req)
	wg.Wait()
	stop := time.Now()
	printStats(fileSize, stop.Sub(start))
}

func printStats(bytes int64, duration time.Duration) {
	rate := int64(0)
	if duration > 0 {
		rate = bytes * int64(time.Second) / int64(time.Duration(duration))
	}

	fmt.Fprintf(os.Stdout, "Total: time=%s size=%s rate=%s/sec\n", duration, humanize.Bytes(uint64(bytes)), humanize.Bytes(uint64(rate)))
}

func validateFlags(count int64, fileRd, fileWr string, threads int) {
	if count < 1 {
		panic("count < 1")
	}

	if len(fileRd) == 0 {
		panic("if == nil")
	}

	if len(fileWr) == 0 {
		panic("of == nil")
	}

	if threads < 1 {
		panic("threads < 1")
	}
}

func worker(id int, fileRd, fileWr string, blockSize int64, req <-chan int64, res chan<- int64) {
	var err error
	var reader, writer *os.File

	if reader, err = os.OpenFile(fileRd, os.O_RDONLY, 0755); err != nil {
		panic("reader")
	}
	defer reader.Close()

	if writer, err = os.OpenFile(fileWr, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
		panic("writer")
	}
	defer writer.Close()

	buf := make([]byte, blockSize)
	n := 0
	for num := range req {
		if n, err = reader.ReadAt(buf, int64(num*blockSize)); err == nil {
			n, err = writer.WriteAt(buf, int64(num*blockSize))
		}
		//fmt.Println("Worker #", id, "read block #", num, "of size", n)
		res <- int64(n)
	}
}
