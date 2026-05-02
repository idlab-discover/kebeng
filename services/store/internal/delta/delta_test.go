package delta

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"testing"
)

// ===== TEST SKIPPER =====

// Not every test environment (CI/CD pipelines, automation) have xdelta3 available.
// We don't want to fail the testing pipeline just because the executable is not there.

// requireXdelta3(t *testint.T) tells testing to skip tests using xdelta3 when xdelta3 is not available
func requireXdelta3(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("xdelta3"); err != nil {
		t.Skip("xdelta3 not found in PATH, skipping integration test")
	}
}

// ===== XDELTA3 GENERATOR =====

// TODO: Round-trip-test: two files, delta, apply delta to first file, verify second file is obtained
// Don't forget to use requireXdelta3, we don't want tests failing due to an uninstalled program in CI/CD

func TestXdelta3GeneratorFormat(t *testing.T) {
	g := NewXdelta3Generator(context.Background())
	if g.Format() != "xdelta3" {
		t.Errorf("expected format 'xdelta3', got '%s'", g.Format())
	}
}

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
