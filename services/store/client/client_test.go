package client_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/store/client"
	"github.com/idlab-discover/kebeng/services/store/internal/logic"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrBool(b bool) *bool {
	return &b
}
func ptrString(s string) *string {
	return &s
}

func TestStoreClient_RegisterSnapName(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	mockID := uuid.New()

	tests := []struct {
		name               string
		snapName           string
		isPrivate          bool
		storeName          string
		dryRun             bool
		accountId          uuid.UUID
		expectedResp       *proto.RegisterSnapNameResponse
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name:      "Successful proto call",
			snapName:  "test_snap",
			isPrivate: false,
			storeName: "test_store",
			dryRun:    false,
			accountId: uuid.New(),
			expectedResp: &proto.RegisterSnapNameResponse{
				Id:       mockID.String(),
				SnapName: "test_snap",
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name:               "proto call returns error",
			snapName:           "test_snap",
			isPrivate:          false,
			storeName:          "test_store",
			dryRun:             false,
			accountId:          uuid.New(),
			expectedResp:       nil,
			expectedProtoError: true,
		},
		{
			name:      "response contains errors",
			snapName:  "test_snap",
			isPrivate: false,
			storeName: "test_store",
			dryRun:    false,
			expectedResp: &proto.RegisterSnapNameResponse{
				Errors: []*proto.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedProtoError {
				mockProtoClient.On("RegisterSnapName", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("RegisterSnapName", mock.Anything, mock.Anything).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.RegisterSnapName(tc.snapName, tc.isPrivate, tc.storeName, tc.dryRun, tc.accountId)
			if !tc.expectedErrors && !tc.expectedProtoError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedProtoError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}

func TestStoreClient_GetEntries(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	tests := []struct {
		name               string
		entries            *proto.GetEntriesRequest
		expectedResp       *proto.GetEntriesResponse
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name: "Successful proto call",
			entries: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{
					{
						Id:                  ptrString("test_id"),
						Name:                ptrString("test_name"),
						PreloadAssociations: nil,
					},
				},
			},
			expectedResp: &proto.GetEntriesResponse{
				Entries: []*proto.GetEntryResponse{
					{
						Id:          "test_id",
						SnapName:    "test_name",
						Confinement: ptrString("strict"),
						Base:        ptrString("core24"),
						Private:     ptrBool(false),
						// Other fields...
						Errors: []*proto.Error{},
					},
				},
				Errors: []*proto.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name: "proto call returns error",
			entries: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{
					{
						Id:                  ptrString("test_id"),
						Name:                ptrString("test_name"),
						PreloadAssociations: nil,
					},
				},
			},
			expectedResp:       nil,
			expectedErrors:     false,
			expectedProtoError: true,
		},
		{
			name: "response contains errors",
			entries: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{
					{
						Id:                  ptrString("test_id"),
						Name:                ptrString("test_name"),
						PreloadAssociations: nil,
					},
				},
			},
			expectedResp: &proto.GetEntriesResponse{
				Entries: []*proto.GetEntryResponse{
					{
						Id:          "test_id",
						SnapName:    "test_name",
						Confinement: ptrString("strict"),
						Base:        ptrString("core24"),
						Private:     ptrBool(false),
						// Other fields...
						Errors: []*proto.Error{{
							Code:    cerror.InternalServerError,
							Message: "mock error",
						}},
					},
				},
				Errors: []*proto.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedProtoError {
				mockProtoClient.On("GetEntries", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("GetEntries", mock.Anything, mock.Anything).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.GetEntries(tc.entries)
			if !tc.expectedErrors && !tc.expectedProtoError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedProtoError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}

func TestStoreClient_GetRevisions(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	tests := []struct {
		name               string
		revisions          *proto.GetRevisionsRequest
		expectedResp       *proto.GetRevisionsResponse
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name: "Successful proto call",
			revisions: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{
					{
						Id:       "test_id",
						SnapName: "test_name",
						Sequence: 1,
					},
				},
			},
			expectedResp: &proto.GetRevisionsResponse{
				Revisions: []*proto.GetRevisionResponse{
					{
						Id:             "test_id",
						SnapName:       "test_name",
						SequenceNumber: 1,
						// Other fields...
						Errors: []*proto.Error{},
					},
				},
				Errors: []*proto.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name: "proto call returns error",
			revisions: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{
					{
						Id:       "test_id",
						SnapName: "test_name",
						Sequence: 1,
					},
				},
			},
			expectedResp:       nil,
			expectedErrors:     false,
			expectedProtoError: true,
		},
		{
			name: "response contains errors",
			revisions: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{
					{
						Id:       "test_id",
						SnapName: "test_name",
						Sequence: 1,
					},
				},
			},
			expectedResp: &proto.GetRevisionsResponse{
				Revisions: []*proto.GetRevisionResponse{
					{
						Id:             "test_id",
						SnapName:       "test_name",
						SequenceNumber: 1,
						// Other fields...
						Errors: []*proto.Error{{
							Code:    cerror.InternalServerError,
							Message: "mock error",
						}},
					},
				},
				Errors: []*proto.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedProtoError {
				mockProtoClient.On("GetRevisions", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("GetRevisions", mock.Anything, mock.Anything).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.GetRevisions(tc.revisions)
			if !tc.expectedErrors && !tc.expectedProtoError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedProtoError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}

func TestStoreClient_GetEntriesByAccountID(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	tests := []struct {
		name               string
		accountID          string
		expectedResp       *proto.GetEntriesResponse
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name:      "Successful proto call",
			accountID: "test_account_id",
			expectedResp: &proto.GetEntriesResponse{
				Entries: []*proto.GetEntryResponse{
					{
						Id:          "test_id",
						SnapName:    "test_name",
						Confinement: ptrString("strict"),
						Base:        ptrString("core24"),
						Private:     ptrBool(false),
						// Other fields...
						Errors: []*proto.Error{},
					},
				},
				Errors: []*proto.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name:               "proto call returns error",
			accountID:          "test_account_id",
			expectedResp:       nil,
			expectedErrors:     false,
			expectedProtoError: true,
		},
		{
			name:      "response contains errors",
			accountID: "test_account_id",
			expectedResp: &proto.GetEntriesResponse{
				Entries: []*proto.GetEntryResponse{
					{
						Id:          "test_id",
						SnapName:    "test_name",
						Confinement: ptrString("strict"),
						Base:        ptrString("core24"),
						Private:     ptrBool(false),
						// Other fields...
						Errors: []*proto.Error{{
							Code:    cerror.InternalServerError,
							Message: "mock error",
						}},
					},
				},
				Errors: []*proto.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedProtoError {
				mockProtoClient.On("GetEntriesByAccountId", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("GetEntriesByAccountId", mock.Anything, mock.Anything).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.GetEntriesByAccountID(tc.accountID)
			if !tc.expectedErrors && !tc.expectedProtoError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedProtoError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}

func TestStoreClient_GetRevisionsByEntryIds(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	tests := []struct {
		name               string
		entryIds           *proto.GetRevisionsByEntryIdRequests
		expectedResp       *proto.GetRevisionsByEntryIdResponses
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name: "Successful proto call",
			entryIds: &proto.GetRevisionsByEntryIdRequests{
				Requests: []*proto.GetRevisionsByEntryIdRequest{
					{EntryId: "test_entry_id_1"},
					{EntryId: "test_entry_id_2"},
				},
			},
			expectedResp: &proto.GetRevisionsByEntryIdResponses{
				Responses: []*proto.GetRevisionsByEntryIdResponse{
					{
						EntryId: "test_entry_id_1",
						Revisions: []*proto.GetRevisionResponse{
							{
								Id:             "revision_id_1",
								SnapName:       "test_snap_1",
								SequenceNumber: 1,
								Errors:         []*proto.Error{},
							},
						},
						Errors: []*proto.Error{},
					},
					{
						EntryId: "test_entry_id_2",
						Revisions: []*proto.GetRevisionResponse{
							{
								Id:             "revision_id_2",
								SnapName:       "test_snap_2",
								SequenceNumber: 2,
								Errors:         []*proto.Error{},
							},
						},
						Errors: []*proto.Error{},
					},
				},
				Errors: []*proto.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name: "proto call returns error",
			entryIds: &proto.GetRevisionsByEntryIdRequests{
				Requests: []*proto.GetRevisionsByEntryIdRequest{
					{EntryId: "test_entry_id_1"},
					{EntryId: "test_entry_id_2"},
				},
			},
			expectedResp:       nil,
			expectedErrors:     false,
			expectedProtoError: true,
		},
		{
			name: "response contains errors",
			entryIds: &proto.GetRevisionsByEntryIdRequests{
				Requests: []*proto.GetRevisionsByEntryIdRequest{
					{EntryId: "test_entry_id_1"},
					{EntryId: "test_entry_id_2"},
				},
			},
			expectedResp: &proto.GetRevisionsByEntryIdResponses{
				Responses: []*proto.GetRevisionsByEntryIdResponse{
					{
						EntryId: "test_entry_id_1",
						Revisions: []*proto.GetRevisionResponse{
							{
								Id:             "revision_id_1",
								SnapName:       "test_snap_1",
								SequenceNumber: 1,
								Errors: []*proto.Error{{
									Code:    cerror.InternalServerError,
									Message: "mock error",
								}},
							},
						},
						Errors: []*proto.Error{{
							Code:    cerror.InternalServerError,
							Message: "mock error",
						}},
					},
				},
				Errors: []*proto.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedProtoError {
				mockProtoClient.On("GetRevisionsByEntryIds", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("GetRevisionsByEntryIds", mock.Anything, mock.Anything).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.GetRevisionsByEntryIds(tc.entryIds)
			if !tc.expectedErrors && !tc.expectedProtoError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedProtoError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}

func TestStoreClient_GetLatestRevision(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	tests := []struct {
		name               string
		snapName           string
		track              string
		channel            string
		expectedReq        *proto.GetLatestRevisionRequest
		expectedResp       *proto.GetRevisionResponse
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name:     "Successful proto call with all fields",
			snapName: "test_snap",
			track:    "stable",
			channel:  "edge",
			expectedReq: &proto.GetLatestRevisionRequest{
				SnapName: "test_snap",
				Track:    "stable",
				Channel:  "edge",
			},
			expectedResp: &proto.GetRevisionResponse{
				Id:             "revision_id",
				SnapName:       "test_snap",
				SequenceNumber: 1,
				Errors:         []*proto.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name:     "Successful proto call with default track and channel",
			snapName: "test_snap",
			track:    "",
			channel:  "",
			expectedReq: &proto.GetLatestRevisionRequest{
				SnapName: "test_snap",
				Track:    "latest",
				Channel:  "stable",
			},
			expectedResp: &proto.GetRevisionResponse{
				Id:             "revision_id",
				SnapName:       "test_snap",
				SequenceNumber: 1,
				Errors:         []*proto.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name:     "Missing snapName",
			snapName: "",
			track:    "stable",
			channel:  "edge",
			expectedResp: &proto.GetRevisionResponse{
				Errors: []*proto.Error{{
					Code:    cerror.MissingField,
					Message: "snapName is required",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
		{
			name:     "proto call returns error",
			snapName: "test_snap",
			track:    "stable",
			channel:  "edge",
			expectedReq: &proto.GetLatestRevisionRequest{
				SnapName: "test_snap",
				Track:    "stable",
				Channel:  "edge",
			},
			expectedResp:       nil,
			expectedErrors:     false,
			expectedProtoError: true,
		},
		{
			name:     "response contains errors",
			snapName: "test_snap",
			track:    "stable",
			channel:  "edge",
			expectedReq: &proto.GetLatestRevisionRequest{
				SnapName: "test_snap",
				Track:    "stable",
				Channel:  "edge",
			},
			expectedResp: &proto.GetRevisionResponse{
				Errors: []*proto.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.snapName == "" {
				resp := storeClient.GetLatestRevision(tc.snapName, tc.track, tc.channel)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.MissingField, resp.Errors[0].Code)
				return
			}

			if tc.expectedProtoError {
				mockProtoClient.On("GetLatestRevision", mock.Anything, tc.expectedReq).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("GetLatestRevision", mock.Anything, tc.expectedReq).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.GetLatestRevision(tc.snapName, tc.track, tc.channel)
			if !tc.expectedErrors && !tc.expectedProtoError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedProtoError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}

// Test will be needing fixing when UnscannedUpload is using streaming
func TestStoreClient_UnscannedUpload(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	tests := []struct {
		name               string
		snapFile           io.Reader
		expectedResp       *proto.UnscannedUploadResponse
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name:     "Successful proto call",
			snapFile: bytes.NewReader([]byte("mock snap file content")),
			expectedResp: &proto.UnscannedUploadResponse{
				TempFileName: "mock_filename",
				FileSize:     12345,
				Errors:       []*proto.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name:               "Error reading snap file",
			snapFile:           &errorReader{},
			expectedResp:       nil,
			expectedErrors:     true,
			expectedProtoError: false,
		},
		{
			name:               "proto call returns error",
			snapFile:           bytes.NewReader([]byte("mock snap file content")),
			expectedResp:       nil,
			expectedErrors:     false,
			expectedProtoError: true,
		},
		{
			name:     "response contains errors",
			snapFile: bytes.NewReader([]byte("mock snap file content")),
			expectedResp: &proto.UnscannedUploadResponse{
				Errors: []*proto.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := tc.snapFile.(*errorReader); ok {
				resp := storeClient.UnscannedUpload(tc.snapFile)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
				return
			}
			if tc.expectedProtoError {
				mockProtoClient.On("UnscannedUpload", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("UnscannedUpload", mock.Anything, mock.Anything).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.UnscannedUpload(tc.snapFile)
			if !tc.expectedErrors && !tc.expectedProtoError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedProtoError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}

// errorReader is a helper type to simulate an error when reading from an io.Reader.
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock read error")
}

func TestStoreClient_AddUpload(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	mockID := uuid.New()

	tests := []struct {
		name               string
		snapName           string
		entryId            uuid.UUID
		status             string
		accountId          uuid.UUID
		expectedResp       *proto.AddUploadResponse
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name:      "Successful proto call",
			snapName:  "test_snap",
			entryId:   mockID,
			status:    "pending",
			accountId: uuid.New(),
			expectedResp: &proto.AddUploadResponse{
				Id:       mockID.String(),
				SnapName: "test_snap",
				Status:   "pending",
				Errors:   []*proto.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name:               "proto call returns error",
			snapName:           "test_snap",
			entryId:            uuid.New(),
			status:             "pending",
			accountId:          uuid.New(),
			expectedResp:       nil,
			expectedProtoError: true,
		},
		{
			name:     "response contains errors",
			snapName: "test_snap",
			entryId:  uuid.New(),
			expectedResp: &proto.AddUploadResponse{
				Errors: []*proto.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedProtoError {
				mockProtoClient.On("AddUpload", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("AddUpload", mock.Anything, mock.Anything).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.AddUpload(tc.snapName, tc.entryId, tc.status, tc.accountId)
			if !tc.expectedErrors && !tc.expectedProtoError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedProtoError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}
