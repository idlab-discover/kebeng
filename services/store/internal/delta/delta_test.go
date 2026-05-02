package delta

import (
	"bytes"
	"fmt"
	"testing"
)

// ===== UTILITY FUNCTIONS =====

func TestWriterCounter(t *testing.T) {
	t.Run("counts bytes written", func(t *testing.T) {
		var buf bytes.Buffer
		var n uint64
		wc := writerCounter{w: &buf, n: &n}

		wc.Write([]byte("hello"))
		wc.Write([]byte("world"))

		if n != 10 {
			t.Errorf("expected 10 bytes counted, got %d", n)
		}
		if buf.String() != "helloworld" {
			t.Errorf("unexpected buffer contents: %s", buf.String())
		}
	})

	t.Run("propagates write error", func(t *testing.T) {
		var n uint64
		wc := writerCounter{w: &failWriter{}, n: &n}

		_, err := wc.Write([]byte("hello"))
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

type failWriter struct{}

func (f *failWriter) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("write failed")
}
