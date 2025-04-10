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

func (m *MockStoreServiceClient) UnscannedUpload(ctx context.Context, in *proto.UnscannedUploadRequest, opts ...grpc.CallOption) (*proto.UnscannedUploadResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.UnscannedUploadResponse), nil
}

func (m *MockStoreServiceClient) AddUpload(ctx context.Context, in *proto.AddUploadRequest, opts ...grpc.CallOption) (*proto.AddUploadResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.AddUploadResponse), nil
}

func (m *MockStoreServiceClient) GetLatestRevision(ctx context.Context, in *proto.GetLatestRevisionRequest, opts ...grpc.CallOption) (*proto.GetRevisionResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetRevisionResponse), nil
}

func (m *MockStoreServiceClient) SnapDownload(ctx context.Context, in *proto.SnapDownloadRequest, opts ...grpc.CallOption) (proto.StoreService_SnapDownloadClient, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(proto.StoreService_SnapDownloadClient), args.Error(1)
}
