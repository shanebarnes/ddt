package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/shanebarnes/ddt"
	"github.com/shanebarnes/goto/tokenbucket"
	"github.com/shanebarnes/goto/units"
)

func main() {
	var (
		blockSize      string
		blockCount     int64
		sourceFilename string
		targetFilename string
		rateLimit      string
		shareFiles     bool
		threadCount    int
	)

	flag.StringVar(&blockSize, "bs", "512", "Set both input and output block size to n bytes")
	flag.Int64Var(&blockCount, "count", -1, "Copy only n input blocks")
	flag.StringVar(&sourceFilename, "if", "", "Read input from file instead of the standard input")
	flag.StringVar(&targetFilename, "of", "", "Write output to file instead of the standard output")
	flag.StringVar(&rateLimit, "rate", "0", "Copy rate limit in bits per second")
	flag.BoolVar(&shareFiles, "share", false, "Share a single read and write file descriptor between threads")
	flag.IntVar(&threadCount, "threads", 1, "Number of copy threads")

	flag.Parse()

	var blockSizeInBytes int64
	if f, err := units.ToNumber(blockSize); err == nil {
		blockSizeInBytes = int64(f)
	}

	var rateLimitInBps uint64
	if f, err := units.ToNumber(rateLimit); err == nil {
		rateLimitInBps = uint64(f)
	}

	err := validateFlags(blockSizeInBytes, sourceFilename, targetFilename, threadCount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Argument validation failed: %v\n\n", err)
		flag.PrintDefaults()
		os.Exit(1)
	}

	sourceFilename, err = filepath.Abs(sourceFilename)
	exitIf("Failed to open input file", err)

	if blockCount < 0 {
		fi, err := os.Stat(sourceFilename)
		exitIf("Failed to stat input file", err)
		blockCount = fi.Size() / blockSizeInBytes
		if fi.Size()%blockSizeInBytes != 0 {
			blockCount++
		}
	}

	targetFilename, err = filepath.Abs(targetFilename)
	exitIf("Failed to create output file", err)

	err = os.MkdirAll(filepath.Dir(targetFilename), 0644)
	exitIf("Failed to create output file", err)

	workerInputCh := make(chan int64, threadCount)
	workerOutputCh := make(chan *ddInfo, threadCount)

	var (
		source, target *os.File
		wg             sync.WaitGroup
	)

	for i := 0; i < threadCount; i++ {
		if i == 0 || !shareFiles {
			source, err = os.OpenFile(sourceFilename, os.O_RDONLY, 0)
			exitIf(fmt.Sprintf("Failed to open source file instance %d", i), err)
			defer func() {
				_ = source.Close()
			}()

			target, err = os.OpenFile(targetFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			exitIf(fmt.Sprintf("Failed to open target file instance %d", i), err)
			defer func() {
				_ = target.Close()
			}()
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			copyWorker(i, source, target, blockSizeInBytes, rateLimitInBps/(8*uint64(threadCount)), workerInputCh, workerOutputCh)
		}()
	}

	start := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var (
		copiedBlocks    int64
		sum             ddInfo
		statusIteration int
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for copiedBlocks < blockCount {
			select {
			case ddi, ok := <-workerOutputCh:
				if ok {
					if ddi.write.NumBytes > 0 {
						copiedBlocks++
						sum.read.NumBytes += ddi.read.NumBytes
						sum.read.Duration += ddi.read.Duration
						sum.write.NumBytes += ddi.write.NumBytes
						sum.write.Duration += ddi.write.Duration
					}
				}
			case _, ok := <-ticker.C:
				if ok {
					printStats(statusIteration, &sum, copiedBlocks, blockCount, time.Since(start))
					statusIteration++
				}
			}
		}

		close(workerOutputCh)
	}()

	for i := int64(0); i < blockCount; i++ {
		workerInputCh <- i
	}
	close(workerInputCh)
	wg.Wait()
	stop := time.Now()
	printStats(statusIteration, &sum, copiedBlocks, blockCount, stop.Sub(start))
}

func printStats(iteration int, sum *ddInfo, blocksCopied, blocksTotal int64, duration time.Duration) {
	rate := int64(0)
	usec := int64(duration / time.Microsecond)
	if usec > 0 {
		rate = sum.write.NumBytes * int64(time.Second/time.Microsecond) / usec
	}

	avgRdTime := time.Duration(0)
	avgWrTime := time.Duration(0)
	progress := float64(0)
	if blocksTotal > 0 {
		progress = float64(blocksCopied) / float64(blocksTotal) * 100
	}

	if blocksCopied > 0 {
		avgRdTime = sum.read.Duration / time.Duration(blocksCopied)
		avgWrTime = sum.write.Duration / time.Duration(blocksCopied)
	}

	if iteration%10 == 0 {
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
		units.ToMetricString(float64(blocksCopied), 3, "", ""),
		fmt.Sprintf("%3.03f%%", progress),
		units.ToMetricString(avgRdTime.Seconds(), 3, "", "s"),
		units.ToMetricString(avgWrTime.Seconds(), 3, "", "s"),
		units.ToMetricString(float64(sum.write.NumBytes), 3, "", "B"),
		units.ToMetricString(float64(rate*8), 3, "", "bps"))
}

func validateFlags(blockSizeInBytes int64, sourceFilename, targetFilename string, threadCount int) error {
	switch {
	case len(os.Args) < 2:
		return fmt.Errorf("argc < 2")
	case blockSizeInBytes <= 0:
		return fmt.Errorf("bs <= 0")
	case sourceFilename == "":
		return fmt.Errorf("if == \"\"")
	case targetFilename == "":
		return fmt.Errorf("of == \"\"")
	case threadCount < 1:
		return fmt.Errorf("threads < 1")
	default:
		return nil
	}
}

func copyWorker(id int, source, target *os.File, blockSize int64, maxRateInBytesPerSecond uint64, input <-chan int64, output chan<- *ddInfo) {
	buf := make([]byte, blockSize)
	limiter := tokenbucket.New(maxRateInBytesPerSecond, uint64(blockSize))

	for blockNumber := range input {
		offset := blockNumber * blockSize
		reader := ddt.NewReader(io.NewSectionReader(source, offset, blockSize))
		writer := ddt.NewWriter(io.NewOffsetWriter(target, offset))
		limiter.Remove(uint64(blockSize))
		_, err := io.CopyBuffer(writer, reader, buf)
		exitIf(fmt.Sprintf("Copy worker instance %d failed", id), err, io.EOF)

		output <- &ddInfo{
			read:  reader.Info(),
			write: writer.Info(),
		}
	}
}
