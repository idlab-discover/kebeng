package util

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type readCloser struct {
	io.Reader
	io.Closer
}

func RandomSuffixReader(src io.ReadCloser, suffixLen int) io.ReadCloser {
	// generate random suffix
	rnd := make([]byte, suffixLen)
	if _, err := rand.Read(rnd); err != nil {
		rnd = fmt.Appendf(rnd, "%d", time.Now().UnixNano())
	}
	// wrap src+suffix in a MultiReader
	mr := io.MultiReader(src, strings.NewReader(string(rnd)))
	// return a ReadCloser that closes src
	return &readCloser{Reader: mr, Closer: src}
}

// returns a random source from the list of snaps and a random snap name
// caller has to close the reader by casting to io.Closer and calling Close
func RandomSnapReader(snaps []string, suffixLen int, dataDir string) (io.ReadCloser, string, error) {
	if len(snaps) == 0 {
		return nil, "", fmt.Errorf("no sources provided")
	}
	idxBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(snaps))))
	if err != nil {
		// fallback to time-based random
		idx := time.Now().UnixNano() % int64(len(snaps))
		idxBig = big.NewInt(idx)
	}
	idx := int(idxBig.Int64())

	fullPath := filepath.Join(dataDir, snaps[idx])
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, snaps[idx], fmt.Errorf("failed to open %q: %w", fullPath, err)
	}
	return RandomSuffixReader(f, suffixLen), fmt.Sprintf("%s_%s", snaps[idx], uuid.New().String()), nil
}
