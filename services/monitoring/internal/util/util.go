package util

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"
)

// used to simulate different snap packages from 1 actual snap package
func RandomSuffixReader(src io.Reader, suffixLen int) io.Reader {
	// generate suffixLen random bytes
	rnd := make([]byte, suffixLen)
	if _, err := rand.Read(rnd); err != nil {
		rnd = fmt.Appendf(rnd, "%d", time.Now().UnixNano())
	}
	// return a reader that streams src, then the suffix
	return io.MultiReader(src, strings.NewReader(string(rnd)))
}

// MultiSourceReader takes a slice of io.Reader factories and returns one at random,
// wrapped in randomSuffixReader. Each factory should return a fresh io.Reader
// (e.g. reopening the file).
func MultiSourceReader(factories []func() (io.Reader, error), suffixLen int) (io.Reader, error) {
	if len(factories) == 0 {
		return nil, fmt.Errorf("no sources provided")
	}
	// pick a random index
	idxBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(factories))))
	if err != nil {
		// fallback to time-based
		idx := time.Now().UnixNano() % int64(len(factories))
		idxBig = big.NewInt(idx)
	}
	idx := int(idxBig.Int64())

	src, err := factories[idx]()
	if err != nil {
		return nil, fmt.Errorf("failed to open source #%d: %w", idx, err)
	}

	return RandomSuffixReader(src, suffixLen), nil
}
