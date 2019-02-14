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

type ddInfo struct {
	RdBytes int64
	RdDur   time.Duration
	WrBytes int64
	WrDur   time.Duration
}

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
	res := make(chan *ddInfo, *threads)

	var reader, writer *os.File
	var err error
	if reader, err = os.OpenFile(*fileRd, os.O_RDONLY, 0755); err != nil {
		panic("reader")
	}
	defer reader.Close()

	if writer, err = os.OpenFile(*fileWr, os.O_CREATE | os.O_WRONLY, 0755); err != nil {
		panic("writer")
	}
	defer writer.Close()
	for i := 0; i < *threads; i++ {
		go worker(i+1, reader, writer, *fileRd, *fileWr, *blockSize, req, res)
	}

	var mutex = &sync.Mutex{}
	var wg sync.WaitGroup
	wg.Add(1)
	blocks := int64(0)
	sum := ddInfo{}

	go func() {
		defer wg.Done()
		for i := int64(0); i < *count; i++ {
			ddi := <-res

			if ddi.WrBytes >= 0 {
				mutex.Lock()
				blocks = blocks + 1
				sum.RdBytes = sum.RdBytes + ddi.RdBytes
				sum.RdDur = sum.RdDur + ddi.RdDur
				sum.WrBytes = sum.WrBytes + ddi.WrBytes
				sum.WrDur = sum.WrDur + ddi.WrDur
				mutex.Unlock()
			}
		}
		close(res)
		if writer, err := os.OpenFile(*fileWr, os.O_WRONLY|os.O_CREATE, 0755); err == nil {
			defer writer.Close()
			writer.Truncate(fileSize)
			//writer.Sync()
		}
	} ()

	start := time.Now()
	ticker := time.NewTicker(time.Millisecond * 1000)
	go func() {
		for range ticker.C {
			mutex.Lock()
			tmpBlocks := blocks
			tmpSum := sum
			mutex.Unlock()

			printStats(&tmpSum, tmpBlocks, time.Since(start))
		}
	} ()

	for i := int64(0); i < *count; i++ {
		req <- i
	}
	close(req)
	wg.Wait()
	stop := time.Now()
	printStats(&sum, blocks, stop.Sub(start))
}

func printStats(sum *ddInfo, blocks int64, duration time.Duration) {
	rate := int64(0)
	sec := int64(time.Duration(duration) / time.Second)
	if sec > 0 {
		rate = sum.WrBytes / sec
	}

	avgRdTime := time.Duration(0)
	avgWrTime := time.Duration(0)
	if blocks > 0 {
		avgRdTime = sum.RdDur / time.Duration(blocks)
		avgWrTime = sum.WrDur / time.Duration(blocks)
	}

	fmt.Fprintf(os.Stdout,
		"Total: time=%s blocks=%d avg read/write=%s/%s size=%s rate=%s/sec\n",
		duration,
		blocks,
		avgRdTime,
		avgWrTime,
		humanize.Bytes(uint64(sum.WrBytes)),
		humanize.Bytes(uint64(rate)))
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

func worker(id int, reader, writer *os.File, fileRd, fileWr string, blockSize int64, req <-chan int64, res chan<- *ddInfo) {
	var err error

	buf := make([]byte, blockSize)
	n := 0
	for num := range req {
		ddi := ddInfo{}
		timeA := time.Now()
		if n, err = reader.ReadAt(buf, int64(num*blockSize)); err == nil {
			timeB := time.Now()
			ddi.RdBytes = int64(n)
			n, err = writer.WriteAt(buf, int64(num*blockSize))
			ddi.WrDur = time.Since(timeB)
			ddi.WrBytes = int64(n)
			ddi.RdDur = timeB.Sub(timeA)
		}
		res <- &ddi
	}
}
