package client

import (
	"io"

	"github.com/google/uuid"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/stretchr/testify/mock"
)

// MockStoreClient is a mock implementation of the StoreClientInterface.
type MockStoreClient struct {
	mock.Mock
}

var _ StoreClientInterface = (*MockStoreClient)(nil)

// Close mocks the Close function.
func (m *MockStoreClient) Close() {
	m.Called()
}

// UploadSnap mocks the UploadSnap function.
func (m *MockStoreClient) UploadSnap(name string, type_name string, confinement string, base string, file []byte) *proto.UploadSnapResponse {
	args := m.Called(name, type_name, confinement, base, file)
	if resp, ok := args.Get(0).(*proto.UploadSnapResponse); ok {
		return resp
	}
	return nil
}

// RegisterSnapName mocks the RegisterSnapName function.
func (m *MockStoreClient) RegisterSnapName(snapName string, isPrivate bool, storeName string, dryRun bool, accountId uuid.UUID) *proto.RegisterSnapNameResponse {
	args := m.Called(snapName, isPrivate, storeName, dryRun, accountId)
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

// GetLatestRevision mocks the GetLatestRevision function.
func (m *MockStoreClient) GetLatestRevision(snapName, track, channel string) *proto.GetRevisionResponse {
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
func (m *MockStoreClient) UnscannedUpload(snapFile io.Reader) *proto.UnscannedUploadResponse {
	args := m.Called(snapFile)
	if resp, ok := args.Get(0).(*proto.UnscannedUploadResponse); ok {
		return resp
	}
	return nil
}

// AddUpload mocks the AddUpload function.
func (m *MockStoreClient) AddUpload(snapName string, entryId uuid.UUID, status string, accountId uuid.UUID) *proto.AddUploadResponse {
	args := m.Called(snapName, entryId, status, accountId)
	if resp, ok := args.Get(0).(*proto.AddUploadResponse); ok {
		return resp
	}
	return nil
}
