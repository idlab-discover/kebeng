package account

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/idlab-discover/kebeng/services/gateway/internal/util"

	"github.com/gin-gonic/gin"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/macaroon.v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
				Errors: []*cerrorpb.Error{
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
				Errors:         []*cerrorpb.Error{},
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

func TestCreateAccountHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAccClient := new(accClient.MockAccountClient)
	baseHandler := util.NewBaseHandler(mockAccClient, nil, nil, nil)

	// Create a handler with our mock account client.
	h := &Handler{
		BaseHandler: baseHandler,
	}

	type testCase struct {
		name                      string
		requestBody               string
		expectedHTTPStatus        int
		expectedResponseSubstring string
		// If non-nil, we expect the account client to be called and return this response.
		accountClientResponse *proto.AccountResponse
	}

	tests := []testCase{
		{
			name:                      "invalid JSON",
			requestBody:               `{"display_name": "John Doe", "username": jdoe, "email": "jdoe@example.com", "hashed_password": "password"}`,
			expectedHTTPStatus:        http.StatusBadRequest,
			expectedResponseSubstring: "Syntax error",
		},
		{
			name:                      "account client error",
			requestBody:               `{"display_name": "John Doe", "username": "jdoe", "email": "jdoe@example.com", "hashed_password": "password"}`,
			expectedHTTPStatus:        http.StatusInternalServerError,
			expectedResponseSubstring: "client error",
			accountClientResponse: &proto.AccountResponse{
				Id: "",
				Errors: []*cerrorpb.Error{
					{Code: cerror.InternalServerError, Message: "client error"},
				},
			},
		},
		{
			name:                      "success",
			requestBody:               `{"display_name": "John Doe", "username": "jdoe", "email": "jdoe@example.com", "hashed_password": "password"}`,
			expectedHTTPStatus:        http.StatusOK,
			expectedResponseSubstring: "123e4567-e89b-12d3-a456-426614174000",
			accountClientResponse: &proto.AccountResponse{
				Id:     "123e4567-e89b-12d3-a456-426614174000",
				Errors: []*cerrorpb.Error{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a Gin test context.
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/createAccount", strings.NewReader(tc.requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			// If we expect a call to CreateAccount, set the expectation on the mock.
			// We need to extract the request values from the JSON.
			if tc.accountClientResponse != nil {
				// Declare a temporary structure to unmarshal the request.
				var req struct {
					DisplayName    string `json:"display_name"`
					Username       string `json:"username"`
					Email          string `json:"email"`
					HashedPassword string `json:"hashed_password"`
				}
				err := json.Unmarshal([]byte(tc.requestBody), &req)
				// If there is an error in binding JSON here, it will be handled in the actual handler.
				if err == nil {
					mockAccClient.
						On("AddAccount", req.DisplayName, req.Username, req.Email, req.HashedPassword).
						Return(tc.accountClientResponse).
						Once()
				}
			}

			// Call the CreateAccount handler.
			h.AddAccount(c)

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

// TestGetAccountHandler tests the GetAccount endpoint using table-driven tests.
func TestGetAccountHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a dummy timestamp for use in responses.
	dummyTime := time.Now()
	dummyTimestamp := timestamppb.New(dummyTime)

	type testCase struct {
		name                      string
		setEmail                  bool
		getAccountResponse        *proto.AccountResponse
		getAccountKeysResponse    *proto.KeysResponse
		getEntriesResponse        *storepb.GetEntriesResponse
		getRevisionsResponse      *storepb.GetRevisionsByEntryIdResponses
		getAccountsResponse       *proto.GetAccountsByIdsResponse
		expectedHTTPStatus        int
		expectedResponseSubstring string
	}

	// Prepare an error for convenience.
	errResp := &cerrorpb.Error{Code: cerror.InternalServerError, Message: "client error"}
	storeErr := &cerrorpb.Error{Code: cerror.InternalServerError, Message: "store error"}

	Id := "entry-1"
	Base := "16"
	SnapName := "snap1"
	Status := "active"
	Price := 9.99
	Private := false
	IconUrl := "http://example.com/icon.png"
	PublisherId := "pub-1"
	Validation := "verified"

	tests := []testCase{
		{
			name:                      "missing email",
			setEmail:                  false,
			expectedHTTPStatus:        http.StatusBadRequest,
			expectedResponseSubstring: "missing email",
		},
		{
			name:     "account lookup error",
			setEmail: true,
			getAccountResponse: &proto.AccountResponse{
				Id:          "",
				DisplayName: "",
				Email:       "",
				Username:    "",
				Errors:      []*cerrorpb.Error{errResp},
			},
			expectedHTTPStatus:        http.StatusInternalServerError,
			expectedResponseSubstring: "client error",
		},
		{
			name:     "account keys error",
			setEmail: true,
			getAccountResponse: &proto.AccountResponse{
				Id:          "123e4567-e89b-12d3-a456-426614174000",
				DisplayName: "John Doe",
				Email:       "john@example.com",
				Username:    "johndoe",
				Errors:      []*cerrorpb.Error{},
			},
			getAccountKeysResponse: &proto.KeysResponse{
				Keys:   nil,
				Errors: []*cerrorpb.Error{errResp},
			},
			expectedHTTPStatus:        http.StatusInternalServerError,
			expectedResponseSubstring: "client error",
		},
		{
			name:     "store entries error",
			setEmail: true,
			getAccountResponse: &proto.AccountResponse{
				Id:          "123e4567-e89b-12d3-a456-426614174000",
				DisplayName: "John Doe",
				Email:       "john@example.com",
				Username:    "johndoe",
				Errors:      []*cerrorpb.Error{},
			},
			getAccountKeysResponse: &proto.KeysResponse{
				// Even an empty keys slice is acceptable, so simulate a valid keys response.
				Keys:   []*proto.KeyResponse{},
				Errors: []*cerrorpb.Error{},
			},
			getEntriesResponse: &storepb.GetEntriesResponse{
				Entries: nil,
				Errors:  []*cerrorpb.Error{storeErr},
			},
			expectedHTTPStatus:        http.StatusInternalServerError,
			expectedResponseSubstring: "store error",
		},
		{
			name:     "revisions error",
			setEmail: true,
			getAccountResponse: &proto.AccountResponse{
				Id:          "123e4567-e89b-12d3-a456-426614174000",
				DisplayName: "John Doe",
				Email:       "john@example.com",
				Username:    "johndoe",
				Errors:      []*cerrorpb.Error{},
			},
			getAccountKeysResponse: &proto.KeysResponse{
				Keys:   []*proto.KeyResponse{},
				Errors: []*cerrorpb.Error{},
			},
			getEntriesResponse: &storepb.GetEntriesResponse{
				Entries: []*storepb.GetEntryResponse{
					{
						Id:          Id,
						Base:        Base,
						SnapName:    SnapName,
						Status:      Status,
						Price:       Price,
						Since:       dummyTimestamp,
						Private:     Private,
						IconUrl:     IconUrl,
						PublisherId: PublisherId,
					},
				},
				Errors: []*cerrorpb.Error{},
			},
			getRevisionsResponse: &storepb.GetRevisionsByEntryIdResponses{
				Responses: nil,
				Errors:    []*cerrorpb.Error{storeErr},
			},
			// Note: No getAccountsResponse is provided, so the expectation for GetAccountsByIds will not be set.
			expectedHTTPStatus:        http.StatusInternalServerError,
			expectedResponseSubstring: "store error",
		},
		{
			name:     "publishers error",
			setEmail: true,
			getAccountResponse: &proto.AccountResponse{
				Id:          "123e4567-e89b-12d3-a456-426614174000",
				DisplayName: "John Doe",
				Email:       "john@example.com",
				Username:    "johndoe",
				Errors:      []*cerrorpb.Error{},
			},
			getAccountKeysResponse: &proto.KeysResponse{
				Keys:   []*proto.KeyResponse{},
				Errors: []*cerrorpb.Error{},
			},
			getEntriesResponse: &storepb.GetEntriesResponse{
				Entries: []*storepb.GetEntryResponse{
					{
						Id:          Id,
						Base:        Base,
						SnapName:    SnapName,
						Status:      Status,
						Price:       Price,
						Since:       dummyTimestamp,
						Private:     Private,
						IconUrl:     IconUrl,
						PublisherId: PublisherId,
					},
				},
				Errors: []*cerrorpb.Error{},
			},
			getRevisionsResponse: &storepb.GetRevisionsByEntryIdResponses{
				// Return a valid (but empty) revisions response.
				Responses: []*storepb.GetRevisionsByEntryIdResponse{
					{
						EntryId:   "entry-1",
						Revisions: []*storepb.GetRevisionResponse{}, // no revisions, but valid response
					},
				},
				Errors: []*cerrorpb.Error{},
			},
			getAccountsResponse: &proto.GetAccountsByIdsResponse{
				Accounts: nil,
				Errors:   []*cerrorpb.Error{errResp},
			},
			expectedHTTPStatus:        http.StatusInternalServerError,
			expectedResponseSubstring: "client error",
		},
		{
			name:     "success",
			setEmail: true,
			getAccountResponse: &proto.AccountResponse{
				Id:          "123e4567-e89b-12d3-a456-426614174000",
				DisplayName: "John Doe",
				Email:       "john@example.com",
				Username:    "johndoe",
				Errors:      []*cerrorpb.Error{},
			},
			getAccountKeysResponse: &proto.KeysResponse{
				Keys: []*proto.KeyResponse{
					{
						Name:    "key1",
						Sha3384: "abc123",
						Since:   dummyTimestamp,
						Until:   dummyTimestamp,
					},
				},
				Errors: []*cerrorpb.Error{},
			},
			getEntriesResponse: &storepb.GetEntriesResponse{
				Entries: []*storepb.GetEntryResponse{
					{
						Id:          Id,
						Base:        Base,
						SnapName:    SnapName,
						Status:      Status,
						Price:       Price,
						Since:       dummyTimestamp,
						Private:     Private,
						IconUrl:     IconUrl,
						PublisherId: PublisherId,
					},
				},
				Errors: []*cerrorpb.Error{},
			},
			getRevisionsResponse: &storepb.GetRevisionsByEntryIdResponses{
				// Return a valid revisions response.
				Responses: []*storepb.GetRevisionsByEntryIdResponse{
					{
						EntryId: "entry-1",
						Revisions: []*storepb.GetRevisionResponse{
							{
								SequenceNumber: 1,
								Version:        "v1",
								Status:         "stable",
								Architectures:  []string{"amd64"},
							},
						},
					},
				},
				Errors: []*cerrorpb.Error{},
			},
			getAccountsResponse: &proto.GetAccountsByIdsResponse{
				Accounts: []*proto.AccountResponse{
					{
						Id:          "pub-1",
						DisplayName: "Publisher",
						Username:    "publisher",
						Validation:  Validation,
					},
				},
				Errors: []*cerrorpb.Error{},
			},
			expectedHTTPStatus:        http.StatusOK,
			expectedResponseSubstring: "John Doe",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh mock clients for each test case.
			mockAccClient := new(accClient.MockAccountClient)
			// Assume MockStoreClient is defined similarly as your account client mock.
			mockStoreClient := new(storeClient.MockStoreClient)

			baseHandler := util.NewBaseHandler(mockAccClient, mockStoreClient, nil, nil)
			h := &Handler{
				BaseHandler: baseHandler,
			}

			// Create a new Gin test context.
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/getAccount", nil)
			c.Request.Header.Set("Content-Type", "application/json")

			// Set email in context if required.
			if tc.setEmail {
				c.Set("email", "test@example.com")
			}

			// Set expectations on the account client and store client mocks.
			if tc.setEmail {
				mockAccClient.On("GetAccountByEmail", "test@example.com").
					Return(tc.getAccountResponse).Once()
			}

			if tc.setEmail && tc.getAccountResponse != nil && len(tc.getAccountResponse.Errors) == 0 {
				mockAccClient.
					On("GetAccountKeysByAccountID", tc.getAccountResponse.Id).
					Return(tc.getAccountKeysResponse).Once()
			}

			if tc.setEmail && tc.getAccountResponse != nil && len(tc.getAccountResponse.Errors) == 0 &&
				tc.getAccountKeysResponse != nil && len(tc.getAccountKeysResponse.Errors) == 0 {
				mockStoreClient.
					On("GetEntriesByAccountID", tc.getAccountResponse.Id).
					Return(tc.getEntriesResponse).Once()
			}

			if tc.setEmail && tc.getAccountResponse != nil && len(tc.getAccountResponse.Errors) == 0 &&
				tc.getAccountKeysResponse != nil && len(tc.getAccountKeysResponse.Errors) == 0 &&
				tc.getEntriesResponse != nil && len(tc.getEntriesResponse.Errors) == 0 {

				mockStoreClient.
					On("GetRevisionsByEntryIds", mock.Anything).
					Return(tc.getRevisionsResponse).Once()

				// Only set expectation for GetAccountsByIds if a response is provided.
				if tc.getAccountsResponse != nil {
					mockAccClient.
						On("GetAccountsByIds", mock.Anything).
						Return(tc.getAccountsResponse).Once()
				}
			}

			// Call the GetAccount handler.
			h.GetAccount(c)

			// Verify the HTTP status and that the response contains the expected substring.
			assert.Equal(t, tc.expectedHTTPStatus, w.Code, "unexpected HTTP status")
			body := w.Body.String()
			assert.Contains(t, body, tc.expectedResponseSubstring, "unexpected response body")

			// Assert that all expectations were met.
			mockAccClient.AssertExpectations(t)
			mockStoreClient.AssertExpectations(t)
		})
	}
}
