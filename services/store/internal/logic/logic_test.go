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
						mockRepo.On(function, tt.req.SnapName, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					case "RegisterSnap":
						mockRepo.On(function, tt.req.SnapName, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					case "AddTrack":
						mockRepo.On(function, mock.Anything, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					case "AddDefaultChannels":
						mockRepo.On(function, mock.Anything, mock.Anything).Return(mockReturn).Once()
					default:
						t.Fatalf("invalid mock return function for CustomError")
					}
				case *models.SnapEntry:
					switch function {
					case "GetEntryByName":
						mockRepo.On(function, tt.req.SnapName, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					case "RegisterSnap":
						mockRepo.On(function, tt.req.SnapName, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					default:
						t.Fatalf("invalid mock return function for SnapEntry")
					}
				case *models.SnapTrack:
					switch function {
					case "AddTrack":
						mockRepo.On(function, mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
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
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors in response")
				assert.NotNil(t, resp, "Expected a valid entry")
			}
		})
	}

}

func TestAddUpload(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name          string
		req           *proto.AddUploadRequest
		mockReturn    map[string]any // either *models.SnapUpload or cerror.CustomError
		expectedError bool
		errorCode     string
	}{
		{
			name: "Successful upload",
			req: &proto.AddUploadRequest{
				SnapName:          "test-snap-name",
				EntryId:           mockUUID.String(),
				AccountId:         mockUUID.String(),
				Status:            "pending",
				UnscannedFileName: "test-file-name",
			},
			mockReturn: map[string]any{
				"AddUpload": &models.SnapUpload{
					ID:               mockUUID,
					SnapName:         "test-snap-name",
					Status:           "pending",
					StatusDetailsURL: "/dev/api/snaps/" + mockUUID.String() + "/status",
				},
			},
			expectedError: false,
		},
		{
			name: "Missing SnapName",
			req: &proto.AddUploadRequest{
				SnapName:          "",
				EntryId:           mockUUID.String(),
				AccountId:         mockUUID.String(),
				Status:            "pending",
				UnscannedFileName: "test-file-name",
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Missing EntryId",
			req: &proto.AddUploadRequest{
				SnapName:          "test-snap-name",
				EntryId:           "",
				AccountId:         mockUUID.String(),
				Status:            "pending",
				UnscannedFileName: "test-file-name",
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Invalid EntryId format",
			req: &proto.AddUploadRequest{
				SnapName:          "test-snap-name",
				EntryId:           "invalid-uuid",
				AccountId:         mockUUID.String(),
				Status:            "pending",
				UnscannedFileName: "test-file-name",
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			errorCode:     cerror.InvalidField,
		},
		{
			name: "Invalid AccountId format",
			req: &proto.AddUploadRequest{
				SnapName:          "test-snap-name",
				EntryId:           mockUUID.String(),
				AccountId:         "invalid-uuid",
				Status:            "pending",
				UnscannedFileName: "test-file-name",
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			errorCode:     cerror.InvalidField,
		},
		{
			name: "Database error during AddUpload",
			req: &proto.AddUploadRequest{
				SnapName:          "test-snap-name",
				EntryId:           mockUUID.String(),
				AccountId:         mockUUID.String(),
				Status:            "pending",
				UnscannedFileName: "test-file-name",
			},
			mockReturn: map[string]any{
				"AddUpload": &cerror.CustomError{
					Code:    cerror.DatabaseError,
					Message: "Database error",
				},
			},
			expectedError: true,
			errorCode:     cerror.DatabaseError,
		},
		{
			name: "Missing AccountId",
			req: &proto.AddUploadRequest{
				SnapName:          "test-snap-name",
				EntryId:           mockUUID.String(),
				AccountId:         "",
				Status:            "pending",
				UnscannedFileName: "test-file-name",
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Missing Status",
			req: &proto.AddUploadRequest{
				SnapName:          "test-snap-name",
				EntryId:           mockUUID.String(),
				AccountId:         mockUUID.String(),
				Status:            "",
				UnscannedFileName: "test-file-name",
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Missing UnscannedFileName",
			req: &proto.AddUploadRequest{
				SnapName:          "test-snap-name",
				EntryId:           mockUUID.String(),
				AccountId:         mockUUID.String(),
				Status:            "pending",
				UnscannedFileName: "",
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
					case "AddUpload":
						mockRepo.On(function, tt.req.SnapName, mock.Anything, tt.req.Status, mock.Anything, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					default:
						t.Fatalf("invalid mock return function for CustomError")
					}
				case *models.SnapUpload:
					switch function {
					case "AddUpload":
						mockRepo.On(function, tt.req.SnapName, mock.Anything, tt.req.Status, mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					default:
						t.Fatalf("invalid mock return function for SnapUpload")
					}
				default:
					t.Fatalf("invalid mock return type")
				}
			}

			// Call the method under test
			resp, _ := service.AddUpload(context.Background(), tt.req)

			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors in response")
				assert.NotNil(t, resp, "Expected a valid response")
				assert.Equal(t, tt.req.SnapName, resp.SnapName, "Expected SnapName to match")
			}
		})
	}
}
