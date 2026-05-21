package util

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type readCloser struct {
	io.Reader
	io.Closer
}

func stripExt(path string) string {
	base := filepath.Base(path)          // e.g. "test.snap"
	ext := filepath.Ext(base)            // e.g. ".snap"
	return strings.TrimSuffix(base, ext) // "test"
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
	return RandomSuffixReader(f, suffixLen), fmt.Sprintf("%s-%s", stripExt(snaps[idx]), uuid.New().String()), nil
}

// parseAssertion reads only top‐level key:value lines until
// the first empty line, and skips unwanted keys.
func ParseAssertion(blob string) map[string]string {
	fields := make(map[string]string)
	// match lines like "key: value" at start of line
	re := regexp.MustCompile(`^([a-z0-9-]+):\s*(.+)$`)

	// keys in snap-declaration we want to ignore entirely
	skip := map[string]bool{
		"refresh-control": true,
		"aliases":         true,
		"plugs":           true,
		"slots":           true,
	}

	scanner := bufio.NewScanner(strings.NewReader(blob))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			// blank line → stop parsing headers
			break
		}
		// only unindented key:value lines
		if m := re.FindStringSubmatch(line); m != nil {
			key := m[1]
			val := m[2]
			if !skip[key] {
				fields[key] = val
			}
		}
	}
	return fields
}

func DeltaFileReader(path string) (io.ReadCloser, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("opening delta file %s: %v", path, err)
	}
	return f, filepath.Base(path), nil
}

func ComputeSHA3_384(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file for SHA computation %s: %w", path, err)
	}
	defer f.Close()

	hasher := sha3.New384()

	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("hashing file %s: %w", path, err)
	}
	return base64.RawURLEncoding.EncodeToString(hasher.Sum(nil)), nil
}

func SpecificSnapReader(filePath string, snapName string) (io.ReadCloser, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening snap file %s: %w", filePath, err)
	}
	return f, nil
}
