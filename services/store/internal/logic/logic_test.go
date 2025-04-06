package logic

import (
	"context"
	"testing"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	"github.com/idlab-discover/kebeng/services/store/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterSnapName(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name          string
		req           *proto.RegisterSnapNameRequest
		mockReturn    map[string]any // either *models.SnapEntry or cerror.CustomError
		expectedError bool
		errorCode     string
	}{
		{
			name: "Successful registration",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "test-snap-name",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    false,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "Snap name not found",
				},
				"RegisterSnap": &models.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap-name",
				},
				"AddTrack": &models.SnapTrack{
					Name: "test-track",
					ID:   mockUUID,
				},
				"AddDefaultChannels": nil,
			},
			expectedError: false,
		},
		{
			name: "failed because snap name already exists",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "test-snap-name",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    false,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &models.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap-name",
				},
			},
			expectedError: true,
			errorCode:     cerror.AlreadyRegistered,
		},
		{
			name: "database error in RegisterSnap",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "test-snap-name",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    false,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "snap name not found",
				},
				"RegisterSnap": &cerror.CustomError{
					Code:    cerror.DatabaseError,
					Message: "database error",
				},
			},
			expectedError: true,
			errorCode:     cerror.DatabaseError,
		},
		{
			name: "Fail to add track",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "test-snap-name",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    false,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "Snap name not found",
				},
				"RegisterSnap": &models.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap-name",
				},
				"AddTrack": &cerror.CustomError{
					Code:    cerror.DatabaseError,
					Message: "database error",
				},
			},
			expectedError: true,
			errorCode:     cerror.DatabaseError,
		},
		{
			name: "fail to add default channels",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "test-snap-name",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    false,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "Snap name not found",
				},
				"RegisterSnap": &models.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap-name",
				},
				"AddTrack": &models.SnapTrack{
					Name: "test-track",
					ID:   mockUUID,
				},
				"AddDefaultChannels": &cerror.CustomError{
					Code:    cerror.DatabaseError,
					Message: "database error",
				},
			},
			expectedError: true,
			errorCode:     cerror.DatabaseError,
		},
		{
			name: "RegisterSnapName with dry run = true for existing name",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "test-snap-name",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    true,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &models.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap-name",
				},
			},
			expectedError: false,
		},
		{
			name: "RegisterSnapName with dry run = true for non-existing name",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "test-snap-name-non-existing",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    true,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "snap with name not found",
				},
			},
			expectedError: false,
		},
		{
			name: "fail to register snap name with empty name",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    false,
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for function, mockReturn := range tt.mockReturn {
				switch mockReturn := mockReturn.(type) {
				case *cerror.CustomError:
					switch function {
					case "GetEntryByName":
						mockRepo.On(function, tt.req.SnapName, mock.Anything).Return(nil, mockReturn).Once()
					case "RegisterSnap":
						mockRepo.On(function, tt.req.SnapName, mock.Anything, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					case "AddTrack":
						mockRepo.On(function, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					case "AddDefaultChannels":
						mockRepo.On(function, mock.Anything, mock.Anything).Return(mockReturn).Once()
					default:
						t.Fatalf("invalid mock return function for CustomError")
					}
				case *models.SnapEntry:
					switch function {
					case "GetEntryByName":
						mockRepo.On(function, tt.req.SnapName, mock.Anything).Return(mockReturn, nil).Once()
					case "RegisterSnap":
						mockRepo.On(function, tt.req.SnapName, mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					default:
						t.Fatalf("invalid mock return function for SnapEntry")
					}
				case *models.SnapTrack:
					switch function {
					case "AddTrack":
						mockRepo.On(function, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					default:
						t.Fatalf("invalid mock return function for SnapTrack")
					}
				case nil:
					switch function {
					case "AddDefaultChannels":
						mockRepo.On(function, mock.Anything, mock.Anything).Return(nil).Once()
					default:
						t.Fatalf("invalid mock return function for nil")
					}
				default:
					t.Fatalf("invalid mock return type")
				}
			}

			// Call the method under test
			resp, _ := service.RegisterSnapName(context.Background(), tt.req)

			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
				assert.GreaterOrEqual(t, 1, len(resp.Errors), "Expected a single error")
				assert.Equal(t, tt.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors in response")
				assert.NotNil(t, resp, "Expected a valid entry")
			}
		})
	}

}
