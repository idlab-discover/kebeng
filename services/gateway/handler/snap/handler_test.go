package snap

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	accountpb "github.com/idlab-discover/kebeng/services/account/proto"
	assertionClient "github.com/idlab-discover/kebeng/services/assertion/client"
	asspb "github.com/idlab-discover/kebeng/services/assertion/proto"
	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"
	storeClient "github.com/idlab-discover/kebeng/services/store/client"
	storepb "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// wrote test so that when function is update the test will fail and will have to be updated
func TestRequestStoreDeviceNonce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.RequestStoreDeviceNonce(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"nonce":"this-nonce"`)
}

func TestRequestStoreDeviceSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.RequestStoreDeviceSessions(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"macaroon":"shakamacaroon"`)
}

// ------------------
// RefreshSnap
// ------------------

func TestRefreshSnapHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Create a dummy handler.
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Send invalid JSON.
	c.Request = httptest.NewRequest("POST", "/refreshSnap", bytes.NewBufferString("{invalid_json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RefreshSnap(c)
	// Expect BadRequest due to JSON binding error.
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Syntax error")
}

func TestRefreshSnapHandler_UnknownAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Request with an unknown action.
	reqJSON := `{"actions": [{"action": "foo", "name": "snap1"}]}`
	c.Request = httptest.NewRequest("POST", "/refreshSnap", bytes.NewBufferString(reqJSON))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RefreshSnap(c)
	// Expect NotImplemented error.
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Contains(t, w.Body.String(), "Action not implemented")
}

// For valid refresh, we simulate the helper refreshSnapDownload using mocks.
func TestRefreshSnapHandler_DownloadAction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create mocks for store and account clients.
	mockStoreClient := new(storeClient.MockStoreClient)
	mockAccClient := new(accClient.MockAccountClient)

	strict := "strict"
	types := "classic"
	base := "16"

	// Prepare dummy responses for getLatestRevisionByEntryName chain.
	// Dummy entry.
	dummyEntry := &storepb.GetEntryResponse{
		Id:          "entry1",
		SnapName:    "snap1",
		PublisherId: "pub1",
		Confinement: &strict,
		Type:        &types,
		Base:        &base,
	}
	// Dummy latest revision.
	dummyRev := &storepb.GetRevisionResponse{
		Id:             "rev1",
		Architectures:  []string{"amd64"},
		Sha3_384:       "sha384hash",
		Size:           1000,
		Version:        "v1",
		SequenceNumber: 1,
	}

	// Expectation: getLatestRevisionByEntryName calls StoreClient.GetEntries.
	mockStoreClient.
		On("GetEntries", mock.Anything).
		Return(&storepb.GetEntriesResponse{
			Entries: []*storepb.GetEntryResponse{dummyEntry},
			Errors:  nil,
		}).Once()

	// And then GetLatestRevision.
	mockStoreClient.
		On("GetLatestRevision", "snap1", "latest", "stable").
		Return(dummyRev).Once()

	// For publisher: AccountClient.GetAccountByID.
	// Return the dummy account response using the proper type from accountProto.
	mockAccClient.
		On("GetAccountByID", "pub1").
		Return(&accountpb.AccountResponse{
			Id:       "pub1",
			Username: "publisher",
			Errors:   []*accountpb.Error{},
		}).Once()

	// Set up a dummy BaseHandler with our mocks and a dummy config.
	baseHandler := util.NewBaseHandler(mockAccClient, mockStoreClient, nil, nil)
	baseHandler.Config = &config.Config{
		StoreUrl: "https://store.example.com",
	}
	handler := &Handler{BaseHandler: baseHandler}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Valid JSON with action "download".
	reqJSON := `{
		"actions": [{
			"action": "download",
			"name": "snap1",
			"channel": "stable",
			"instance_key": "instance-123"
		}]
	}`
	c.Request = httptest.NewRequest("POST", "/refreshSnap", bytes.NewBufferString(reqJSON))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RefreshSnap(c)
	assert.Equal(t, http.StatusOK, w.Code)
	// Expect to see the result "download" in the response JSON.
	assert.Contains(t, w.Body.String(), "download")
	mockStoreClient.AssertExpectations(t)
	mockAccClient.AssertExpectations(t)
}

// ------------------
// DownloadSnap
// ------------------

func TestDownloadSnapHandler_MissingRevisionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No revision_id parameter.
	c.Request = httptest.NewRequest("GET", "/downloadSnap", nil)

	handler.DownloadSnap(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "revision_id is required")
}

func TestDownloadSnapHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStoreClient := new(storeClient.MockStoreClient)
	// Dummy SnapDownload response.
	dummyData := []byte("filedata")
	dummyRev := &storepb.GetRevisionResponse{
		SnapName:       "snap1",
		SequenceNumber: 1,
	}
	downloadResp := &storepb.SnapDownloadCompleteResponse{
		Data:     dummyData,
		Revision: dummyRev,
		Errors:   nil,
	}
	mockStoreClient.
		On("SnapDownload", "rev1").
		Return(downloadResp).
		Once()

	baseHandler := util.NewBaseHandler(nil, mockStoreClient, nil, nil)
	handler := &Handler{BaseHandler: baseHandler}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Set revision_id parameter.
	c.Params = append(c.Params, gin.Param{Key: "revision_id", Value: "rev1"})

	handler.DownloadSnap(c)
	// Expect success 200.
	assert.Equal(t, http.StatusOK, w.Code)
	// Check that the Content-Disposition header is set with filename.
	assert.Contains(t, c.Writer.Header().Get("Content-Disposition"), "snap1_1.snap")
	// Check that the response body equals dummyData.
	assert.Equal(t, string(dummyData), w.Body.String())
	mockStoreClient.AssertExpectations(t)
}

// ------------------
// RegisterSnapName
// ------------------

func TestRegisterSnapNameHandler_MissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Valid JSON request.
	reqJSON := `{"snap_name": "test-snap", "is_private": false, "store": "default"}`
	c.Request = httptest.NewRequest("POST", "/registerSnapName", bytes.NewBufferString(reqJSON))
	c.Request.Header.Set("Content-Type", "application/json")
	// Do not set email in context.
	handler.RegisterSnapName(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "email not found in macaroon")
}

func TestRegisterSnapNameHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAccClient := new(accClient.MockAccountClient)
	mockStoreClient := new(storeClient.MockStoreClient)
	baseHandler := util.NewBaseHandler(mockAccClient, mockStoreClient, nil, nil)
	handler := &Handler{BaseHandler: baseHandler}

	// Prepare a dummy account response.
	accountResp := &accountpb.AccountResponse{
		Id:          uuid.New().String(),
		DisplayName: "John Doe",
		Email:       "john@example.com",
		Username:    "johndoe",
	}
	mockAccClient.
		On("GetAccountByEmail", "john@example.com").
		Return(accountResp).
		Once()

	// Expect GetAccountsByIds is not used here.
	// Prepare a dummy snap name registration response.
	regResp := &storepb.RegisterSnapNameResponse{
		Id:       "550e8400-e29b-41d4-a716-446655440000",
		SnapName: "test-snap",
		Errors:   nil,
	}
	mockStoreClient.
		On("RegisterSnapName", "test-snap", false, "default", false, mock.AnythingOfType("uuid.UUID")).
		Return(regResp).
		Once()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqJSON := `{"snap_name": "test-snap", "is_private": false, "store": "default"}`
	c.Request = httptest.NewRequest("POST", "/registerSnapName", bytes.NewBufferString(reqJSON))
	c.Request.Header.Set("Content-Type", "application/json")
	// Set email in context.
	c.Set("email", "john@example.com")

	handler.RegisterSnapName(c)
	// Expect HTTP 201 Created.
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "test-snap")
	mockAccClient.AssertExpectations(t)
	mockStoreClient.AssertExpectations(t)
}

func TestRegisterSnapNameDisputeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/registerSnapNameDispute", nil)

	handler.RegisterSnapNameDispute(c)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Contains(t, w.Body.String(), "Not implemented")
}

// ------------------
// ProcessSnapBuildAssertion
// ------------------

func TestProcessSnapBuildAssertionHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/processSnapBuildAssertion/123", bytes.NewBufferString("invalid_json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ProcessSnapBuildAssertion(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProcessSnapBuildAssertionHandler_MissingSnapID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqJSON := `{"assertion": "test"}`
	c.Request = httptest.NewRequest("POST", "/processSnapBuildAssertion/", bytes.NewBufferString(reqJSON))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ProcessSnapBuildAssertion(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "snap_id is required")
}

func TestProcessSnapBuildAssertionHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock AssertionClient.
	mockAssertionClient := new(assertionClient.MockAssertionClient)

	// For this example, we create a dummy using assertion client's proto definitions:
	dummySnapAssertionResp := &asspb.SnapBuildAssertionResponse{
		AuthorityId:     "auth-001",
		Grade:           "A",
		SignKeySha3_384: "hash001",
		SnapId:          "snap-001",
		SnapSha3_384:    "shahash",
		SnapSize:        "12345",
		Timestamp:       "2025-01-01T00:00:00Z",
		Type:            "test",
		Errors:          nil,
	}

	// "ZHVtbXk=" is the base64-encoding of "dummy".
	decodedAssertion := []byte("dummy")
	mockAssertionClient.
		On("ProcessSnapBuildAssertion", decodedAssertion).
		Return(dummySnapAssertionResp).
		Once()

	// Create a BaseHandler and handler with the mock injected.
	baseHandler := util.NewBaseHandler(nil, nil, mockAssertionClient, nil)
	handler := &Handler{
		BaseHandler: baseHandler,
	}

	// Set up the Gin test context.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Use valid base64 for assertion.
	reqJSON := `{"assertion": "ZHVtbXk="}`
	c.Request = httptest.NewRequest("POST", "/processSnapBuildAssertion/123", bytes.NewBufferString(reqJSON))
	c.Request.Header.Set("Content-Type", "application/json")
	// Set the snap_id path parameter.
	c.Params = append(c.Params, gin.Param{Key: "snap_id", Value: "snap-001"})

	handler.ProcessSnapBuildAssertion(c)

	// Verify the HTTP response.
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "authority-id", "Expected response body to contain authority-id")
	mockAssertionClient.AssertExpectations(t)
}

// ------------------
// SnapPush
// ------------------

func TestSnapPushHandler_MissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqJSON := `{"name": "snap1", "dry_run": true}`
	c.Request = httptest.NewRequest("POST", "/snapPush", bytes.NewBufferString(reqJSON))
	c.Request.Header.Set("Content-Type", "application/json")
	// Do not set email.
	handler.SnapPush(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "email not found in macaroon")
}

func TestSnapPushHandler_DryRunSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAccClient := new(accClient.MockAccountClient)
	mockStoreClient := new(storeClient.MockStoreClient)
	baseHandler := util.NewBaseHandler(mockAccClient, mockStoreClient, nil, nil)
	handler := &Handler{BaseHandler: baseHandler}

	// Dummy account response.
	accountResp := &accountpb.AccountResponse{
		Id:       uuid.New().String(),
		Username: "john",
	}
	mockAccClient.
		On("GetAccountByEmail", "john@example.com").
		Return(accountResp).
		Once()

	// Dummy RegisterSnapName response.
	regResp := &storepb.RegisterSnapNameResponse{
		SnapName: "snap1",
		Id:       "entry-001",
		Errors:   nil,
	}
	mockStoreClient.
		On("RegisterSnapName", "snap1", false, "", true, mock.Anything).
		Return(regResp).
		Once()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqJSON := `{"name": "snap1", "dry_run": true}`
	c.Request = httptest.NewRequest("POST", "/snapPush", bytes.NewBufferString(reqJSON))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "john@example.com")

	handler.SnapPush(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "dry-run")
	mockAccClient.AssertExpectations(t)
	mockStoreClient.AssertExpectations(t)
}

// ------------------
// UnscannedUpload
// ------------------

func TestUnscannedUploadHandler_MissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Do not add file to form data.
	c.Request = httptest.NewRequest("POST", "/unscannedUpload", nil)

	handler.UnscannedUpload(c)
	// Expect a bad request error.
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "binary file not found")
}

func TestUnscannedUploadHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStoreClient := new(storeClient.MockStoreClient)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, mockStoreClient, nil, nil)}

	// Prepare a dummy response from UnscannedUpload.
	dummyResp := &storepb.UnscannedUploadResponse{
		Errors:       nil,
		TempFileName: "temp123",
	}
	mockStoreClient.
		On("UnscannedUpload", mock.Anything).
		Return(dummyResp).
		Once()

	// Create a multipart form request with a dummy file.
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("binary", "dummy.snap")
	assert.NoError(t, err)
	// Write some dummy data.
	_, err = part.Write([]byte("dummy data"))
	assert.NoError(t, err)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/unscannedUpload", &b)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.UnscannedUpload(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "upload_id")
	mockStoreClient.AssertExpectations(t)
}

func TestFindSnapsHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Create a handler; since FindSnaps doesn't use any clients, pass nil in the BaseHandler.
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Provide invalid JSON input.
	c.Request = httptest.NewRequest("POST", "/findSnaps", strings.NewReader("{invalid_json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.FindSnaps(c)

	// The error list should have been populated with a binding error, resulting in a Bad Request.
	assert.Equal(t, http.StatusBadRequest, w.Code)
	// Verify that the response body contains an error message. (The exact text may vary.)
	assert.Contains(t, w.Body.String(), "Syntax error", "expected error message to mention a syntax error")
}

// TestFindSnapsHandler_ValidJSON tests the valid JSON case.
func TestFindSnapsHandler_ValidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{BaseHandler: util.NewBaseHandler(nil, nil, nil, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Provide valid JSON. Since we don't know the expected fields,
	// a minimal valid JSON object is provided.
	c.Request = httptest.NewRequest("POST", "/findSnaps", strings.NewReader("{}"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.FindSnaps(c)

	// When valid JSON is received, the function does nothing and does not write any response.
	// In such a case, the response body remains empty and status defaults to 200.
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", w.Body.String(), "expected empty response when JSON is valid")
}

// ------------------
// Helper function isChannel
// ------------------

func TestIsChannel(t *testing.T) {
	assert.True(t, isChannel("stable"))
	assert.True(t, isChannel("beta"))
	assert.False(t, isChannel("foo"))
}
