package miscio

import (
	"fmt"
	"io"
	"sync"
	"testing"
)

func WriteInChunks(w io.WriterAt, b []byte, base, chunkSize int) error {
	var wg sync.WaitGroup

	for i := 0; i < len(b); i += chunkSize {
		wg.Add(1)

		upperBound := i + chunkSize
		if upperBound > len(b) {
			upperBound = len(b)
		}

		chunk := b[i:upperBound]

		go func(chunk []byte, offset int64) {
			defer wg.Done()
			w.WriteAt(chunk, offset)
		}(chunk, int64(i+base))
	}

	wg.Wait()

	return nil
}

func TestWriteThenRead(t *testing.T) {
	w := NewWriterAtReadCloser([]byte{})
	WriteInChunks(w, []byte("hello world"), 0, 2)

	buf := make([]byte, len("hello world"))
	n, err := w.Read(buf)

	if err != nil {
		t.Errorf("got error reading: %s. %d bytes read", err, n)
	}

	if string(buf) != "hello world" {
		t.Errorf("Read mismatch, have got %s want %s", buf, "hello world")
	}
}

func TestReadBeforeWrite(t *testing.T) {
	w := NewWriterAtReadCloser([]byte{})
	WriteInChunks(w, []byte("world"), 6, 4)
	fmt.Println(string(w.buf))

	buf := make([]byte, len("hello world"))
	n, err := w.Read(buf)

	if err != nil {
		t.Errorf("got error reading: %s. %d bytes read", err, n)
	}

	if n != 0 {
		t.Errorf("managed to read bytes even though LHS of reader wasn't ready: %d bytes read; got %s", n, buf)
	}

	WriteInChunks(w, []byte("hello "), 0, 6)
	n, err = w.Read(buf)

	if err != nil {
		t.Errorf("got error reading: %s. %d bytes read", err, n)
	}

	if n != len("hello world") {
		t.Errorf("expected %d bytes read, only read %d", len("hello world"), n)
	}

	if string(buf) != "hello world" {
		t.Errorf("Read mismatch, have %s want %s", buf, "hello world")
	}
}
