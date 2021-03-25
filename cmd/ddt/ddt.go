package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/shanebarnes/ddt/internal/ddt"
	ddtpath "github.com/shanebarnes/ddt/internal/path"
	"github.com/shanebarnes/goto/tokenbucket"
	"github.com/shanebarnes/goto/units"
)

type ddInfo struct {
	RdBytes int64
	RdDur   time.Duration
	WrBytes int64
	WrDur   time.Duration
}

type flagStringSlice []string

// Implement the flag.Value interface: https://golang.org/src/flag/flag.go?s=7450:7510#L281
func (f *flagStringSlice) String() string {
	return fmt.Sprintf("%s", *f)
}

func (f *flagStringSlice) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func panicIf(msg string, err error, ignore ...error) {
	if err != nil {
		for _, e := range ignore {
			if err == e {
				return
			}
		}
		panic(msg + ": " + err.Error())
	}
}

func main() {
	var inputPatterns flagStringSlice

	blockSizeStr := flag.String("bs", "512", "Set both input and output block size to n bytes")
	count := flag.Int64("count", -1, "Copy only n input blocks")
	fileRd := flag.String("if", "", "Read input from file instead of the standard input")
	fileWr := flag.String("of", "", "Write output to file instead of the standard output")
	flag.Var(&inputPatterns, "ip", "Create input blocks from input patterns")
	rateBpsStr := flag.String("rate", "0", "Copy rate limit in bits per second")
	share := flag.Bool("share", true, "Share a single read and write file descriptor between threads")
	skip := flag.Int64("skip", 0, "Skip n blocks from beginning of the input before copying")
	threads := flag.Int("threads", 1, "Number of copy threads")

	flag.Parse()

	blockSize := int64(0)
	if f, err := units.ToNumber(*blockSizeStr); err == nil {
		blockSize = int64(f)
	}

	rateBps := uint64(0)
	if f, err := units.ToNumber(*rateBpsStr); err == nil {
		rateBps = uint64(f)
	}

	validateFlags(blockSize, skip, count, &inputPatterns, *fileRd, *fileWr, *threads)

	req := make(chan int64, *threads)
	res := make(chan *ddInfo, *threads)

	var fpRd string
	var err error
	fpRd, err = filepath.Abs(*fileRd)
	fpRd = ddtpath.FixLongUncPath(fpRd)
	panicIf("fileRd", err)

	var fpWr string
	fpWr, err = filepath.Abs(*fileWr)
	fpWr = ddtpath.FixLongUncPath(fpWr)
	panicIf("fileWr", err)
	err = os.MkdirAll(filepath.Dir(fpWr), ddt.FilePerm)
	panicIf("", err)

	copier := ddt.Create(blockSize, fpRd, fpWr, inputPatterns)

	var wg sync.WaitGroup
	for i := 0; i < *threads; i++ {
		err = copier.Open(i, *share)
		panicIf("copier "+strconv.Itoa(i), err)
		wg.Add(1)
		go copyWorker(i, copier, *skip, rateBps/(8*uint64(*threads)), &wg, req, res)
	}

	var mutex = &sync.Mutex{}
	blocks := int64(0)
	sum := ddInfo{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := int64(0); i < *count; i++ {
			ddi := <-res

			if ddi.WrBytes > 0 {
				mutex.Lock()
				blocks++
				sum.RdBytes = sum.RdBytes + ddi.RdBytes
				sum.RdDur = sum.RdDur + ddi.RdDur
				sum.WrBytes = sum.WrBytes + ddi.WrBytes
				sum.WrDur = sum.WrDur + ddi.WrDur
				mutex.Unlock()
			}
		}
		close(res)
		//writer.Sync()
	}()

	start := time.Now()
	ticker := time.NewTicker(time.Millisecond * 1000)
	it := 0
	go func() {
		for range ticker.C {
			mutex.Lock()
			tmpBlocks := blocks
			tmpSum := sum
			mutex.Unlock()

			printStats(it, &tmpSum, tmpBlocks, *count, time.Since(start))
			it = it + 1
		}
	}()

	for i := int64(0); i < *count; i++ {
		req <- i
	}
	close(req)
	wg.Wait()
	stop := time.Now()
	printStats(it, &sum, blocks, blocks, stop.Sub(start))
}

func printStats(it int, sum *ddInfo, blocksComplete, blocksTotal int64, duration time.Duration) {
	rate := int64(0)
	usec := int64(duration / time.Microsecond)
	if usec > 0 {
		rate = sum.WrBytes * int64(time.Second/time.Microsecond) / usec
	}

	avgRdTime := time.Duration(0)
	avgWrTime := time.Duration(0)
	progress := float64(0)
	if blocksTotal > 0 {
		progress = float64(blocksComplete) / float64(blocksTotal) * 100
	}

	if blocksComplete > 0 {
		avgRdTime = sum.RdDur / time.Duration(blocksComplete)
		avgWrTime = sum.WrDur / time.Duration(blocksComplete)
	}

	if it%10 == 0 {
		fmt.Fprintf(os.Stdout,
			"%17s %9s %9s %9s %9s %9s %12s\n",
			"ELAPSED TIME",
			"BLOCKS",
			"PROGRESS",
			"AVG READ",
			"AVG WRITE",
			"SIZE",
			"RATE")
	}

	fmt.Fprintf(os.Stdout,
		"%17s %9s %9s %9s %9s %9s %12s\n",
		units.ToTimeString(float64(duration)/float64(time.Second)),
		units.ToMetricString(float64(blocksComplete), 3, "", ""),
		units.ToMetricString(progress, 3, "", "%"),
		units.ToMetricString(avgRdTime.Seconds(), 3, "", "s"),
		units.ToMetricString(avgWrTime.Seconds(), 3, "", "s"),
		units.ToMetricString(float64(sum.WrBytes), 3, "", "B"),
		units.ToMetricString(float64(rate*8), 3, "", "bps"))
}

func validateFlags(blockSize int64, skip, count *int64, patternRd *flagStringSlice, fileRd, fileWr string, threads int) {
	if len(os.Args) < 2 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if blockSize <= 0 {
		panic("bs <= 0")
	}

	if *skip < 0 {
		*skip = 0
	}

	if *count < 0 {
		if len(*patternRd) == 0 {
			fi, err := os.Stat(fileRd)
			panicIf("if file failure", err)
			*count = fi.Size() / blockSize
			if fi.Size()%blockSize != 0 {
				*count++
			}
		} else {
			*count = 1
		}
	}

	if len(*patternRd) > 0 {
		for i, pattern := range *patternRd {
			if len(pattern) == 0 {
				panic("ip[" + strconv.FormatInt(int64(i), 10) + "] == nil")
			}
		}
	}

	if len(*patternRd) == 0 && len(fileRd) == 0 {
		panic("if == nil")
	}

	if len(fileWr) == 0 {
		panic("of == nil")
	}

	if threads < 1 {
		panic("threads < 1")
	}
}

func copyWorker(id int, copier *ddt.Copier, skip int64, rate uint64, wg *sync.WaitGroup, req <-chan int64, res chan<- *ddInfo) {
	defer wg.Done()
	defer copier.ReadClose(id)
	defer copier.WriteClose(id)

	var err error
	buf := make([]byte, copier.BlockSize())
	n := 0
	tb := tokenbucket.New(rate, uint64(copier.BlockSize())*1)
	eof := false

	for num := range req {
		ddi := ddInfo{}
		if !eof {
			timeA := time.Now()
			tb.Remove(uint64(copier.BlockSize()))
			n, err = copier.ReadAt(id, buf, (num+skip)*copier.BlockSize())
			eof = (err == io.EOF)
			panicIf(fmt.Sprintf("Reader %d failed", id), err, io.EOF)

			timeB := time.Now()
			ddi.RdBytes = int64(n)
			n, err = copier.WriteAt(id, buf[:n], num*copier.BlockSize())
			panicIf(fmt.Sprintf("Writer %d failed", id), err)

			ddi.WrDur = time.Since(timeB)
			ddi.WrBytes = int64(n)
			ddi.RdDur = timeB.Sub(timeA)
		}
		res <- &ddi
	}
}
