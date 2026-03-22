package logic

import (
	"context"

	proto "github.com/idlab-discover/kebeng/services/store/proto"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockStoreServiceClient struct {
	mock.Mock
}

type MockSnapDownloadClient struct {
	mock.Mock
	grpc.ClientStream
}

type MockUnscannedUploadClient struct {
	mock.Mock
	grpc.ClientStream
}

func (m *MockSnapDownloadClient) Recv() (*proto.SnapDownloadResponse, error) {
	args := m.Called()
	if resp := args.Get(0); resp != nil {
		return resp.(*proto.SnapDownloadResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUnscannedUploadClient) Recv() (*proto.UnscannedUploadCompleteResponse, error) {
	args := m.Called()
	if resp := args.Get(0); resp != nil {
		return resp.(*proto.UnscannedUploadCompleteResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUnscannedUploadClient) Send(msg *proto.UnscannedUploadRequest) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockUnscannedUploadClient) CloseAndRecv() (*proto.UnscannedUploadCompleteResponse, error) {
	args := m.Called()
	if resp := args.Get(0); resp != nil {
		return resp.(*proto.UnscannedUploadCompleteResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStoreServiceClient) RegisterSnapName(ctx context.Context, in *proto.RegisterSnapNameRequest, opts ...grpc.CallOption) (*proto.RegisterSnapNameResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.RegisterSnapNameResponse), nil
}

func (m *MockStoreServiceClient) GetEntries(ctx context.Context, in *proto.GetEntriesRequest, opts ...grpc.CallOption) (*proto.GetEntriesResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetEntriesResponse), nil
}

func (m *MockStoreServiceClient) GetEntryById(ctx context.Context, in *proto.GetEntryRequest, opts ...grpc.CallOption) (*proto.GetEntryResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetEntryResponse), args.Error(1)
}

func (m *MockStoreServiceClient) GetEntriesByQuery(ctx context.Context, in *proto.GetEntriesByQueryRequest, opts ...grpc.CallOption) (*proto.GetEntriesResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetEntriesResponse), args.Error(1)
}

func (m *MockStoreServiceClient) GetEntryByName(ctx context.Context, in *proto.GetEntryRequest, opts ...grpc.CallOption) (*proto.GetEntryResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetEntryResponse), args.Error(1)
}

func (m *MockStoreServiceClient) GetRevisions(ctx context.Context, in *proto.GetRevisionsRequest, opts ...grpc.CallOption) (*proto.GetRevisionsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetRevisionsResponse), nil
}

func (m *MockStoreServiceClient) GetEntriesByAccountId(ctx context.Context, in *proto.GetEntriesByAccountIdRequest, opts ...grpc.CallOption) (*proto.GetEntriesResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetEntriesResponse), nil
}

func (m *MockStoreServiceClient) GetRevisionsByEntryIds(ctx context.Context, in *proto.GetRevisionsByEntryIdRequests, opts ...grpc.CallOption) (*proto.GetRevisionsByEntryIdResponses, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetRevisionsByEntryIdResponses), nil
}

func (m *MockStoreServiceClient) GetRevisionByNameAndSequence(ctx context.Context, in *proto.GetRevisionRequest, opts ...grpc.CallOption) (*proto.GetRevisionResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetRevisionResponse), args.Error(1)
}

func (m *MockStoreServiceClient) UnscannedUpload(ctx context.Context, opts ...grpc.CallOption) (proto.StoreService_UnscannedUploadClient, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(proto.StoreService_UnscannedUploadClient), nil
}

func (m *MockStoreServiceClient) AddUpload(ctx context.Context, in *proto.AddUploadRequest, opts ...grpc.CallOption) (*proto.AddUploadResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.AddUploadResponse), nil
}

func (m *MockStoreServiceClient) GetLatestRevisionByTrackAndChannel(ctx context.Context, in *proto.GetLatestRevisionRequest, opts ...grpc.CallOption) (*proto.GetRevisionResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetRevisionResponse), nil
}

func (m *MockStoreServiceClient) GetRevisionById(ctx context.Context, in *proto.GetRevisionRequest, opts ...grpc.CallOption) (*proto.GetRevisionResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetRevisionResponse), args.Error(1)
}

func (m *MockStoreServiceClient) SnapDownload(ctx context.Context, in *proto.SnapDownloadRequest, opts ...grpc.CallOption) (proto.StoreService_SnapDownloadClient, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(proto.StoreService_SnapDownloadClient), args.Error(1)
}

func (m *MockStoreServiceClient) GetUploadStatus(ctx context.Context, in *proto.GetUploadStatusRequest, opts ...grpc.CallOption) (*proto.GetUploadStatusResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetUploadStatusResponse), nil
}

func (m *MockStoreServiceClient) AddRevision(ctx context.Context, in *proto.AddRevisionRequest, opts ...grpc.CallOption) (*proto.AddRevisionResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.AddRevisionResponse), nil
}

func (m *MockStoreServiceClient) GetObjectCustomMetadata(ctx context.Context, in *proto.GetObjectCustomMetadataRequest, opts ...grpc.CallOption) (*proto.GetObjectCustomMetadataResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetObjectCustomMetadataResponse), nil
}

func (m *MockStoreServiceClient) UpdateUploadStatus(ctx context.Context, in *proto.UpdateUploadStatusRequest, opts ...grpc.CallOption) (*proto.UpdateUploadStatusResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.UpdateUploadStatusResponse), nil
}

func (m *MockStoreServiceClient) UpdateSnapEntryWithMetadata(ctx context.Context, in *proto.UpdateSnapEntryWithMetadataRequest, opts ...grpc.CallOption) (*proto.UpdateEntryResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.UpdateEntryResponse), nil
}
