package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	mc "github.com/idlab-discover/kebeng/services/gateway/internal/macaroon"
	"github.com/idlab-discover/kebeng/services/gateway/internal/model"
	"github.com/stretchr/testify/assert"
	macaroon "gopkg.in/macaroon.v2"
)

// TestGenerateMacaroon_Success tests a successful generation.
func TestGenerateMacaroon_Success(t *testing.T) {
	ctx := context.Background()
	req := &model.GenerateMacaroonRequest{
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
	m, err := mc.MacaroonDeserialize(resp.Macaroon)
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
	req := &model.GenerateMacaroonRequest{
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
		req          *model.GenerateMacaroonRequest
		expectErrors []string
	}{
		{
			name: "valid request",
			req: &model.GenerateMacaroonRequest{
				Permissions: []string{"edit_account", "package_access"},
				Channels:    []string{"stable", "edge"},
				Packages: []model.PackageRestriction{
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
			req: &model.GenerateMacaroonRequest{
				Permissions: []string{"edit_account", "invalid_perm"},
				Channels:    []string{"stable"},
				Packages:    []model.PackageRestriction{},
				Expires:     "",
			},
			expectErrors: []string{"permission value 'invalid_perm' is not allowed"},
		},
		{
			name: "invalid channel",
			req: &model.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"invalid_channel"},
				Packages:    []model.PackageRestriction{},
				Expires:     "",
			},
			expectErrors: []string{"channel value 'invalid_channel' is not allowed"},
		},
		{
			name: "package with nothing provided",
			req: &model.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"stable"},
				Packages: []model.PackageRestriction{
					{Name: "", Series: "", SnapId: ""},
				},
				Expires: "",
			},
			expectErrors: []string{"package at index 0: must provide either name/series or snap_id"},
		},
		{
			name: "package with snap_id and extra fields",
			req: &model.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"stable"},
				Packages: []model.PackageRestriction{
					{Name: "pkg1", Series: "16", SnapId: "snap1"},
				},
				Expires: "",
			},
			expectErrors: []string{"package at index 0: snap_id should be provided alone"},
		},
		{
			name: "package missing name or series",
			req: &model.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"stable"},
				Packages: []model.PackageRestriction{
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
			req: &model.GenerateMacaroonRequest{
				Permissions: []string{"edit_account"},
				Channels:    []string{"stable"},
				Packages:    []model.PackageRestriction{},
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
			if len(tc.expectErrors) == 0 {
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

func TestHasPermission(t *testing.T) {
	// Test case 1: Macaroon with no ACL caveats returns false.
	m1, err := macaroon.New([]byte("secret"), []byte("id"), "location", macaroon.V1)
	assert.NoError(t, err)
	assert.False(t, HasPermission(m1, "edit_account"), "Expected false when no ACL caveats exist")

	// Test case 2: Valid ACL caveat that includes "edit_account".
	m2, err := macaroon.New([]byte("secret"), []byte("id"), "location", macaroon.V1)
	assert.NoError(t, err)
	permissions := []string{"edit_account", "other_perm"}
	permsJSON, err := json.Marshal(permissions)
	assert.NoError(t, err)
	aclCaveat := fmt.Sprintf("location|acl|%s", permsJSON)
	err = m2.AddFirstPartyCaveat([]byte(aclCaveat))
	assert.NoError(t, err)
	assert.True(t, HasPermission(m2, "edit_account"), "Expected true for permission 'edit_account'")
	assert.False(t, HasPermission(m2, "nonexistent"), "Expected false for non-existent permission")

	// Test case 3: Malformed ACL caveat: missing JSON part.
	m3, err := macaroon.New([]byte("secret"), []byte("id"), "location", macaroon.V1)
	assert.NoError(t, err)
	// Intentionally not including the JSON array after "|acl|"
	err = m3.AddFirstPartyCaveat([]byte("location|acl"))
	assert.NoError(t, err)
	assert.False(t, HasPermission(m3, "edit_account"), "Expected false when ACL caveat is malformed")

	// Test case 4: ACL caveat with invalid JSON.
	m4, err := macaroon.New([]byte("secret"), []byte("id"), "location", macaroon.V1)
	assert.NoError(t, err)
	err = m4.AddFirstPartyCaveat([]byte("location|acl|notjson"))
	assert.NoError(t, err)
	assert.False(t, HasPermission(m4, "edit_account"), "Expected false when JSON in ACL caveat is invalid")
}

func TestGetRootMacaroonsFromString(t *testing.T) {
	// Each test case defines an input string and the expected root and discharge tokens
	// according to the current implementation.
	tests := []struct {
		name              string
		input             string
		expectedRoot      string
		expectedDischarge string
	}{
		{
			name:              "Both tokens present with comma",
			input:             "Macaroon root=abc123, discharge=def456",
			expectedRoot:      "abc123",
			expectedDischarge: "def456",
		},
		{
			name:              "Both tokens present without comma",
			input:             "Macaroon root=abc123 discharge=def456",
			expectedRoot:      "abc123",
			expectedDischarge: "def456",
		},
		{
			name:              "Only root token provided",
			input:             "Macaroon root=abc123",
			expectedRoot:      "abc123",
			expectedDischarge: "",
		},
		{
			name:              "Only discharge token provided",
			input:             "Macaroon discharge=def456",
			expectedRoot:      "",
			expectedDischarge: "def456",
		},
		{
			name:              "No Macaroon prefix provided",
			input:             "root=abc123, discharge=def456",
			expectedRoot:      "abc123",
			expectedDischarge: "def456",
		},
		{
			name:              "Extra spaces in tokens",
			input:             "Macaroon   root=abc123,   discharge=def456",
			expectedRoot:      "abc123",
			expectedDischarge: "def456",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, discharge := GetRootMacaroonsFromString(tc.input)
			assert.Equal(t, tc.expectedRoot, root, "unexpected root token (input: %q)", tc.input)
			assert.Equal(t, tc.expectedDischarge, discharge, "unexpected discharge token (input: %q)", tc.input)
		})
	}
}
