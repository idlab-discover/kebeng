package client

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockStoreClient is a mock implementation of the StoreClientInterface.
type MockStoreClient struct {
	mock.Mock
}

type MockUnscannedUploadClient struct {
	mock.Mock
	grpc.ClientStream
}

func (m *MockUnscannedUploadClient) Recv() (*proto.UnscannedUploadCompleteResponse, error) {
	args := m.Called()
	if resp := args.Get(0); resp != nil {
		return resp.(*proto.UnscannedUploadCompleteResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUnscannedUploadClient) CloseAndRecv() (*proto.UnscannedUploadCompleteResponse, error) {
	args := m.Called()
	if resp := args.Get(0); resp != nil {
		return resp.(*proto.UnscannedUploadCompleteResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

var _ StoreClientInterface = (*MockStoreClient)(nil)

// Close mocks the Close function.
func (m *MockStoreClient) Close() {
	m.Called()
}

// RegisterSnapName mocks the RegisterSnapName function.
func (m *MockStoreClient) RegisterSnapName(snapName string, snapType string, confinement string, base string, isPrivate bool, status string, price float64, storeName string, iconUrl string, dryRun bool, accountId uuid.UUID) *proto.RegisterSnapNameResponse {
	args := m.Called(snapName, snapType, confinement, base, isPrivate, status, price, storeName, iconUrl, dryRun, accountId)
	if resp, ok := args.Get(0).(*proto.RegisterSnapNameResponse); ok {
		return resp
	}
	return nil
}

// GetEntries mocks the GetEntries function.
func (m *MockStoreClient) GetEntries(entries *proto.GetEntriesRequest) *proto.GetEntriesResponse {
	args := m.Called(entries)
	if resp, ok := args.Get(0).(*proto.GetEntriesResponse); ok {
		return resp
	}
	return nil
}

// GetRevisions mocks the GetRevisions function.
func (m *MockStoreClient) GetRevisions(revisions *proto.GetRevisionsRequest) *proto.GetRevisionsResponse {
	args := m.Called(revisions)
	if resp, ok := args.Get(0).(*proto.GetRevisionsResponse); ok {
		return resp
	}
	return nil
}

// GetEntriesByAccountID mocks the GetEntriesByAccountID function.
func (m *MockStoreClient) GetEntriesByAccountID(accountID string) *proto.GetEntriesResponse {
	args := m.Called(accountID)
	if resp, ok := args.Get(0).(*proto.GetEntriesResponse); ok {
		return resp
	}
	return nil
}

// GetRevisionsByEntryIds mocks the GetRevisionsByEntryIds function.
func (m *MockStoreClient) GetRevisionsByEntryIds(entryIds *proto.GetRevisionsByEntryIdRequests) *proto.GetRevisionsByEntryIdResponses {
	args := m.Called(entryIds)
	if resp, ok := args.Get(0).(*proto.GetRevisionsByEntryIdResponses); ok {
		return resp
	}
	return nil
}

// GetLatestRevisionByTrackAndChannel mocks the GetLatestRevisionByTrackAndChannel function.
func (m *MockStoreClient) GetLatestRevisionByTrackAndChannel(snapName, track, channel string) *proto.GetRevisionResponse {
	args := m.Called(snapName, track, channel)
	if resp, ok := args.Get(0).(*proto.GetRevisionResponse); ok {
		return resp
	}
	return nil
}

// SnapDownload mocks the SnapDownload function.
func (m *MockStoreClient) SnapDownload(revisionId string) *proto.SnapDownloadCompleteResponse {
	args := m.Called(revisionId)
	if resp, ok := args.Get(0).(*proto.SnapDownloadCompleteResponse); ok {
		return resp
	}
	return nil
}

// UnscannedUpload mocks the UnscannedUpload function.
func (m *MockStoreClient) UnscannedUpload(ctx context.Context, snapFile io.Reader) *proto.UnscannedUploadCompleteResponse {
	args := m.Called(snapFile)
	if resp, ok := args.Get(0).(*proto.UnscannedUploadCompleteResponse); ok {
		return resp
	}
	return nil
}

// AddUpload mocks the AddUpload function.
func (m *MockStoreClient) AddUpload(snapName string, entryId uuid.UUID, status string, accountId uuid.UUID, unscannedFileName string) *proto.AddUploadResponse {
	args := m.Called(snapName, entryId, status, accountId)
	if resp, ok := args.Get(0).(*proto.AddUploadResponse); ok {
		return resp
	}
	return nil
}

func (m *MockStoreClient) GetUploadStatus(uploadId string) *proto.GetUploadStatusResponse {
	args := m.Called(uploadId)
	if resp, ok := args.Get(0).(*proto.GetUploadStatusResponse); ok {
		return resp
	}
	return nil
}

func (m *MockStoreClient) AddRevision(snapName string, sha3_384_encoded string, size uint64, architectures []string, tracksAndChannels []string, unscannedFileName string) *proto.AddRevisionResponse {
	args := m.Called(snapName, sha3_384_encoded, size, architectures, tracksAndChannels, unscannedFileName)
	if resp, ok := args.Get(0).(*proto.AddRevisionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockStoreClient) GetObjectCustomMetadata(bucket string, objectKey string) *proto.GetObjectCustomMetadataResponse {
	args := m.Called(bucket, objectKey)
	if resp, ok := args.Get(0).(*proto.GetObjectCustomMetadataResponse); ok {
		return resp
	}
	return nil
}

func (m *MockStoreClient) UpdateUploadStatus(uploadId string, status string, revision uint32, el *cerror.ErrorList) *proto.UpdateUploadStatusResponse {
	args := m.Called(uploadId, status, revision, el)
	if resp, ok := args.Get(0).(*proto.UpdateUploadStatusResponse); ok {
		return resp
	}
	return nil
}
