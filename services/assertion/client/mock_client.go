package client

import (
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
