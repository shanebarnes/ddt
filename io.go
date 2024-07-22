package ddt

import (
	"io"
	"math/rand/v2"
	"time"

	"v8.run/go/exp/fastrand"
)

type OpInfo struct {
	NumBytes int64
	NumCalls int64
	Duration time.Duration
	start    time.Time
}

func newOpInfo() OpInfo {
	return OpInfo{}
}

func (op *OpInfo) startTime() {
	op.start = time.Now()
}

func (op *OpInfo) stopTime() {
	op.Duration += time.Since(op.start)
}

type NullWriter struct {
	writer io.Writer
}

func NewNullWriter() *NullWriter {
	return &NullWriter{
		writer: io.Discard,
	}
}

func (nw *NullWriter) Write(p []byte) (int, error) {
	return nw.writer.Write(p)
}

func (nw *NullWriter) WriteAt(p []byte, off int64) (int, error) {
	return nw.Write(p)
}

// RandReader implements io.Reader and will provide as many pseudo-random bytes
// as are read from it.
type RandReader struct {
	reader *fastrand.FastRandReader
	seed   uint64
}

// NewRandReader creates and returns a new instance of RandReader.
func NewRandReader() *RandReader {
	seed := rand.Uint64()
	return &RandReader{
		reader: &fastrand.FastRandReader{
			RNG: fastrand.WithSeed(seed),
		},
		seed: seed,
	}
}

// Read fills the provided buffer with pseudo-random bytes.
func (rr *RandReader) Read(p []byte) (int, error) {
	return rr.reader.Read(p)
}

func (rr *RandReader) ReadAt(p []byte, off int64) (int, error) {
	return rr.Read(p)
}

type Reader struct {
	info   OpInfo
	reader io.Reader
}

func NewReader(reader io.Reader) *Reader {
	return &Reader{
		info:   newOpInfo(),
		reader: reader,
	}
}

func (r *Reader) Info() OpInfo {
	return r.info
}

func (r *Reader) Read(p []byte) (n int, err error) {
	r.info.startTime()
	defer func() {
		r.info.NumCalls++
		if n > 0 {
			r.info.NumBytes += int64(n)
		}
		r.info.stopTime()
	}()
	n, err = r.reader.Read(p)
	return
}

type Writer struct {
	info   OpInfo
	writer io.Writer
}

func NewWriter(writer io.Writer) *Writer {
	return &Writer{
		info:   newOpInfo(),
		writer: writer,
	}
}

func (w *Writer) Info() OpInfo {
	return w.info
}

func (w *Writer) Write(p []byte) (n int, err error) {
	w.info.startTime()
	defer func() {
		w.info.NumCalls++
		if n > 0 {
			w.info.NumBytes += int64(n)
		}
		w.info.stopTime()
	}()
	n, err = w.writer.Write(p)
	return
}

// ZeroReader implements io.Reader and will provide as many ASCII NULL
// (0x00) bytes as are read from it.
type ZeroReader struct{}

// NewZeroReader creates and returns a new instance of ZeroReader.
func NewZeroReader() *ZeroReader {
	return &ZeroReader{}
}

// Read fills the provided buffer with ASCII NULL (0x00) bytes.
func (zr *ZeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func (zr *ZeroReader) ReadAt(p []byte, off int64) (int, error) {
	return zr.Read(p)
}
