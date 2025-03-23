package auth

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/message"
	"github.com/stretchr/testify/assert"
	macaroon "gopkg.in/macaroon.v2"
)

// validMacaroonForTest is a helper that deserializes a macaroon string using our MacaroonDeserialize.
func validMacaroonForTest(t *testing.T, m *macaroon.Macaroon) string {
	t.Helper()
	bin, err := m.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal macaroon: %v", err)
	}
	// Our deserializer expects a base64.RawURLEncoding-encoded string.
	return base64.RawURLEncoding.EncodeToString(bin)
}

// TestGenerateMacaroon_Success tests a successful generation.
func TestGenerateMacaroon_Success(t *testing.T) {
	ctx := context.Background()
	req := &message.GenerateMacaroonRequest{
		Permissions: []string{"edit_account", "modify_account_key"},
		Channels:    []string{"stable", "beta"},
		Expires:     "", // let expiry be computed automatically if a default permission is found
	}
	// Example snapIDs map.
	snapIDs := map[string]bool{
		"snap1": true,
		"snap2": true,
	}
	macConfig := &config.MacaroonConfig{
		RootKey:            "my-root-key",
		RootId:             "my-root-id",
		RootLocation:       "my-location",
		DischargeKey:       "my-discharge-key",
		ThirdPartyLocation: "third-party-location",
	}

	resp := GenerateMacaroon(ctx, req, snapIDs, macConfig)
	// Expect no errors.
	if len(resp.Errors) > 0 {
		t.Fatalf("Expected no errors, got: %+v", resp.Errors)
	}
	assert.NotEmpty(t, resp.Macaroon, "Expected non-empty macaroon string")

	// Verify we can deserialize the generated macaroon.
	m, err := MacaroonDeserialize(resp.Macaroon)
	assert.NoError(t, err, "Expected to deserialize macaroon without error")

	// Check that at least one caveat contains the ACL prefix.
	caveats := m.Caveats()
	var aclFound bool
	for _, cav := range caveats {
		if strings.Contains(string(cav.Id), "|acl|") {
			aclFound = true
			break
		}
	}
	assert.True(t, aclFound, "Expected to find an ACL caveat in the macaroon")

	// Optionally, you can print out caveats for debugging.
	// for _, cav := range caveats {
	//     t.Logf("Caveat: %s", string(cav.Id))
	// }
}

// TestGenerateMacaroon_Failure simulates a failure when creating the macaroon.
// For example, providing an empty RootKey might cause macaroon.New to fail.
func TestGenerateMacaroon_Failure(t *testing.T) {
	ctx := context.Background()
	// lets pass in a bad expires field
	req := &message.GenerateMacaroonRequest{
		Permissions: []string{"edit_account"},
		Channels:    []string{"stable"},
		Expires:     "bad-expires", // invalid expiry
	}
	snapIDs := map[string]bool{"snap1": true}
	// config will always correct since it is checked at startup
	macConfig := &config.MacaroonConfig{
		RootKey:            "my-root-key",
		RootId:             "my-root-id",
		RootLocation:       "my-location",
		DischargeKey:       "my-discharge-key",
		ThirdPartyLocation: "third-party-location",
	}

	resp := GenerateMacaroon(ctx, req, snapIDs, macConfig)
	// Expect errors.
	assert.NotEmpty(t, resp.Errors, "Expected errors due to invalid configuration")
}

func TestValidateGenerateMacaroonRequest(t *testing.T) {
	tests := []struct {
		name         string
		req          *message.GenerateMacaroonRequest
		expectErrors []string
	}{
		{
			name: "valid request",
			req: &message.GenerateMacaroonRequest{
				Permissions: []string{"edit_account", "package_access"},
				Channels:    []string{"stable", "edge"},
				Packages: []message.PackageRestriction{
					// Valid package using snap_id only.
					{SnapId: "snap1"},
					// Valid package using name and series.
					{Name: "pkg1", Series: "16"},
				},
				Expires: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
			expectErrors: nil,
		},
		{
			name: "invalid permission",
			req: &message.GenerateMacaroonRequest{
				Permissions: []string{"edit_account", "invalid_perm"},
				Channels:    []string{"stable"},
				Packages:    []message.PackageRestriction{},
				Expires:     "",
			},
			expectErrors: []string{"permission value 'invalid_perm' is not allowed"},
		},
		{
			name: "invalid channel",
			req: &message.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"invalid_channel"},
				Packages:    []message.PackageRestriction{},
				Expires:     "",
			},
			expectErrors: []string{"channel value 'invalid_channel' is not allowed"},
		},
		{
			name: "package with nothing provided",
			req: &message.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"stable"},
				Packages: []message.PackageRestriction{
					{Name: "", Series: "", SnapId: ""},
				},
				Expires: "",
			},
			expectErrors: []string{"package at index 0: must provide either name/series or snap_id"},
		},
		{
			name: "package with snap_id and extra fields",
			req: &message.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"stable"},
				Packages: []message.PackageRestriction{
					{Name: "pkg1", Series: "16", SnapId: "snap1"},
				},
				Expires: "",
			},
			expectErrors: []string{"package at index 0: snap_id should be provided alone"},
		},
		{
			name: "package missing name or series",
			req: &message.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"stable"},
				Packages: []message.PackageRestriction{
					{Name: "pkg1", Series: "", SnapId: ""},
					{Name: "", Series: "16", SnapId: ""},
				},
				Expires: "",
			},
			expectErrors: []string{
				"package at index 0: must provide both name and series",
				"package at index 1: must provide both name and series",
			},
		},
		{
			name: "invalid expires format",
			req: &message.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"stable"},
				Packages:    []message.PackageRestriction{},
				Expires:     "invalid-date",
			},
			expectErrors: []string{"expires:"}, // we look for the substring "expires:" in the error
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			el := cerror.NewErrorList()
			// Call the validation function.
			ValidateGenerateMacaroonRequest(tc.req, el)
			if tc.expectErrors == nil || len(tc.expectErrors) == 0 {
				assert.Equal(t, 0, len(*el), "Expected no errors, got: %v", *el)
			} else {
				// Combine error messages into one string for easier searching.
				combined := ""
				for _, err := range *el {
					combined += err.Message + " "
				}
				for _, expected := range tc.expectErrors {
					assert.Contains(t, combined, expected, "Expected error message to contain %q", expected)
				}
			}
		})
	}
}
