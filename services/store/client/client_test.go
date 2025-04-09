package client_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/store/client"
	"github.com/idlab-discover/kebeng/services/store/internal/logic"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStoreClient_AddUpload(t *testing.T) {
	// Create our mock proto client and the StoreClient that wraps it.
	mockProtoClient := new(logic.MockStoreServiceClient)
	storeClient := client.NewStoreClientWithClient(mockProtoClient)

	mockID := uuid.New()

	testCases := []struct {
		name               string
		snapName           string
		entryId            uuid.UUID
		status             string
		accountId          uuid.UUID
		expectedResp       *proto.AddUploadResponse
		expectedErrors     bool
		expectedProtoError bool
	}{
		{
			name:      "Successful proto call",
			snapName:  "test_snap",
			entryId:   mockID,
			status:    "pending",
			accountId: uuid.New(),
			expectedResp: &proto.AddUploadResponse{
				Id:       mockID.String(),
				SnapName: "test_snap",
				Status:   "pending",
				Errors:   []*proto.Error{},
			},
			expectedErrors:     false,
			expectedProtoError: false,
		},
		{
			name:               "proto call returns error",
			snapName:           "test_snap",
			entryId:            uuid.New(),
			status:             "pending",
			accountId:          uuid.New(),
			expectedResp:       nil,
			expectedProtoError: true,
		},
		{
			name:     "response contains errors",
			snapName: "test_snap",
			entryId:  uuid.New(),
			expectedResp: &proto.AddUploadResponse{
				Errors: []*proto.Error{{
					Code:    cerror.InternalServerError,
					Message: "mock error",
				}},
			},
			expectedErrors:     true,
			expectedProtoError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedProtoError {
				mockProtoClient.On("AddUpload", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
			} else {
				mockProtoClient.On("AddUpload", mock.Anything, mock.Anything).Return(tc.expectedResp, nil).Once()
			}

			resp := storeClient.AddUpload(tc.snapName, tc.entryId, tc.status, tc.accountId)
			if !tc.expectedErrors && !tc.expectedProtoError {
				assert.Equal(t, tc.expectedResp, resp)
				assert.Empty(t, resp.Errors)
			} else if tc.expectedErrors {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
			} else if tc.expectedProtoError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				assert.Equal(t, cerror.InternalServerError, resp.Errors[0].Code)
			}
			mockProtoClient.AssertExpectations(t)
		})
	}
}
