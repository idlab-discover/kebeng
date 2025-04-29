package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"gateway/internal/config"
	"gateway/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"

	// Adjust the import path for your mock store client.
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestVerifyMacaroonHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Create a dummy base handler. For VerifyMacaroon the internals aren’t used.
	baseHandler := util.NewBaseHandler(nil, nil, nil, nil)
	h := &Handler{BaseHandler: baseHandler}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Make a dummy request.
	c.Request = httptest.NewRequest("GET", "/verifyMacaroon", nil)

	h.VerifyMacaroon(c)

	// Expect HTTP 501 (Not Implemented) and an error message.
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "not implemented too many unknowns", "Expected error message not found")
}

func TestGenerateMacaroonHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	macConfigValid := &config.MacaroonConfig{
		RootKey:            "my-root-key",
		RootId:             "my-root-id",
		RootLocation:       "my-location",
		DischargeKey:       "my-discharge-key",
		ThirdPartyLocation: "third-party-location",
	}

	storeErr := &cerrorpb.Error{Code: cerror.InternalServerError, Message: "store error"}

	successEntriesResponse := &storepb.GetEntriesResponse{
		Entries: []*storepb.GetEntryResponse{
			{
				Id: "snap1",
			},
		},
		Errors: nil,
	}

	errorEntriesResponse := &storepb.GetEntriesResponse{
		Entries: nil,
		Errors:  []*cerrorpb.Error{storeErr},
	}

	type testCase struct {
		name                      string
		requestBody               string
		macConfig                 *config.MacaroonConfig
		storeResponse             *storepb.GetEntriesResponse
		expectedHTTPStatus        int
		expectedResponseSubstring string
	}
	tests := []testCase{
		{
			name:        "invalid JSON",
			requestBody: "{invalid_json",
			macConfig:   macConfigValid,
			// Store client is not called because binding fails.
			expectedHTTPStatus:        http.StatusBadRequest,
			expectedResponseSubstring: "Syntax error",
		},
		{
			name: "validation error",
			// A request with an invalid package restriction: package fields all empty.
			requestBody: `{
				"permissions": ["edit_account"],
				"channels": ["stable"],
				"packages": [{"name": "", "series": "", "snap_id": ""}],
				"expires": ""
			}`,
			macConfig: macConfigValid,
			// Store client is not reached because validation fails.
			expectedHTTPStatus:        http.StatusBadRequest,
			expectedResponseSubstring: "must provide either name/series or snap_id",
		},
		{
			name: "store error",
			requestBody: `{
				"permissions": ["edit_account"],
				"channels": ["stable"],
				"packages": [{"snap_id": "snap1"}],
				"expires": ""
			}`,
			macConfig:                 macConfigValid,
			storeResponse:             errorEntriesResponse,
			expectedHTTPStatus:        http.StatusInternalServerError,
			expectedResponseSubstring: "store error",
		},
		{
			name: "macaroon generation error", // Invalid expiration time.
			requestBody: `{
				"permissions": ["edit_account"],
				"channels": ["stable"],
				"packages": [{"snap_id": "snap1"}],
				"expires": "not_a_valid_time" 
			}`,
			macConfig: macConfigValid,
			// Do not set a store response expectation as the request is rejected during validation.
			storeResponse:             nil,
			expectedHTTPStatus:        http.StatusBadRequest,
			expectedResponseSubstring: "cannot parse",
		},
		{
			name: "success",
			requestBody: `{
				"permissions": ["edit_account"],
				"channels": ["stable"],
				"packages": [{"snap_id": "snap1"}],
				"expires": ""
			}`,
			macConfig:          macConfigValid,
			storeResponse:      successEntriesResponse,
			expectedHTTPStatus: http.StatusOK,
			// In a successful response, the output JSON includes a non-empty "macaroon" string.
			expectedResponseSubstring: "macaroon",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new mock store client.
			mockStoreClient := new(storeClient.MockStoreClient)
			// Setup base handler with nil for account client, our mock store client, and nil for others.
			baseHandler := util.NewBaseHandler(nil, mockStoreClient, nil, nil)
			// Override Config.
			baseHandler.Config = &config.Config{
				MacaroonConfig: tc.macConfig,
			}
			h := &Handler{
				BaseHandler: baseHandler,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/generateMacaroon", bytes.NewBufferString(tc.requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			// Set up expectation on the store client only if storeResponse is not nil.
			if tc.storeResponse != nil {
				mockStoreClient.
					On("GetEntries", mock.Anything).
					Return(tc.storeResponse).
					Once()
			}

			h.GenerateMacaroon(c)

			assert.Equal(t, tc.expectedHTTPStatus, w.Code, "unexpected HTTP status for test: %s", tc.name)
			bodyStr := w.Body.String()
			assert.Contains(t, bodyStr, tc.expectedResponseSubstring, "unexpected response body for test: %s", tc.name)

			mockStoreClient.AssertExpectations(t)
		})
	}
}
