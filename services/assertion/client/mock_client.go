package client

import (
	"time"

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

func (m *MockAssertionClient) AddAccountKeyAssertion(revisionSequenceNumber uint32, publicKeySha3_384 string, accountId string, name string, since *time.Time, until *time.Time, body []byte) *proto.AccountKeyAssertionResponse {
	args := m.Called(revisionSequenceNumber, publicKeySha3_384, accountId, name, since, until, body)
	if resp, ok := args.Get(0).(*proto.AccountKeyAssertionResponse); ok {
		return resp
	}
	return nil
}

func (m *MockAssertionClient) AddSnapRevisionAssertion(snapSha3_384 string, developerId string, snapEntryId string, snapRevisionSequenceNumber uint32, snapSize uint64, timestamp *time.Time) *proto.SnapRevisionAssertionResponse {
	args := m.Called(snapSha3_384, developerId, snapEntryId, snapRevisionSequenceNumber, snapSize, timestamp)
	if resp, ok := args.Get(0).(*proto.SnapRevisionAssertionResponse); ok {
		return resp
	}
	return nil
}
