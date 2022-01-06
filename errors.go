package miscio

import (
	"fmt"
	"io"
)

// ErrShortBuffer thinly wraps io.ErrShortBuffer. Calls to (*RollingLineBuffer).Read
// may return errors of this type.
type ErrShortBuffer struct {
	minimumSize int
}

// Unwrap allows miscio.ErrShortBuffer to satisfy an errors.Is(err, io.ErrShortBuffer)
// check.
func (err *ErrShortBuffer) Unwrap() error { return io.ErrShortBuffer }

// SizeNeeded returns the minimum size needed for a RollingLineBuffer's Read
// call to succeed.
func (err *ErrShortBuffer) SizeNeeded() int { return err.minimumSize }

// Error implements error for ErrShortBuffer
func (err *ErrShortBuffer) Error() string {
	return fmt.Errorf("%w: need buffer at least length %d", io.ErrShortBuffer, err.minimumSize).Error()
}
