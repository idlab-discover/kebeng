package account

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"
	"gopkg.in/macaroon.v2"

	"github.com/stretchr/testify/assert"
)

// createDummyMacaroon creates a dummy macaroon with an ACL caveat if hasPermission is true.
func createDummyMacaroon(hasPermission bool) *macaroon.Macaroon {
	m, err := macaroon.New([]byte("secret"), []byte("id"), "location", macaroon.V1)
	if err != nil {
		panic(err)
	}
	if hasPermission {
		acl := []string{"edit_account"}
		permsJson, _ := json.Marshal(acl)
		_ = m.AddFirstPartyCaveat([]byte("location|acl|" + string(permsJson)))
	}
	return m
}

func TestPatchAccountHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAccClient := new(accClient.MockAccountClient)

	baseHandler := util.NewBaseHandler(mockAccClient, nil, nil, nil)

	// Create a handler with our mock account client.
	h := &Handler{
		BaseHandler: baseHandler,
	}

	type testCase struct {
		name                      string
		setEmail                  bool
		setMacaroon               bool
		macaroonHasPermission     bool
		requestBody               string
		expectedHTTPStatus        int
		expectedResponseSubstring string
		// If non-nil, we expect the account client to be called and return this response.
		accountClientResponse *proto.PatchAccountByEmailResponse
	}

	tests := []testCase{
		{
			name:                      "missing email",
			setEmail:                  false,
			setMacaroon:               true,
			macaroonHasPermission:     true,
			requestBody:               `{"short_namespace": "new_username"}`,
			expectedHTTPStatus:        http.StatusBadRequest,
			expectedResponseSubstring: "missing email",
		},
		{
			name:                      "missing macaroon",
			setEmail:                  true,
			setMacaroon:               false,
			macaroonHasPermission:     true,
			requestBody:               `{"short_namespace": "new_username"}`,
			expectedHTTPStatus:        http.StatusBadRequest,
			expectedResponseSubstring: "missing macaroon",
		},
		{
			name:                      "lacks permission",
			setEmail:                  true,
			setMacaroon:               true,
			macaroonHasPermission:     false, // dummy macaroon without ACL caveat
			requestBody:               `{"short_namespace": "new_username"}`,
			expectedHTTPStatus:        http.StatusForbidden,
			expectedResponseSubstring: "missing permission to edit account",
		},
		{
			name:                      "invalid JSON",
			setEmail:                  true,
			setMacaroon:               true,
			macaroonHasPermission:     true,
			requestBody:               `{"short_namespace": new_username}`, // invalid JSON
			expectedHTTPStatus:        http.StatusBadRequest,
			expectedResponseSubstring: "Syntax error", // expected part of the bind error message
		},
		{
			name:                  "account client error",
			setEmail:              true,
			setMacaroon:           true,
			macaroonHasPermission: true,
			requestBody:           `{"short_namespace": "new_username"}`,
			expectedHTTPStatus:    http.StatusInternalServerError,
			// Expect an error message from the account client response.
			expectedResponseSubstring: "internal-server-error",
			accountClientResponse: &proto.PatchAccountByEmailResponse{
				ShortNamespace: "",
				Errors: []*proto.Error{
					{Code: cerror.InternalServerError, Message: "client error"},
				},
			},
		},
		{
			name:                      "success",
			setEmail:                  true,
			setMacaroon:               true,
			macaroonHasPermission:     true,
			requestBody:               `{"short_namespace": "new_username"}`,
			expectedHTTPStatus:        http.StatusOK,
			expectedResponseSubstring: "success", // we expect {"success": true}
			accountClientResponse: &proto.PatchAccountByEmailResponse{
				ShortNamespace: "new_username",
				Errors:         []*proto.Error{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a Gin test context.
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("PATCH", "/patchAccount", strings.NewReader(tc.requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			// Set required context keys.
			if tc.setEmail {
				c.Set("email", "test@example.com")
			}
			if tc.setMacaroon {
				dummyMac := createDummyMacaroon(tc.macaroonHasPermission)
				c.Set("macaroon", dummyMac)
			}
			// Create a mock account client.
			// If we expect a call to PatchAccountByEmail, set the expectation.
			if tc.setEmail && tc.setMacaroon && tc.macaroonHasPermission && tc.accountClientResponse != nil {
				mockAccClient.On("PatchAccountByEmail", "test@example.com", "new_username").
					Return(tc.accountClientResponse).Once()
			}

			// Call the patchAccount handler.
			h.PatchAccount(c) // Note: make sure the function name matches your actual handler function.

			// Verify the HTTP status.
			assert.Equal(t, tc.expectedHTTPStatus, w.Code, "unexpected HTTP status")
			body := w.Body.String()
			assert.Contains(t, body, tc.expectedResponseSubstring, "unexpected response body")

			// If we expected the account client call, assert that expectations were met.
			if tc.accountClientResponse != nil {
				mockAccClient.AssertExpectations(t)
			}
		})
	}
}
