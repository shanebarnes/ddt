package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shanebarnes/goto/tokenbucket"
	"github.com/shanebarnes/goto/units"
)

const (
	filePerm = 0755
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

type ddCopier struct {
	blocks    [][]byte
	blockSize int64
	fileRd    []*os.File
	fileWr    []*os.File
	muRd      sync.Mutex
	muWr      sync.Mutex
	nameRd    string
	nameWr    string
	open      bool
	patterns  []string
	refRd     int
	refWr     int
}

func (ddc *ddCopier) Open(idx int, share bool) error {
	var err error
	var fileRd, fileWr *os.File

	if share && len(ddc.fileRd) == 1 && len(ddc.fileWr) == 1 {
		ddc.muRd.Lock()
		ddc.refRd++
		ddc.muRd.Unlock()
		ddc.muWr.Lock()
		ddc.refWr++
		ddc.muWr.Unlock()
	} else if idx == len(ddc.fileRd) && idx == len(ddc.fileWr) {
		if len(ddc.patterns) > 0 {
			for _, pattern := range ddc.patterns {
				count := ddc.blockSize/int64(len(pattern)) + 1
				block := []byte(strings.Repeat(pattern, int(count)))
				ddc.blocks = append(ddc.blocks, block[0:ddc.blockSize])
			}
		} else {
			fileRd, err = OpenFileRd(ddc.nameRd, filePerm)
		}

		if err != nil {
			// Do nothing
		} else if fileWr, err = OpenFileWr(ddc.nameWr, filePerm); err != nil {
			fileRd.Close()
		} else {
			ddc.muRd.Lock()
			ddc.refRd++
			ddc.muRd.Unlock()
			ddc.muWr.Lock()
			ddc.refWr++
			ddc.muWr.Unlock()
			ddc.fileRd = append(ddc.fileRd, fileRd)
			ddc.fileWr = append(ddc.fileWr, fileWr)
		}
	} else {
		err = syscall.EINVAL
	}

	return err
}

func (ddc *ddCopier) ReadAt(idx int, buf []byte, off int64) (int, error) {
	if len(ddc.patterns) > 0 {
		blockIndex := (off / ddc.blockSize) % int64(len(ddc.blocks))
		patternSize := len(ddc.patterns[blockIndex])
		n := 0

		for n < len(buf) {
			r := len(buf) - n
			if r > patternSize {
				r = patternSize
			}
			copy(buf[n:n+r], ddc.blocks[blockIndex][0:r])
			n = n + r
		}
		return n, nil
	} else if len(ddc.fileRd) == 0 {
		return -1, syscall.EBADF
	} else if idx < len(ddc.fileRd) {
		ddc.fileRd[idx].ReadAt(buf, off)
	}
	return ddc.fileRd[0].ReadAt(buf, off) // share
}

func (ddc *ddCopier) ReadClose(idx int) error {
	var err error
	if len(ddc.fileRd) == 0 && len(ddc.patterns) == 0 {
		err = syscall.EBADF
	} else if len(ddc.fileRd) == 1 { // share
		ddc.muRd.Lock()
		if ddc.refRd > 0 {
			ddc.refRd--
		}
		ref := ddc.refRd
		ddc.muRd.Unlock()
		if ref == 0 {
			err = ddc.fileRd[0].Close()
		}
	} else if idx < len(ddc.fileRd) {
		err = ddc.fileRd[idx].Close()
	}
	return err
}

func (ddc *ddCopier) WriteAt(idx int, buf []byte, off int64) (int, error) {
	if len(ddc.fileWr) == 0 {
		return -1, syscall.EBADF
	} else if idx < len(ddc.fileWr) {
		return ddc.fileWr[idx].WriteAt(buf, off)
	}
	return ddc.fileWr[0].WriteAt(buf, off) // share
}

func (ddc *ddCopier) WriteClose(idx int) error {
	var err error
	if len(ddc.fileWr) == 0 {
		err = syscall.EBADF
	} else if len(ddc.fileWr) == 1 { // share
		ddc.muWr.Lock()
		if ddc.refWr > 0 {
			ddc.refWr--
		}
		ref := ddc.refWr
		ddc.muWr.Unlock()
		if ref == 0 {
			err = ddc.fileWr[0].Close()
		}
	} else if idx < len(ddc.fileWr) {
		err = ddc.fileWr[idx].Close()
	}
	return err
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

	blockSizeStr := flag.String("bs", "1Mi", "Set both input and output block size to n bytes")
	count := flag.Int64("count", -1, "Copy only n input blocks")
	fileRd := flag.String("if", "", "Read input from file instead of the standard input")
	fileWr := flag.String("of", "", "Write output to file instead of the standard output")
	flag.Var(&inputPatterns, "ip", "Create input blocks from input patterns")
	rateBpsStr := flag.String("rate", "0", "Copy rate limit in bits per second")
	share := flag.Bool("share", true, "Share a single read and write file descriptor between threads")
	threads := flag.Int("threads", runtime.NumCPU(), "Number of copy threads")

	flag.Parse()

	blockSize := int64(0)
	if f, err := units.ToNumber(*blockSizeStr); err == nil {
		blockSize = int64(f)
	}

	rateBps := uint64(0)
	if f, err := units.ToNumber(*rateBpsStr); err == nil {
		rateBps = uint64(f)
	}

	validateFlags(blockSize, count, &inputPatterns, *fileRd, *fileWr, *threads)

	req := make(chan int64, *threads)
	res := make(chan *ddInfo, *threads)

	var fpRd string
	var err error
	fpRd, err = filepath.Abs(*fileRd)
	fpRd = FixLongUncPath(fpRd)
	panicIf("fileRd", err)

	var fpWr string
	fpWr, err = filepath.Abs(*fileWr)
	fpWr = FixLongUncPath(fpWr)
	panicIf("fileWr", err)
	err = os.MkdirAll(filepath.Dir(fpWr), filePerm)
	panicIf("", err)

	copier := &ddCopier{
		blockSize: blockSize,
		nameRd: fpRd,
		nameWr: fpWr,
		patterns: inputPatterns,
	}

	var wg sync.WaitGroup
	for i := 0; i < *threads; i++ {
		err = copier.Open(i, *share)
		panicIf("copier " + strconv.Itoa(i), err)
		wg.Add(1)
		go copyWorker(i, copier, rateBps/(8*uint64(*threads)), &wg, req, res)
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
				blocks = blocks + 1
				sum.RdBytes = sum.RdBytes + ddi.RdBytes
				sum.RdDur = sum.RdDur + ddi.RdDur
				sum.WrBytes = sum.WrBytes + ddi.WrBytes
				sum.WrDur = sum.WrDur + ddi.WrDur
				mutex.Unlock()
			}
		}
		close(res)
		//writer.Sync()
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
	usec := int64(duration / time.Microsecond)
	if usec > 0 {
		rate = sum.WrBytes * int64(time.Second / time.Microsecond) / usec
	}

	avgRdTime := time.Duration(0)
	avgWrTime := time.Duration(0)
	if blocks > 0 {
		avgRdTime = sum.RdDur / time.Duration(blocks)
		avgWrTime = sum.WrDur / time.Duration(blocks)
	}

	if it % 10 == 0 {
		fmt.Fprintf(os.Stdout,
			"%17s %9s %9s %9s %9s %12s\n",
			"ELAPSED TIME",
			"BLOCKS",
			"AVG READ",
			"AVG WRITE",
			"SIZE",
			"RATE")
	}

	fmt.Fprintf(os.Stdout,
		"%17s %9s %9s %9s %9s %12s\n",
		units.ToTimeString(float64(duration)/float64(time.Second)),
		units.ToMetricString(float64(blocks), 3, "", ""),
		units.ToMetricString(avgRdTime.Seconds(), 3, "", "s"),
		units.ToMetricString(avgWrTime.Seconds(), 3, "", "s"),
		units.ToMetricString(float64(sum.WrBytes), 3, "", "B"),
		units.ToMetricString(float64(rate * 8), 3, "", "bps"))
}

func validateFlags(blockSize int64, count *int64, patternRd *flagStringSlice, fileRd, fileWr string, threads int) {
	if len(os.Args) < 2 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if blockSize <= 0 {
		panic("bs <= 0")
	}

	if *count < 0 {
		if len(*patternRd) == 0 {
			fi, err := os.Stat(fileRd)
			panicIf("if file failure", err)
			*count = fi.Size() / blockSize
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

func copyWorker(id int, copier *ddCopier, rate uint64, wg *sync.WaitGroup, req <-chan int64, res chan<- *ddInfo) {
	defer wg.Done()
	defer copier.ReadClose(id)
	defer copier.WriteClose(id)

	var err error
	buf := make([]byte, copier.blockSize)
	n := 0
	tb := tokenbucket.New(rate, uint64(copier.blockSize) * 1)
	eof := false

	for num := range req {
		ddi := ddInfo{}
		if !eof {
			timeA := time.Now()
			tb.Remove(uint64(copier.blockSize))
			n, err = copier.ReadAt(id, buf, num*copier.blockSize)
			eof = (err == io.EOF)
			panicIf(fmt.Sprintf("Reader %d failed", id), err, io.EOF)

			timeB := time.Now()
			ddi.RdBytes = int64(n)
			n, err = copier.WriteAt(id, buf[:n], num*copier.blockSize)
			panicIf(fmt.Sprintf("Writer %d failed", id), err)

			ddi.WrDur = time.Since(timeB)
			ddi.WrBytes = int64(n)
			ddi.RdDur = timeB.Sub(timeA)
		}
		res <- &ddi
	}
}
