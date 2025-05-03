package util

import (
	"testing"
)

// sample assertions
var snapRevision = `type:              snap-revision
authority-id:      auth123
snap-sha3-384:     abcdef
developer-id:      dev456
snap-id:           snap789
snap-revision:     42
snap-size:         12345
timestamp:         2025-04-29T12:00:00Z
sign-key-sha3-384: key321

signaturedatahere`

var snapDeclaration = `type:               snap-declaration
authority-id:       auth123
revision:           7
series:             16
snap-id:            snap789
snap-name:          test-snap
publisher-id:       pub999
timestamp:          2025-04-29T12:00:00Z
refresh-control:
  - snap1
aliases:
  - name: alias1
    target: cmd1
plugs:
  interface1:
    allow-installation: true
sign-key-sha3-384: key321

signature here`

var accountKey = `type:                account-key
authority-id:        auth123
revision:            3
public-key-sha3-384: keyABC
account-id:          acc001
name:                primary
since:               2025-01-01T00:00:00Z
until:               2026-01-01T00:00:00Z
sign-key-sha3-384:   keyXYZ

BASE64KEY

signature`

var account = `type:              account
authority-id:      auth123
revision:          5
account-id:        acc001
display-name:      Test User
username:          kebeng
validation:        certified
timestamp:         2025-04-29T12:00:00Z
sign-key-sha3-384: keyXYZ

signature`

func TestParseAssertion(t *testing.T) {
	tests := []struct {
		name string
		blob string
		want map[string]string
	}{
		{
			name: "snapRevision",
			blob: snapRevision,
			want: map[string]string{
				"type":              "snap-revision",
				"authority-id":      "auth123",
				"snap-sha3-384":     "abcdef",
				"developer-id":      "dev456",
				"snap-id":           "snap789",
				"snap-revision":     "42",
				"snap-size":         "12345",
				"timestamp":         "2025-04-29T12:00:00Z",
				"sign-key-sha3-384": "key321",
			},
		},
		{
			name: "snapDeclaration",
			blob: snapDeclaration,
			want: map[string]string{
				"type":              "snap-declaration",
				"authority-id":      "auth123",
				"revision":          "7",
				"series":            "16",
				"snap-id":           "snap789",
				"snap-name":         "test-snap",
				"publisher-id":      "pub999",
				"timestamp":         "2025-04-29T12:00:00Z",
				"sign-key-sha3-384": "key321",
			},
		},
		{
			name: "accountKey",
			blob: accountKey,
			want: map[string]string{
				"type":                "account-key",
				"authority-id":        "auth123",
				"revision":            "3",
				"public-key-sha3-384": "keyABC",
				"account-id":          "acc001",
				"name":                "primary",
				"since":               "2025-01-01T00:00:00Z",
				"until":               "2026-01-01T00:00:00Z",
				"sign-key-sha3-384":   "keyXYZ",
			},
		},
		{
			name: "account",
			blob: account,
			want: map[string]string{
				"type":              "account",
				"authority-id":      "auth123",
				"revision":          "5",
				"account-id":        "acc001",
				"display-name":      "Test User",
				"username":          "kebeng",
				"validation":        "certified",
				"timestamp":         "2025-04-29T12:00:00Z",
				"sign-key-sha3-384": "keyXYZ",
			},
		},
	}

	for _, tc := range tests {
		got := ParseAssertion(tc.blob)
		for k, wantV := range tc.want {
			if gotV, ok := got[k]; !ok {
				t.Errorf("%s: missing key %q", tc.name, k)
			} else if gotV != wantV {
				t.Errorf("%s: key %q: got %q, want %q", tc.name, k, gotV, wantV)
			}
		}
		// ensure no extra keys
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %d fields, want %d", tc.name, len(got), len(tc.want))
		}
	}
}
