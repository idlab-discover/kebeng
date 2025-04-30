package logic

import (
	"context"

	proto "github.com/idlab-discover/kebeng/services/account/proto"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockAccountServiceClient is a mock implementation of the AccountServiceClient interface.
type MockAccountServiceClient struct {
	mock.Mock
}

var _ proto.AccountServiceClient = (*MockAccountServiceClient)(nil)

// PatchAccountByEmail is implemented as follows.
func (m *MockAccountServiceClient) PatchAccountByEmail(ctx context.Context, in *proto.PatchAccountByEmailRequest, opts ...grpc.CallOption) (*proto.PatchAccountByEmailResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.PatchAccountByEmailResponse), args.Error(1)
}

// AddAccount mocks the AddAccount method.
func (m *MockAccountServiceClient) AddAccount(ctx context.Context, in *proto.AddAccountRequest, opts ...grpc.CallOption) (*proto.AccountResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.AccountResponse), args.Error(1)
}

// UpdateAccount mocks the UpdateAccount method.
func (m *MockAccountServiceClient) UpdateAccount(ctx context.Context, in *proto.UpdateAccountRequest, opts ...grpc.CallOption) (*proto.AccountResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.AccountResponse), args.Error(1)
}

// DeleteAccount mocks the DeleteAccount method.
func (m *MockAccountServiceClient) DeleteAccount(ctx context.Context, in *proto.DeleteAccountRequest, opts ...grpc.CallOption) (*proto.DeleteAccountResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.DeleteAccountResponse), args.Error(1)
}

// GetAccountByEmail mocks the GetAccountByEmail method.
func (m *MockAccountServiceClient) GetAccountByEmail(ctx context.Context, in *proto.GetAccountByEmailRequest, opts ...grpc.CallOption) (*proto.AccountResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.AccountResponse), args.Error(1)
}

// GetAccountByID mocks the GetAccountByID method.
func (m *MockAccountServiceClient) GetAccountByID(ctx context.Context, in *proto.GetAccountByIDRequest, opts ...grpc.CallOption) (*proto.AccountResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.AccountResponse), args.Error(1)
}

// GetAccountsByIds mocks the GetAccountsByIds method.
func (m *MockAccountServiceClient) GetAccountsByIds(ctx context.Context, in *proto.GetAccountsByIdsRequest, opts ...grpc.CallOption) (*proto.GetAccountsByIdsResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.GetAccountsByIdsResponse), args.Error(1)
}

// GetAccountByUsername mocks the GetAccountByUsername method.
func (m *MockAccountServiceClient) GetAccountByUsername(ctx context.Context, in *proto.GetAccountByUsernameRequest, opts ...grpc.CallOption) (*proto.AccountResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.AccountResponse), args.Error(1)
}

// AddKey mocks the AddKey method.
func (m *MockAccountServiceClient) AddKey(ctx context.Context, in *proto.AddKeyRequest, opts ...grpc.CallOption) (*proto.KeyResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.KeyResponse), args.Error(1)
}

// GetKeyBySHA3384 mocks the GetKeyBySHA3384 method.
func (m *MockAccountServiceClient) GetKeyBySHA3384(ctx context.Context, in *proto.GetKeyBySHA3384Request, opts ...grpc.CallOption) (*proto.KeyResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.KeyResponse), args.Error(1)
}

// GetKeysByAccountId mocks the GetKeysByAccountId method.
func (m *MockAccountServiceClient) GetKeysByAccountId(ctx context.Context, in *proto.GetKeysByAccountIdRequest, opts ...grpc.CallOption) (*proto.KeysResponse, error) {
	args := m.Called(ctx, in)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*proto.KeysResponse), args.Error(1)
}
