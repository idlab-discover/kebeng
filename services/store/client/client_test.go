package client_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/idlab-discover/kebeng/services/store/client"
	"github.com/idlab-discover/kebeng/services/store/internal/logic"
	proto "github.com/idlab-discover/kebeng/services/store/proto"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
		snapType           string
		confinement        string
		base               string
		isPrivate          bool
		status             string
		price              float64
		storeName          string
		iconUrl            string
		dryRun             bool
		accountId          uuid.UUID
		expectedResp       *proto.RegisterSnapNameResponse
		expectedErrors     bool
		expectedProtoError bool
		NoCall             bool
	}{
		{
			name:        "Successful proto call",
			snapName:    "test-snap",
			snapType:    "app",
			confinement: "strict",
			base:        "core24",
			isPrivate:   false,
			status:      "active",
			price:       9.99,
			storeName:   "test_store",
			iconUrl:     "http://example.com/icon.png",
			dryRun:      false,
			accountId:   uuid.New(),
			expectedResp: &proto.RegisterSnapNameResponse{
				Id:       mockID.String(),
				SnapName: "test_snap",
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name:               "proto call returns error",
			snapName:           "test-snap",
			snapType:           "app",
			confinement:        "strict",
			base:               "core24",
			isPrivate:          false,
			status:             "active",
			price:              9.99,
			storeName:          "test_store",
			iconUrl:            "http://example.com/icon.png",
			dryRun:             false,
			accountId:          uuid.New(),
			expectedResp:       nil,
			expectedProtoError: true,
		},
		{
			name:        "response contains errors",
			snapName:    "test-snap",
			snapType:    "app",
			confinement: "strict",
			base:        "core24",
			isPrivate:   false,
			status:      "active",
			price:       9.99,
			storeName:   "test_store",
			iconUrl:     "http://example.com/icon.png",
			dryRun:      false,
			expectedResp: &proto.RegisterSnapNameResponse{
				Errors: []*cerrorpb.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
		{
			name:        "invalid snap name",
			snapName:    "snap.snap",
			snapType:    "app",
			confinement: "strict",
			base:        "core24",
			isPrivate:   false,
			status:      "active",
			price:       9.99,
			storeName:   "test_store",
			iconUrl:     "http://example.com/icon.png",
			dryRun:      false,
			expectedResp: &proto.RegisterSnapNameResponse{
				Errors: []*cerrorpb.Error{{
					Code:    cerror.InvalidField,
					Message: "snap name is invalid, it should only have ASCII lowercase letters, numbers, and hyphens, and must have at least one letter: snap.snap",
				},
				},
			},
			expectedErrors:     true,
			expectedProtoError: false,
			NoCall:             true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedProtoError && !tc.NoCall {
				mockProtoClient.On("RegisterSnapName", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else if !tc.NoCall {
				mockProtoClient.On("RegisterSnapName", mock.Anything, mock.Anything).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.RegisterSnapName(tc.snapName, tc.snapType, tc.confinement, tc.base, tc.isPrivate, tc.status, tc.price, tc.storeName, tc.iconUrl, tc.dryRun, tc.accountId)
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
						Confinement: "strict",
						Base:        "core24",
						Private:     false,
						// Other fields...
						Errors: []*cerrorpb.Error{},
					},
				},
				Errors: []*cerrorpb.Error{},
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
						Confinement: "strict",
						Base:        "core24",
						Private:     false,
						// Other fields...
						Errors: []*cerrorpb.Error{{
							Code:    cerror.InternalServerError,
							Message: "mock error",
						}},
					},
				},
				Errors: []*cerrorpb.Error{{
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
						Errors: []*cerrorpb.Error{},
					},
				},
				Errors: []*cerrorpb.Error{},
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
						Errors: []*cerrorpb.Error{{
							Code:    cerror.InternalServerError,
							Message: "mock error",
						}},
					},
				},
				Errors: []*cerrorpb.Error{{
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
						Confinement: "strict",
						Base:        "core24",
						Private:     false,
						// Other fields...
						Errors: []*cerrorpb.Error{},
					},
				},
				Errors: []*cerrorpb.Error{},
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
						Confinement: "strict",
						Base:        "core24",
						Private:     false,
						// Other fields...
						Errors: []*cerrorpb.Error{{
							Code:    cerror.InternalServerError,
							Message: "mock error",
						}},
					},
				},
				Errors: []*cerrorpb.Error{{
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
								Errors:         []*cerrorpb.Error{},
							},
						},
						Errors: []*cerrorpb.Error{},
					},
					{
						EntryId: "test_entry_id_2",
						Revisions: []*proto.GetRevisionResponse{
							{
								Id:             "revision_id_2",
								SnapName:       "test_snap_2",
								SequenceNumber: 2,
								Errors:         []*cerrorpb.Error{},
							},
						},
						Errors: []*cerrorpb.Error{},
					},
				},
				Errors: []*cerrorpb.Error{},
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
								Errors: []*cerrorpb.Error{{
									Code:    cerror.InternalServerError,
									Message: "mock error",
								}},
							},
						},
						Errors: []*cerrorpb.Error{{
							Code:    cerror.InternalServerError,
							Message: "mock error",
						}},
					},
				},
				Errors: []*cerrorpb.Error{{
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
				Errors:         []*cerrorpb.Error{},
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
				Errors:         []*cerrorpb.Error{},
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
				Errors: []*cerrorpb.Error{{
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
				Errors: []*cerrorpb.Error{{
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
				resp := storeClient.GetLatestRevisionByTrackAndChannel(tc.snapName, tc.track, tc.channel)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.MissingField, resp.Errors[0].Code)
				return
			}

			if tc.expectedProtoError {
				mockProtoClient.On("GetLatestRevisionByTrackAndChannel", mock.Anything, tc.expectedReq).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("GetLatestRevisionByTrackAndChannel", mock.Anything, tc.expectedReq).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.GetLatestRevisionByTrackAndChannel(tc.snapName, tc.track, tc.channel)
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
		name                   string
		fileName               string
		mockStream             func() proto.StoreService_UnscannedUploadClient
		expectedResp           *proto.UnscannedUploadCompleteResponse
		expectedErrors         bool
		expectedStreamError    bool
		expectedErrorAndStream bool
	}{
		{
			name:     "Successful streaming upload",
			fileName: "test_file",
			mockStream: func() proto.StoreService_UnscannedUploadClient {
				mockStream := new(logic.MockUnscannedUploadClient)
				mockStream.On("Send", mock.Anything).Return(nil).Once()
				mockStream.On("CloseAndRecv").Return(&proto.UnscannedUploadCompleteResponse{
					TempFileName: "test_file",
					Size:         12345,
					Errors:       []*cerrorpb.Error{},
				}, nil).Once()
				return mockStream
			},
			expectedResp: &proto.UnscannedUploadCompleteResponse{
				TempFileName: "test_file",
				Size:         12345,
				Errors:       []*cerrorpb.Error{},
			},
			expectedErrors:         false,
			expectedStreamError:    false,
			expectedErrorAndStream: false,
		},
		{
			name: "Error setting up stream",
			expectedResp: &proto.UnscannedUploadCompleteResponse{
				Errors: []*cerrorpb.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:         true,
			expectedStreamError:    false,
			expectedErrorAndStream: true,
		},
		{
			name: "Error sending data",
			mockStream: func() proto.StoreService_UnscannedUploadClient {
				mockStream := new(logic.MockUnscannedUploadClient)
				mockStream.On("Send", mock.Anything).Return(errors.New("")).Once()
				mockStream.On("CloseAndRecv").Return(&proto.UnscannedUploadCompleteResponse{
					TempFileName: "test_file",
					Size:         12345,
					Errors: []*cerrorpb.Error{
						{
							Code:    cerror.InternalServerError,
							Message: "mock error",
						},
					},
				}, nil).Once()
				return mockStream
			},
			expectedResp: &proto.UnscannedUploadCompleteResponse{
				Errors: []*cerrorpb.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:         true,
			expectedStreamError:    false,
			expectedErrorAndStream: false,
		},
		{
			name: "Error closing stream",
			mockStream: func() proto.StoreService_UnscannedUploadClient {
				mockStream := new(logic.MockUnscannedUploadClient)
				mockStream.On("Send", mock.Anything).Return(nil).Once()
				mockStream.On("CloseAndRecv").Return(nil, errors.New("")).Once()
				return mockStream
			},
			expectedResp: &proto.UnscannedUploadCompleteResponse{
				Errors: []*cerrorpb.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:         true,
			expectedStreamError:    false,
			expectedErrorAndStream: false,
		},
	}
	data := []byte("test data")
	reader := bytes.NewReader(data)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedErrorAndStream {
				mockProtoClient.On("UnscannedUpload", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("UnscannedUpload", mock.Anything, mock.Anything).Return(tc.mockStream(), nil).Once()
			}

			ctx := context.Background()
			resp := storeClient.UnscannedUpload(ctx, reader, "test", false)
			if !tc.expectedErrors && !tc.expectedStreamError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedStreamError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}

func TestStoreClient_AddUpload(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	mockID := uuid.New()

	tests := []struct {
		name               string
		entryId            uuid.UUID
		accountId          uuid.UUID
		snapName           string
		status             string
		unscannedFileName  string
		revision           uint32
		expectedResp       *proto.AddUploadResponse
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name:              "Successful proto call",
			entryId:           mockID,
			accountId:         uuid.New(),
			snapName:          "test_snap",
			status:            "pending",
			unscannedFileName: "test_file",
			revision:          1,
			expectedResp: &proto.AddUploadResponse{
				Id:       mockID.String(),
				SnapName: "test_snap",
				Status:   "pending",
				Errors:   []*cerrorpb.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name:               "proto call returns error",
			entryId:            uuid.New(),
			accountId:          uuid.New(),
			snapName:           "test_snap",
			status:             "pending",
			unscannedFileName:  "test_file",
			revision:           1,
			expectedResp:       nil,
			expectedProtoError: true,
		},
		{
			name:              "response contains errors",
			entryId:           uuid.New(),
			accountId:         uuid.New(),
			snapName:          "test_snap",
			status:            "pending",
			unscannedFileName: "test_file",
			revision:          1,
			expectedResp: &proto.AddUploadResponse{
				Errors: []*cerrorpb.Error{{
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

			resp := storeClient.AddUpload(tc.entryId, tc.accountId, tc.snapName, tc.status, tc.unscannedFileName, tc.revision)
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
