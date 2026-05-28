package snap

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/idlab-discover/kebeng/services/gateway/internal/config"
	"github.com/idlab-discover/kebeng/services/gateway/internal/model"
	"github.com/idlab-discover/kebeng/services/gateway/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	accClient "github.com/idlab-discover/kebeng/services/account/client"
	accountpb "github.com/idlab-discover/kebeng/services/account/proto"
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
		Confinement: strict,
		Type:        types,
		Base:        base,
	}
	// Dummy latest revision.
	dummyRev := &storepb.GetRevisionResponse{
		Id:              "rev1",
		Architectures:   []string{"amd64"},
		Sha3_384Encoded: "sha384hash",
		Size:            1000,
		Version:         "v1",
		SequenceNumber:  1,
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
		On("GetLatestRevisionByTrackAndChannel", "snap1", "latest", "stable").
		Return(dummyRev).Once()

	// For publisher: AccountClient.GetAccountByID.
	// Return the dummy account response using the proper type from accountProto.
	mockAccClient.
		On("GetAccountByID", "pub1").
		Return(&accountpb.AccountResponse{
			Id:       "pub1",
			Username: "publisher",
			Errors:   []*cerrorpb.Error{},
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

func TestRefreshSnapHandler_DownloadAction_NoLatestRevision(t *testing.T) {
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
		Confinement: strict,
		Type:        types,
		Base:        base,
	}
	dummyRev := &storepb.GetRevisionResponse{
		Errors: []*cerrorpb.Error{
			{Code: cerror.ResourceNotFound, Message: "latest revision not found"},
		},
	}
	// Expectation: getLatestRevisionByEntryName calls StoreClient.GetEntries.
	mockStoreClient.
		On("GetEntries", mock.Anything).
		Return(&storepb.GetEntriesResponse{
			Entries: []*storepb.GetEntryResponse{dummyEntry},
			Errors:  nil,
		}).Once()

	// return nil for GetLatestRevision to simulate no latest revision.
	mockStoreClient.
		On("GetLatestRevisionByTrackAndChannel", "snap1", "latest", "stable").
		Return(dummyRev).Once()

	// accountID doesn't have to be mocked doesn't get there

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
	assert.Equal(t, http.StatusNotFound, w.Code)
	// Expect to see the result "download" in the response JSON.
	assert.Contains(t, w.Body.String(), "latest revision not found")
	mockStoreClient.AssertExpectations(t)
	mockAccClient.AssertExpectations(t)
}

func TestRefreshInstallOrDownload_CohortKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStoreClient := new(storeClient.MockStoreClient)
	mockAccClient := new(accClient.MockAccountClient)

	dummyEntry := &storepb.GetEntryResponse{
		Id:          "entry1",
		SnapName:    "snap1",
		PublisherId: "pub1",
		Confinement: "strict",
		Type:        "app",
		Base:        "core20",
	}
	dummyRev := &storepb.GetRevisionResponse{
		Id:              "rev1",
		Architectures:   []string{"amd64"},
		Sha3_384Encoded: "c2hhMzg0aGFzaA",
		Size:            1000,
		Version:         "v1",
		SequenceNumber:  1,
	}

	// getLatestRevisionByEntryName -> GetEntries + GetLatestRevisionByTrackAndChannel
	mockStoreClient.
		On("GetEntries", mock.Anything).
		Return(&storepb.GetEntriesResponse{Entries: []*storepb.GetEntryResponse{dummyEntry}}).Once()
	mockStoreClient.
		On("GetLatestRevisionByTrackAndChannel", "snap1", "latest", "stable").
		Return(dummyRev).Once()

	mockStoreClient.
		On("GetLatestRevisionBeforeDateById", mock.Anything, "entry1").
		Return(dummyRev).Once()

	mockAccClient.
		On("GetAccountByID", "pub1").
		Return(&accountpb.AccountResponse{Id: "pub1", Username: "publisher"}).Once()

	baseHandler := util.NewBaseHandler(mockAccClient, mockStoreClient, nil, nil)
	baseHandler.Config = &config.Config{StoreUrl: "https://store.example.com", CohortSigningKey: "test-key"}
	handler := &Handler{BaseHandler: baseHandler}

	// build a valid signed cohort key for entry1
	ckey := model.CohortKey{Version: 1, SnapID: "entry1", CreatedAt: time.Now().Add(-24 * time.Hour)}
	signedKey, err := handler.signCohortKey(ckey)
	assert.NoError(t, err)
	cohortKeyStr := cohortKeyToString(signedKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqJSON := fmt.Sprintf(`{
		"actions": [{
			"action": "download",
			"name": "snap1",
			"channel": "stable",
			"instance-key": "instance-123",
			"cohort-key": "%s"
		}]
	}`, cohortKeyStr)
	c.Request = httptest.NewRequest("POST", "/refreshSnap", bytes.NewBufferString(reqJSON))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RefreshSnap(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "download")
	mockStoreClient.AssertExpectations(t)
	mockAccClient.AssertExpectations(t)
}

// ------------------
// DownloadSnap
// ------------------

func TestDownloadSnapHandler_MissingRevisionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// We don't even need a real store client here, because the handler
	// returns early.
	handler := &Handler{
		BaseHandler: util.NewBaseHandler(nil, nil, nil, nil),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/downloadSnap", nil)

	handler.DownloadSnap(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "revision_id is required")
}

func TestDownloadSnapHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// --- build a mock stream that speaks our protocol ---
	mockStream := new(storeClient.MockSnapDownloadClient)

	// First, the server sends an “initial” payload (we ignore it in the body).
	mockStream.
		On("Recv").
		Return(&storepb.SnapDownloadResponse{
			Payload: &storepb.SnapDownloadResponse_Initial{
				Initial: &storepb.InitialDownloadResponse{
					Revision: &storepb.GetRevisionResponse{
						Id:             "rev1",
						SnapName:       "snap1",
						SequenceNumber: 1,
					},
				},
			},
		}, nil).
		Once()

	// Next, it sends an actual data chunk
	dataChunk := []byte("hello world")
	mockStream.
		On("Recv").
		Return(&storepb.SnapDownloadResponse{
			Payload: &storepb.SnapDownloadResponse_Data{
				Data: &storepb.DataChunk{Chunk: dataChunk},
			},
		}, nil).
		Once()

	// Then EOF
	mockStream.
		On("Recv").
		Return((*storepb.SnapDownloadResponse)(nil), io.EOF).
		Once()

	// And the handler defers a CloseSend on the stream
	mockStream.
		On("CloseSend").
		Return(nil).
		Once()

	// --- mock the store client to return our stream ---
	mockStore := new(storeClient.MockStoreClient)
	mockStore.
		On("SnapDownloadStream", "rev1").
		Return(mockStream, nil).
		Once()

	// Wire up the handler
	base := util.NewBaseHandler(nil, mockStore, nil, nil)
	handler := &Handler{BaseHandler: base}

	// Exercise the HTTP handler
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// inject the path parameter
	c.Params = append(c.Params, gin.Param{Key: "revision_id", Value: "rev1"})

	handler.DownloadSnap(c)

	// ---- assertions ----
	assert.Equal(t, http.StatusOK, w.Code)

	// Content‐Disposition should be exactly revision_id+".snap"
	hdr := w.Header().Get("Content-Disposition")
	assert.Equal(t, `attachment; filename="rev1.snap"`, hdr)

	// We wrote exactly the data chunk
	assert.Equal(t, string(dataChunk), w.Body.String())

	mockStream.AssertExpectations(t)
	mockStore.AssertExpectations(t)
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
		On("RegisterSnapName", "test-snap", mock.Anything, mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything, mock.Anything, false, mock.AnythingOfType("uuid.UUID")).
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
		On("RegisterSnapName", "snap1", mock.Anything, mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything, mock.Anything, true, mock.AnythingOfType("uuid.UUID")).
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
	dummyResp := &storepb.UnscannedUploadCompleteResponse{
		Errors:       nil,
		TempFileName: "temp123",
	}
	mockStoreClient.
		On("UnscannedUpload", mock.Anything, "dummy.snap", false).
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

// ------------------
// FindSnap
// ------------------

// TestFindSnaps tests a valid query that yields no result
func TestFindSnaps(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStoreClient := new(storeClient.MockStoreClient)
	mockResp := &storepb.GetEntriesResponse{
		Entries: nil,
		Errors: nil,
	}
	mockStoreClient.On("GetEntriesByQuery",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("[]string"),
			mock.AnythingOfType("[]string"),
			mock.AnythingOfType("[]string"),
			mock.AnythingOfType("[]string"),
			mock.AnythingOfType("bool"),
			mock.AnythingOfType("string"),
		).
		Return(mockResp).
		Once()
	handler := &Handler{BaseHandler: util.NewBaseHandler(
		nil,
		mockStoreClient,
		nil,
		nil,
	)}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	obscureSearchQuery := "obscure_value_that_is_not_present_in_db_989Y789"
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/v2/snaps/find?q=%s&architecture=amd64&confinement=strict,classic", obscureSearchQuery), strings.NewReader("{}"))

	handler.FindSnaps(c)

	// The query should succeed (status code OK)
	assert.Equal(t, http.StatusOK, w.Code)
	// The query result should be empty (no matching snaps)
	assert.Equal(t, "{\"results\":[]}", w.Body.String())
	mockStoreClient.AssertExpectations(t)
}

// TestFindSnaps_EmptyQuery tests an empty query
func TestFindSnaps_EmptyQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := Handler{BaseHandler: util.NewBaseHandler(
		nil,
		nil,
		nil,
		nil,
	)}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("GET", "/v2/snaps/find", strings.NewReader("{}"))

	handler.FindSnaps(c)

	// The query should succeed
	assert.Equal(t, http.StatusOK, w.Code)
	// The body should contain an empty results string (for now)
	assert.Contains(t, w.Body.String(), "\"results\":[]")
	
	// TODO: Once features snaps are tracked, this test should no longer return an empty result set
	// But a list of top featured snaps
}

// ------------------
// Helper function isChannel
// ------------------

func TestIsChannel(t *testing.T) {
	assert.True(t, isChannel("stable"))
	assert.True(t, isChannel("beta"))
	assert.False(t, isChannel("foo"))
}
