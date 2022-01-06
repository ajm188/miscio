package miscio

import "testing"

func TestRollingLineBuffer(t *testing.T) {
	rb := NewRollingLineBuffer(2)
	_, err := rb.Write([]byte("hello\nworld\ngoodbye"))
	if err != nil {
		t.Fatalf("Write failed with %s", err)
	}

	assertBufferContents(t, []string{"world", "goodbye"}, rb)
}

func assertBufferContents(t *testing.T, expectedBuffer []string, rb *RollingLineBuffer) {
	t.Helper()

	if len(rb.buf) != len(expectedBuffer) {
		t.Errorf("assertBufferContents: should have %d elements, got %d", len(expectedBuffer), len(rb.buf))
		return
	}

	for i, expected := range expectedBuffer {
		if string(rb.buf[i]) != expected {
			t.Errorf("assertBufferContents mismatch at pos %d; got %v want %v", i, string(rb.buf[i]), expected)
		}
	}
}
