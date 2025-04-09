package client

import (
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"github.com/stretchr/testify/mock"
)

// MockAccountClient is a mock implementation of the AccountClient interface.
type MockAccountClient struct {
	mock.Mock
}

var _ AccountClientInterface = (*MockAccountClient)(nil)

func (m *MockAccountClient) Close() {
	m.Called()
}

func (m *MockAccountClient) CreateAccount(displayName, username, email string) *proto.AccountResponse {
	args := m.Called(displayName, username, email)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.AccountResponse)
}

func (m *MockAccountClient) UpdateAccount(id, displayName, username, email string) *proto.AccountResponse {
	args := m.Called(id, displayName, username, email)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.AccountResponse)
}

func (m *MockAccountClient) DeleteAccount(id string) *proto.DeleteAccountResponse {
	args := m.Called(id)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.DeleteAccountResponse)
}

func (m *MockAccountClient) GetAccountByEmail(email string) *proto.AccountResponse {
	args := m.Called(email)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.AccountResponse)
}

func (m *MockAccountClient) GetAccountByID(id string) *proto.AccountResponse {
	args := m.Called(id)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.AccountResponse)
}

func (m *MockAccountClient) GetAccountsByIds(ids []string) *proto.GetAccountsByIdsResponse {
	args := m.Called(ids)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.GetAccountsByIdsResponse)
}

func (m *MockAccountClient) GetAccountByUsername(username string) *proto.AccountResponse {
	args := m.Called(username)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.AccountResponse)
}

func (m *MockAccountClient) AddKey(accountEmail, keyName, sha3384, encodedPublicKey string) *proto.KeyResponse {
	args := m.Called(accountEmail, keyName, sha3384, encodedPublicKey)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.KeyResponse)
}

func (m *MockAccountClient) GetAccountKey(Sha3384 string) *proto.KeyResponse {
	args := m.Called(Sha3384)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.KeyResponse)
}

func (m *MockAccountClient) GetAccountKeysByAccountID(accountID string) *proto.KeysResponse {
	args := m.Called(accountID)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.KeysResponse)
}

func (m *MockAccountClient) PatchAccountByEmail(email, username string) *proto.PatchAccountByEmailResponse {
	args := m.Called(email, username)
	resp := args.Get(0)
	if resp == nil {
		return nil
	}
	return resp.(*proto.PatchAccountByEmailResponse)
}
