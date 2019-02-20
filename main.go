package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shanebarnes/goto/tokenbucket"
	"github.com/shanebarnes/goto/units"
)

type ddInfo struct {
	RdBytes int64
	RdDur   time.Duration
	WrBytes int64
	WrDur   time.Duration
}

func main() {
	blockSizeStr := flag.String("bs", "4Ki", "Set both input and output block size to n bytes")
	count := flag.Int64("count", 1, "Copy only n input blocks")
	fileRd  := flag.String("if", "", "Read input from file instead of the standard input")
	fileWr := flag.String("of", "", "Write output to file instead of the standard output")
	rateBpsStr := flag.String("rate", "0", "Read rate limit in bits per second")
	threads := flag.Int("threads", runtime.NumCPU(), "")

	flag.Parse()
	validateFlags(*count, *fileRd, *fileWr, *threads)

	blockSize := int64(0)
	if f, err := units.ToNumber(*blockSizeStr); err == nil {
		blockSize = int64(f)
	}

	rateBps := uint64(0)
	if f, err := units.ToNumber(*rateBpsStr); err == nil {
		rateBps = uint64(f)
	}

	fileSize := int64((*count) * (blockSize))
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
		go worker(i+1, reader, writer, *fileRd, *fileWr, blockSize, rateBps/(8*uint64(*threads)), req, res)
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
	it := 0
	go func() {
		for range ticker.C {
			mutex.Lock()
			tmpBlocks := blocks
			tmpSum := sum
			mutex.Unlock()

			printStats(it, &tmpSum, tmpBlocks, time.Since(start))
			it = it + 1
		}
	} ()

	for i := int64(0); i < *count; i++ {
		req <- i
	}
	close(req)
	wg.Wait()
	stop := time.Now()
	printStats(it, &sum, blocks, stop.Sub(start))
}

func printStats(it int, sum *ddInfo, blocks int64, duration time.Duration) {
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

	if it % 10 == 0 {
		fmt.Fprintf(os.Stdout,
			"%17s %7s %9s %9s %9s %9s\n",
			"ELAPSED TIME",
			"BLOCKS",
			"AVG READ",
			"AVG WRITE",
			"SIZE",
			"RATE")
	}

	fmt.Fprintf(os.Stdout,
		"%17s %7s %9s %9s %9s %9s\n",
		units.ToTimeString(float64(duration)/float64(time.Second)),
		units.ToMetricString(float64(blocks), 3, "", ""),
		units.ToMetricString(avgRdTime.Seconds(), 3, "", "s"),
		units.ToMetricString(avgWrTime.Seconds(), 3, "", "s"),
		units.ToMetricString(float64(sum.WrBytes), 3, "", "B"),
		units.ToMetricString(float64(rate * 8), 3, "", "bps"))
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

func worker(id int, reader, writer *os.File, fileRd, fileWr string, blockSize int64, rate uint64, req <-chan int64, res chan<- *ddInfo) {
	var err error

	buf := make([]byte, blockSize)
	n := 0
	tb := tokenbucket.New(rate, uint64(blockSize) * 1)
	for num := range req {
		ddi := ddInfo{}
		timeA := time.Now()
		tb.Remove(uint64(blockSize))
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
