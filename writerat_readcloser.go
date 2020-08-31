package miscio

import (
	"io"
	"os"
	"sync"
)

type rangeSet struct {
	m map[int64]bool
}

func newRangeSet() *rangeSet {
	return &rangeSet{map[int64]bool{}}
}

// Add marks all values [a, b) as included in the range set.
func (rs *rangeSet) Add(a, b int64) {
	for i := a; i < b; i++ {
		rs.m[i] = true
	}
}

// NextCap returns the highest value N for which [0, N) is covered by the range set.
func (rs *rangeSet) NextCap() int64 {
	i := int64(0)

	for {
		if covered, ok := rs.m[i]; ok && covered {
			i++

			continue
		}

		return i
	}
}

// Consume removes the first N values from the range set, adjusting all other values down by N.
// Calling Consume with a value of N greater than NextCap() results in undefined behavior.
//
// For example, if you have a rangeSet covering the following:
//   - [0, 5)
//   - [6, 8)
//
// Then calling Consume(4) would result in a range set with the following:
//   - [0, 1)
//   - [2, 4)
func (rs *rangeSet) Consume(n int64) {
	newMap := make(map[int64]bool, int64(len(rs.m))-n)

	for k, v := range rs.m {
		if k < n {
			continue
		}

		newMap[k-n] = v
	}

	rs.m = newMap
}

// WriterAtReadCloser is a struct implementing io.WriterAt and io.ReadCloser
// Writes are buffered in memory only until they are used by a call to Read().
// Bytes can only be read once from the buffer — they are dropped after a successful
// Read() call.
type WriterAtReadCloser struct {
	buf []byte
	m   sync.Mutex

	bytesAvail *rangeSet
	bytesRead  int64

	readClosed bool

	GrowthCoeff float64
}

// NewWriterAtReadCloser returns a new WriterAtReadCloser object. Its underlying
// buffer is preallocated to have n bytes.
func NewWriterAtReadCloser(n int) *WriterAtReadCloser {
	return &WriterAtReadCloser{
		buf:        make([]byte, n),
		bytesAvail: newRangeSet(),
		bytesRead:  0,
		readClosed: false,
	}
}

// Write copies the contents of p into the underlying buffer, beginning at the
// specified offset. The underlying buffer will expand as necessary, according
// to len(p) and wr.GrowthCoeff. It is not an error to write over the same section
// of the underlying buffer. Write returns an os.ErrClosed if Closed() was previously
// called.
func (wr *WriterAtReadCloser) WriteAt(p []byte, off int64) (n int, err error) {
	wr.m.Lock()
	defer wr.m.Unlock()

	if wr.readClosed {
		return 0, os.ErrClosed
	}

	// the caller shouldn't have to know about or care that we're shrinking the buffer from the
	// left-hand side as they're read.
	adjustedOffset := off - wr.bytesRead

	expLen := adjustedOffset + int64(len(p))
	if int64(len(wr.buf)) < expLen {
		if int64(cap(wr.buf)) < expLen {
			wr.growBuffer(expLen)
		}

		wr.buf = wr.buf[:expLen]
	}

	copy(wr.buf[adjustedOffset:], p)
	wr.bytesAvail.Add(adjustedOffset, adjustedOffset+int64(len(p)))

	return len(p), nil
}

func (wr *WriterAtReadCloser) growBuffer(expLen int64) {
	if wr.GrowthCoeff < 1 {
		wr.GrowthCoeff = 1
	}

	newBuf := make([]byte, expLen, int64(wr.GrowthCoeff*float64(expLen)))
	copy(newBuf, wr.buf)
	wr.buf = newBuf
}

// Read consumes up to len(p) bytes from the underlying buffer and writes them into
// p. io.EOF is Closed() was previously called.
func (wr *WriterAtReadCloser) Read(p []byte) (n int, err error) {
	wr.m.Lock()
	defer wr.m.Unlock()

	if wr.readClosed {
		return 0, io.EOF
	}

	// nolint:godox
	// TODO: If `readable` is zero, maybe block until some bytes were written?
	// 		Alternatively, consider parameterizing a `NumAllowedEmptyReads`, then
	// 		return an `io.ErrNoProgress` if `readable` is zero that many times in
	//		a row.
	readable := wr.bytesAvail.NextCap()
	if readable >= int64(len(p)) {
		readable = int64(len(p))
	}

	wr.bytesAvail.Consume(readable)
	wr.bytesRead += readable

	copy(p, wr.buf[:readable])
	wr.buf = wr.buf[readable:]

	return int(readable), nil
}

// Close closes off the WriterAtReadCloser for both future reading and writing.
// Subsequent calls to Read() will return io.EOF, and subsequent calls to Write()
// will return os.ErrClosed.
func (wr *WriterAtReadCloser) Close() error {
	wr.m.Lock()
	defer wr.m.Unlock()

	wr.readClosed = true

	return nil
}
