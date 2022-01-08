package miscio

import (
	"io"
	"testing"
)

func mustWrite(t *testing.T, rb *RollingLineBuffer, w []byte) {
	t.Helper()
	_, err := rb.Write(w)
	if err != nil {
		t.Fatalf("Failed to write to buffer: %s", err)
	}
}

func TestRollingLineBuffer(t *testing.T) {
	rb := NewRollingLineBuffer(2)
	mustWrite(t, rb, []byte("hello\nworld\ngoodbye\n"))
	assertBufferContents(t, []string{"world", "goodbye"}, rb)
	assertPartialLine(t, nil, rb)
}

func assertBufferContents(t *testing.T, expectedBuffer []string, rb *RollingLineBuffer) {
	t.Helper()

	if len(rb.buf) != len(expectedBuffer) {
		t.Errorf("assertBufferContents: should have %d elements, got %d", len(expectedBuffer), len(rb.buf))
	}

	for i, expected := range expectedBuffer {
		if string(rb.buf[i]) != expected {
			t.Errorf("assertBufferContents mismatch at pos %d; got %v want %v", i, string(rb.buf[i]), expected)
		}
	}
}

func assertPartialLine(t *testing.T, want []byte, rb *RollingLineBuffer) {
	t.Helper()
	if len(want) != len(rb.curLine) {
		t.Errorf("assertPartialLine: expected partial line length %d, got %d", len(want), len(rb.curLine))
		return
	}
	if string(want) != string(rb.curLine) {
		t.Errorf("assertPartialLine: expected partial line to be '%v', got '%v'", []byte(want), rb.curLine)
	}
}

func TestRLBPartialWrite(t *testing.T) {
	rb := NewRollingLineBuffer(2)

	mustWrite(t, rb, []byte("a123456789\nb1234\nc1234567"))
	assertBufferContents(t, []string{"a123456789", "b1234"}, rb)
	assertPartialLine(t, []byte("c1234567"), rb)

	mustWrite(t, rb, []byte("89\naoeu"))
	assertBufferContents(t, []string{"b1234", "c123456789"}, rb)
	assertPartialLine(t, []byte("aoeu"), rb)

	mustWrite(t, rb, []byte("\n"))
	assertBufferContents(t, []string{"c123456789", "aoeu"}, rb)
	assertPartialLine(t, nil, rb)
}

func assertReadResults(t *testing.T, want string, got []byte, wantN, gotN int, wantErr, gotErr error) {
	t.Helper()
	gotString := string(got)
	if gotN > 0 {
		gotString = string(got[:gotN])
	}
	// only compare strings if we actually expected to read something
	if wantN > 0 && want != gotString {
		t.Errorf("Expected read buffer to contain '%v', got '%v'", []byte(want), got)
	}
	if wantN != gotN {
		t.Errorf("Expected bytes read to be %d, got %d", wantN, gotN)
	}
	if wantErr != gotErr {
		t.Errorf("Expected read error to be %v, got %v", wantErr, gotErr)
	}
}

func TestRLBReadToEnd(t *testing.T) {
	rb := NewRollingLineBuffer(2)
	mustWrite(t, rb, []byte("a123456789\nb123456789\nc123456789\n"))
	assertBufferContents(t, []string{"b123456789", "c123456789"}, rb)
	assertPartialLine(t, nil, rb)

	b := make([]byte, 12)
	n, err := rb.Read(b)
	assertReadResults(t, "b123456789\n", b, 11, n, nil, err)

	n, err = rb.Read(b)
	assertReadResults(t, "c123456789\n", b, 11, n, nil, err)

	n, err = rb.Read(b)
	assertReadResults(t, "c123456789\n", b, 0, n, io.EOF, err)
}

func TestRLBPartialRead(t *testing.T) {
	rb := NewRollingLineBuffer(2)
	mustWrite(t, rb, []byte("a123456789\nb1234\nc1234567"))
	assertBufferContents(t, []string{"a123456789", "b1234"}, rb)
	assertPartialLine(t, []byte("c1234567"), rb)

	b := make([]byte, 12)
	n, err := rb.Read(b)
	assertReadResults(t, "a123456789\n", b, 11, n, nil, err)

	n, err = rb.Read(b)
	assertReadResults(t, "b1234\n", b, 6, n, nil, err)

	n, err = rb.Read(b)
	assertReadResults(t, "", b, 0, n, io.EOF, err)

	mustWrite(t, rb, []byte("89\naoeu"))
	assertBufferContents(t, []string{"b1234", "c123456789"}, rb)
	assertPartialLine(t, []byte("aoeu"), rb)

	n, err = rb.Read(b)
	assertReadResults(t, "c123456789\n", b, 11, n, nil, err)

	mustWrite(t, rb, []byte("\n"))
	assertBufferContents(t, []string{"c123456789", "aoeu"}, rb)
	assertPartialLine(t, nil, rb)

	n, err = rb.Read(b)
	assertReadResults(t, "aoeu\n", b, 5, n, nil, err)

	n, err = rb.Read(b)
	assertReadResults(t, "", b, 0, n, io.EOF, err)
}

func TestRLBAllNewlines(t *testing.T) {
	rb := NewRollingLineBuffer(5)
	mustWrite(t, rb, []byte("\n\n\n\n\n\n\n\n\n"))
	assertBufferContents(t, []string{"", "", "", "", ""}, rb)
	assertPartialLine(t, nil, rb)

	b := make([]byte, 100)
	n, err := rb.Read(b)
	assertReadResults(t, "\n\n\n\n\n", b, 5, n, nil, err)
	n, err = rb.Read(b)
	assertReadResults(t, "", b, 0, n, io.EOF, err)
}
