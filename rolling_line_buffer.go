package miscio

import (
	"bytes"
	"io"
	"sync"
)

// RollingLineBuffer provides an implementation of io.Reader and io.Writer that
// stores the N most recent lines (delimited with '\n') written to it. Reads are
// done forward-only; it does not implement io.Seeker.
type RollingLineBuffer struct {
	m        sync.Mutex
	buf      [][]byte
	capacity int
	readpos  int
}

// NewRollingLineBuffer returns a new RollingLineBuffer that holds `capacity`
// most recently-written lines.
func NewRollingLineBuffer(capacity int) *RollingLineBuffer {
	return &RollingLineBuffer{
		buf:      make([][]byte, 0, capacity),
		capacity: capacity,
	}
}

var (
	_ io.Reader = (*RollingLineBuffer)(nil)
	_ io.Writer = (*RollingLineBuffer)(nil)
)

// Read implements io.Reader for RollingLineBuffer. Read reads one or more full
// lines into buf and returns according to the io.Reader specification. If buf
// is too small to hold the first available line, Read returns ErrShortBuffer
// to signal to the caller they need a bigger buffer.
func (rb *RollingLineBuffer) Read(buf []byte) (int, error) {
	rb.m.Lock()
	defer rb.m.Unlock()

	if len(rb.buf) == 0 {
		return 0, nil
	}

	if len(rb.buf[rb.readpos]) > len(buf) {
		return 0, &ErrShortBuffer{minimumSize: len(rb.buf[rb.readpos])}
	}

	tmp := make([]byte, 0, len(buf))
	read := 0
	for rb.readpos < len(rb.buf) {
		if read+len(rb.buf[rb.readpos]) > len(buf) {
			break
		}

		tmp = append(tmp, rb.buf[rb.readpos]...)
		rb.readpos++
	}

	return copy(buf, tmp), nil
}

// Write implements io.Writer for RollingLineBuffer.
func (rb *RollingLineBuffer) Write(data []byte) (int, error) {
	lines := bytes.Split(data, []byte{'\n'})

	rb.m.Lock()
	defer rb.m.Unlock()

	rb.buf = append(rb.buf, lines...)
	if len(rb.buf) > rb.capacity {
		shift := len(rb.buf) - rb.capacity
		rb.buf = rb.buf[shift:]
		rb.readpos -= shift
		if rb.readpos < 0 {
			rb.readpos = 0
		}
	}

	return len(data), nil
}
