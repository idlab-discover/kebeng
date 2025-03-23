package client

import (
	"errors"
	"testing"

	"github.com/go-playground/assert/v2"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	logic "github.com/idlab-discover/kebeng/services/account/internal/logic"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"github.com/stretchr/testify/mock"
)

func TestAccountClient_PatchAccountByEmail(t *testing.T) {
	// Create our mock proto client and the AccountClient that wraps it.
	mockProtoClient := new(logic.MockAccountServiceClient)
	accountClient := NewAccountClientWithClient(mockProtoClient)

	testCases := []struct {
		name     string
		email    string
		username string
		// protoResp is what the underlying proto call returns.
		protoResp *proto.PatchAccountByEmailResponse
		protoErr  error
		// expectedResp is what we expect the client method to return.
		expectedResp *proto.PatchAccountByEmailResponse
	}{
		{
			name:     "Successful proto call",
			email:    "test@example.com",
			username: "new_username",
			protoResp: &proto.PatchAccountByEmailResponse{
				ShortNamespace: "new_username",
				Errors:         []*proto.Error{},
			},
			protoErr: nil,
			expectedResp: &proto.PatchAccountByEmailResponse{
				ShortNamespace: "new_username",
				Errors:         []*proto.Error{},
			},
		},
		{
			name:      "Proto call returns error",
			email:     "test@example.com",
			username:  "new_username",
			protoResp: nil,
			protoErr:  errors.New("proto error"),
			expectedResp: &proto.PatchAccountByEmailResponse{
				Errors: []*proto.Error{
					{
						Code:    cerror.InternalServerError,
						Message: "proto error",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the request that we expect the client to send.
			req := &proto.PatchAccountByEmailRequest{
				Email:    tc.email,
				Username: tc.username,
			}
			// Set up the expectation on the mock proto client.
			mockProtoClient.On("PatchAccountByEmail", mock.Anything, req).
				Return(tc.protoResp, tc.protoErr).Once()

			// Call the client method.
			resp := accountClient.PatchAccountByEmail(tc.email, tc.username)

			// Assert that the response matches the expected response.
			assert.Equal(t, tc.expectedResp, resp)

			// Ensure all expectations were met.
			mockProtoClient.AssertExpectations(t)
		})
	}
}
