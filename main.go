package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
)

func main() {
	blockSize  := flag.Int("bs", 4096, "Set both input and output block size to n bytes")
	count := flag.Int("count", 1, "Copy only n input blocks")
	fileRd  := flag.String("if", "", "Read input from file instead of the standard input")
	fileWr := flag.String("of", "", "Write output to file instead of the standard output")
	threads := flag.Int("n", runtime.NumCPU(), "")

	flag.Parse()
	validateFlags(*count, *fileRd, *fileWr, *threads)

	fileSize := int64((*count) * (*blockSize))
	req := make(chan int, *threads)
	res := make(chan int, *threads)

	for i := 0; i < *threads; i++ {
		go worker(i+1, *fileRd, *fileWr, *blockSize, req, res)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < *count; i++ {
			<-res
		}
		close(res)
		if writer, err := os.OpenFile(*fileWr, os.O_WRONLY|os.O_CREATE, 0755); err == nil {
			defer writer.Close()
			writer.Truncate(fileSize)
			writer.Sync()
		}
	} ()

	tStart := time.Now()
	for i := 0; i < *count; i++ {
		req <- i
	}
	close(req)
	wg.Wait()
	tStop := time.Now()

	duration := tStop.Sub(tStart)
	rate := int64(0)
	if duration > 0 {
		rate = fileSize * int64(time.Second) / int64(time.Duration(duration))
	}

	fmt.Fprintf(os.Stdout, "Total: time=%s size=%s B rate=%s/sec\n", duration, humanize.Comma(fileSize), humanize.Bytes(uint64(rate)))
}

func validateFlags(count int, fileRd, fileWr string, threads int) {
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

func worker(id int, fileRd, fileWr string, blockSize int, req <-chan int, res chan<- int) {
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
		res <- n
	}
}
