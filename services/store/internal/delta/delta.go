package delta

// TODO: Finish this implementation

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Strategy pattern for different delta generation techniques
// Every delta calculation tool needs random access to the source file
// And _may_ need random access to the target file
// *os.File for input guarantees random access, io.Reader for target allows simple scanning
// or if needed a temporary disk file can also be created based on it for random access

type DeltaHandler interface {
	DeltaFormat() string
	GenerateDelta(source *os.File, target io.Reader, out io.Writer) (uint64, error)
	ApplyDelta(source *os.File, delta *os.File, out io.Writer) (uint64, error)
}

type Xdelta3Generator struct {
	ctx context.Context
}

func NewXdelta3Generator(ctx context.Context) *Xdelta3Generator {
	return &Xdelta3Generator{ctx}
}

func (g *Xdelta3Generator) DeltaFormat() string {
	return "xdelta3"
}

func (g *Xdelta3Generator) GenerateDelta(source *os.File, target io.Reader, out io.Writer) (uint64, error) {
	cmd := exec.CommandContext(
		g.ctx,
		"xdelta3", "-e", "-f", "-S", "none",
		"-s", source.Name(),
		"-",
	)

	cmd.Stdin = target

	var written uint64
	cmd.Stdout = writerCounter{w: out, n: &written}

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("xdelta3 failed: %v (stderr: %s)", err, stderr.String())
	}

	return written, nil
}

func (g *Xdelta3Generator) ApplyDelta(source *os.File, delta *os.File, out io.Writer) (uint64, error) {
	sourcePath := source.Name()
	deltaPath := delta.Name()

	cmd := exec.CommandContext(
		g.ctx,
		"xdelta3", "-d", "-f",
		"-s", sourcePath,
		deltaPath,
		"-",
	)

	var written uint64
	cmd.Stdout = writerCounter{w: out, n: &written}

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("xdelta3 apply failed: %w (stderr: %s)", err, stderr.String())
	}

	return written, nil
}

// TODO: Add `snap-1-1-xdelta3` generator
// (see https://github.com/canonical/snapd/blob/dee46fb8fe1b0d4f1f7c6d82270bdd43996011a7/snap/squashfs/delta.go#L43)
// For "snap-aware" delta support

// ===== UTILITIES =======
type writerCounter struct {
	w io.Writer
	n *uint64
}

func (wc writerCounter) Write(p []byte) (int, error) {
	nn, err := wc.w.Write(p)
	*wc.n += uint64(nn)
	return nn, err
}
