package client

import (
	"time"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/assertion/client/model"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"

	"github.com/stretchr/testify/mock"
)

// MockAssertionClient is a mock implementation of the AssertionClientInterface.
type MockAssertionClient struct {
	mock.Mock
}

var _ AssertionClientInterface = (*MockAssertionClient)(nil)

// Close mocks the Close function.
func (m *MockAssertionClient) Close() {
	m.Called()
}

// ProcessSnapBuildAssertion mocks the ProcessSnapBuildAssertion function.
func (m *MockAssertionClient) ProcessSnapBuildAssertion(assertion []byte) *proto.SnapBuildAssertionResponse {
	args := m.Called(assertion)
	if resp, ok := args.Get(0).(*proto.SnapBuildAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) AddAccountKeyAssertion(encoded_public_key, publicKeySha3_384Encoded string, accountId string, name string, since time.Time, until time.Time) *proto.AccountKeyAssertionResponse {
	args := m.Called(publicKeySha3_384Encoded, accountId, name, since, until)
	if resp, ok := args.Get(0).(*proto.AccountKeyAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) AddSnapRevisionAssertion(snapSha3_384 string, developerId string, snapEntryId string, snapRevisionSequenceNumber uint32, snapSize uint64) *proto.SnapRevisionAssertionResponse {
	args := m.Called(snapSha3_384, developerId, snapEntryId, snapRevisionSequenceNumber, snapSize)
	if resp, ok := args.Get(0).(*proto.SnapRevisionAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) GetAccountKeyAssertionByPublicKeySha(publicKeySha3_384Encoded string) *proto.AccountKeyAssertionResponse {
	args := m.Called(publicKeySha3_384Encoded)
	if resp, ok := args.Get(0).(*proto.AccountKeyAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) AddAccountAssertion(accountId, displayName, username, validation string, timestamp time.Time) *proto.AccountAssertionResponse {
	args := m.Called(accountId, displayName, username, validation, timestamp)
	if resp, ok := args.Get(0).(*proto.AccountAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) GetLatestAccountKeyAssertion(accountId string) *proto.AccountKeyAssertionResponse {
	args := m.Called(accountId)
	if resp, ok := args.Get(0).(*proto.AccountKeyAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) GetSnapRevisionAssertionBySHA3_384(snapSha3_384 string) *proto.SnapRevisionAssertionResponse {
	args := m.Called(snapSha3_384)
	if resp, ok := args.Get(0).(*proto.SnapRevisionAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) AddSnapDeclarationAssertion(snapID, snapName, publisherID string, series string, refreshControl []string, aliases []model.Alias, plugs model.Plugs, slots model.SlotMap) *proto.SnapDeclarationAssertionResponse {
	args := m.Called(snapID, snapName, publisherID, series, refreshControl, aliases, plugs, slots)
	if resp, ok := args.Get(0).(*proto.SnapDeclarationAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) GetSnapDeclarationAssertionBySnapID(snapId string) *proto.SnapDeclarationAssertionResponse {
	args := m.Called(snapId)
	if resp, ok := args.Get(0).(*proto.SnapDeclarationAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) GetAccountAssertionByAccountID(accountId string) *proto.AccountAssertionResponse {
	args := m.Called(accountId)
	if resp, ok := args.Get(0).(*proto.AccountAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) DeserializePlugMap(data string) (model.Plugs, *cerror.CustomError) {
	args := m.Called(data)
	if resp, ok := args.Get(0).(model.Plugs); ok {
		return resp, nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}
