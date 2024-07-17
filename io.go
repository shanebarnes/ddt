package ddt

import (
	"io"
	"time"
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
