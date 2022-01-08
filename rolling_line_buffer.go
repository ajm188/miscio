package miscio

import (
	"bytes"
	"io"
	"sync"
)

// RollingLineBuffer provides an implementation of io.Reader and io.Writer that
// stores the N most recent lines (delimited with '\n') written to it. Reads are
// done forward-only; it does not implement io.Seeker.
//
// Reads will retrieve only full lines from a RollingLineBuffer and Writes will
// buffer incomplete lines. See Read / Write docs for details.
type RollingLineBuffer struct {
	m          sync.Mutex
	curLine    []byte
	buf        [][]byte
	capacity   int
	readpos    int
	lossyReads bool
	delim      string
}

// NewRollingLineBuffer returns a new RollingLineBuffer that holds `capacity`
// most recently-written lines. Buffers writes until a '\n' is encountered.
func NewRollingLineBuffer(capacity int) *RollingLineBuffer {
	return NewRollingLineBufferWithDelimiter(capacity, "\n")
}

// NewRollingLineBuffer returns a new RollingLineBuffer than holds `capacity`
// most recently-written lines. Buffers writes until the specified delimiter
// is encountered.
//
// TODO: multi-byte delimiters currently not reliable; if the Write call breaks
// the delimiter over two writes we won't notice so use them at your own risk.
func NewRollingLineBufferWithDelimiter(capacity int, delimiter string) *RollingLineBuffer {
	return &RollingLineBuffer{
		buf:      make([][]byte, 0, capacity),
		capacity: capacity,
		delim:    delimiter,
	}
}

var (
	_ io.Reader = (*RollingLineBuffer)(nil)
	_ io.Writer = (*RollingLineBuffer)(nil)
)

func (rb *RollingLineBuffer) LossyReads() bool {
	return rb.lossyReads
}

// Read implements io.Reader for RollingLineBuffer. Read reads one or more full
// lines into buf and returns according to the io.Reader specification. If buf
// is too small to hold the first available line, Read returns ErrShortBuffer
// to signal to the caller they need a bigger buffer.
//
// A partial line will not be read until the end-of-line delimiter is
// written.
//
// If more lines are written than the buffer has been allocated to store
// LossyReads will return true before (and only before) the next Read.
func (rb *RollingLineBuffer) Read(buf []byte) (int, error) {
	rb.m.Lock()
	defer rb.m.Unlock()

	if len(rb.buf) == 0 || rb.readpos >= len(rb.buf) {
		return 0, nil
	}

	if len(rb.buf[rb.readpos])+len(rb.delim) > len(buf) {
		return 0, &ErrShortBuffer{minimumSize: len(rb.buf[rb.readpos]) + len(rb.delim)}
	}

	tmp := make([]byte, 0, len(buf)+len(rb.delim))
	read := 0
	for rb.readpos < len(rb.buf) {
		if read+(len(rb.buf[rb.readpos])+len(rb.delim)) > len(buf) {
			break
		}

		tmp = append(tmp, rb.buf[rb.readpos]...)
		tmp = append(tmp, []byte(rb.delim)...)
		read += len(rb.buf[rb.readpos]) + len(rb.delim)
		rb.readpos++
	}

	rb.lossyReads = false
	return copy(buf, tmp), nil
}

// Write implements io.Writer for RollingLineBuffer. It uses the configured
// delimiter as a marker for line separation and will buffer content until
// the marker is written.
func (rb *RollingLineBuffer) Write(data []byte) (int, error) {
	lines := bytes.Split(data, []byte(rb.delim))
	lastIdx := bytes.LastIndex(data, []byte(rb.delim))

	// is the data being written flushable?
	flushableLastLine := false
	if lastIdx == (len(data) - len(rb.delim)) {
		flushableLastLine = true
		// strip the last "" that comes from bytes.Split when the last segment is
		// a delimeter
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		return 0, nil
	}

	rb.m.Lock()
	defer rb.m.Unlock()

	switch {
	case !flushableLastLine && len(lines) == 1:
		// the last line is not flushable, and no full lines created so just
		// append to the growing buffer and bail
		rb.curLine = append(rb.curLine, lines[0]...)
		lines = nil

	case len(rb.curLine) > 0:
		// we have a current line in progress that should be flushed
		rb.curLine = append(rb.curLine, lines[0]...)
		lines = lines[1:]
		rb.buf = append(rb.buf, rb.curLine)
		rb.curLine = nil
	}

	if len(lines) > 0 {
		if flushableLastLine {
			rb.buf = append(rb.buf, lines...)
		} else {
			rb.buf = append(rb.buf, lines[:len(lines)-1]...)
			rb.curLine = lines[len(lines)-1]
		}
	}

	if len(rb.buf) > rb.capacity {
		shift := len(rb.buf) - rb.capacity
		rb.buf = rb.buf[shift:]
		rb.readpos -= shift
		if rb.readpos < 0 {
			rb.lossyReads = true
			rb.readpos = 0
		}
	}

	return len(data), nil
}
