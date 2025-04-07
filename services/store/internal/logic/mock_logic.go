package logic

import (
	"context"

	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockAccountServiceClient struct {
	mock.Mock
}

func (m *MockAccountServiceClient) RegisterSnapName(ctx context.Context, in *proto.RegisterSnapNameRequest, opts ...grpc.CallOption) (*proto.RegisterSnapNameResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.RegisterSnapNameResponse), args.Error(1)
}

func (m *MockAccountServiceClient) GetEntries(ctx context.Context, in *proto.GetEntriesRequest, opts ...grpc.CallOption) (*proto.GetEntriesResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetEntriesResponse), args.Error(1)
}

func (m *MockAccountServiceClient) GetEntryById(ctx context.Context, in *proto.GetEntryRequest, opts ...grpc.CallOption) (*proto.GetEntryResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetEntryResponse), args.Error(1)
}

func (m *MockAccountServiceClient) GetEntryByName(ctx context.Context, in *proto.GetEntryRequest, opts ...grpc.CallOption) (*proto.GetEntryResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetEntryResponse), args.Error(1)
}

func (m *MockAccountServiceClient) GetRevisions(ctx context.Context, in *proto.GetRevisionsRequest, opts ...grpc.CallOption) (*proto.GetRevisionsResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetRevisionsResponse), args.Error(1)
}

func (m *MockAccountServiceClient) GetEntriesByAccountId(ctx context.Context, in *proto.GetEntriesByAccountIdRequest, opts ...grpc.CallOption) (*proto.GetEntriesResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetEntriesResponse), args.Error(1)
}

func (m *MockAccountServiceClient) GetRevisionsByEntryIds(ctx context.Context, in *proto.GetRevisionsByEntryIdRequests, opts ...grpc.CallOption) (*proto.GetRevisionsByEntryIdResponses, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetRevisionsByEntryIdResponses), args.Error(1)
}

func (m *MockAccountServiceClient) GetRevisionByNameAndSequence(ctx context.Context, in *proto.GetRevisionRequest, opts ...grpc.CallOption) (*proto.GetRevisionResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetRevisionResponse), args.Error(1)
}

func (m *MockAccountServiceClient) UnscannedUpload(ctx context.Context, in *proto.UnscannedUploadRequest, opts ...grpc.CallOption) (*proto.UnscannedUploadResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.UnscannedUploadResponse), args.Error(1)
}

func (m *MockAccountServiceClient) AddUpload(ctx context.Context, in *proto.AddUploadRequest, opts ...grpc.CallOption) (*proto.AddUploadResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.AddUploadResponse), args.Error(1)
}
