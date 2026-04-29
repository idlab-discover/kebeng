package logic

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/model"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	"github.com/idlab-discover/kebeng/services/store/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/store/proto"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	protoCommon "github.com/idlab-discover/kebeng/common/cerror/proto"
	"github.com/sirupsen/logrus"
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
		mockReturn    map[string]any // either *model.SnapEntry or cerror.CustomError
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
				"RegisterSnap": &model.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap-name",
				},
				"AddTrack": &model.SnapTrack{
					Name: "test-track",
					ID:   mockUUID,
				},
				"AddDefaultChannels": nil,
			},
			expectedError: false,
		},
		{
			name: "failed because invalid snap name",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "123_INVALID",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    false,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.InvalidField,
					Message: "snap name is invalid, it should only have ASCII lowercase letters, numbers, and hyphens, and must have at least one letter",
				},
			},
			expectedError: true,
			errorCode:     cerror.InvalidField,
		},
		{
			name: "failed because empty snap name",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    false,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.InvalidField,
					Message: "snap name is invalid, it should only have ASCII lowercase letters, numbers, and hyphens, and must have at least one letter",
				},
			},
			expectedError: true,
			errorCode:     cerror.InvalidField,
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
				"GetEntryByName": &model.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap-name",
				},
			},
			expectedError: true,
			errorCode:     cerror.AlreadyRegistered,
		},
		{
			name: "database error in GetEntryByName",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "test-snap-name",
				IsPrivate: false,
				Store:     "global",
				AccountId: mockUUID.String(),
				DryRun:    false,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.DatabaseError,
					Message: "database error",
				},
			},
			expectedError: true,
			errorCode:     cerror.DatabaseError,
		},
		{
			name: "failed parsing account id",
			req: &proto.RegisterSnapNameRequest{
				SnapName:  "test-snap-name",
				IsPrivate: false,
				Store:     "global",
				AccountId: "invalid-uuid",
				DryRun:    false,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "snap name not found",
				},
			},
			expectedError: true,
			errorCode:     cerror.InvalidField,
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
				"RegisterSnap": &model.SnapEntry{
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
				"RegisterSnap": &model.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap-name",
				},
				"AddTrack": &model.SnapTrack{
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
				"GetEntryByName": &model.SnapEntry{
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
				case *model.SnapEntry:
					switch function {
					case "GetEntryByName":
						mockRepo.On(function, tt.req.SnapName, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					case "RegisterSnap":
						mockRepo.On(function, tt.req.SnapName, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					default:
						t.Fatalf("invalid mock return function for SnapEntry")
					}
				case *model.SnapTrack:
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

func TestGetEntries(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name          string
		req           *proto.GetEntriesRequest
		mockReturn    map[string]any // either *model.SnapEntry or cerror.CustomError
		expectedError bool
		expectedCount int
	}{
		{
			name: "Successful retrieval by ID",
			req: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{
					{Id: stringToPointer(mockUUID.String())},
				},
			},
			mockReturn: map[string]any{
				"GetEntryById": &model.SnapEntry{
					ID:          mockUUID,
					Name:        "test-snap",
					Type:        "app",
					Confinement: "strict",
					Base:        "core20",
					Private:     false,
					AccountID:   mockUUID,
					Price:       0,
					Status:      "active",
					CreatedAt:   time.Now(),
					IconURL:     "http://example.com/icon.png",
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name: "Invalid UUID format",
			req: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{
					{Id: stringToPointer("invalid-uuid")},
				},
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name: "Missing ID and Name",
			req: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{
					{Id: nil, Name: nil},
				},
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name: "Successful retrieval by Name",
			req: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{
					{Name: stringToPointer("test-snap")},
				},
			},
			mockReturn: map[string]any{
				"GetEntryByName": &model.SnapEntry{
					ID:          mockUUID,
					Name:        "test-snap",
					Type:        "app",
					Confinement: "strict",
					Base:        "core20",
					Private:     false,
					AccountID:   mockUUID,
					Price:       0,
					Status:      "active",
					CreatedAt:   time.Now(),
					IconURL:     "http://example.com/icon.png",
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name: "error in GetEntryById",
			req: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{
					{Id: stringToPointer(mockUUID.String())},
				},
			},
			mockReturn: map[string]any{
				"GetEntryById": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "entry not found",
				},
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name: "error in GetEntryByName",
			req: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{
					{Name: stringToPointer("test-snap")},
				},
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "entry not found",
				},
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for function, mockReturn := range tt.mockReturn {
				switch mockReturn := mockReturn.(type) {
				case *model.SnapEntry:
					switch function {
					case "GetEntryById":
						mockRepo.On(function, mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					case "GetEntryByName":
						mockRepo.On(function, mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					default:
						t.Fatalf("invalid mock return function for SnapEntry")
					}
				case *cerror.CustomError:
					switch function {
					case "GetEntryById":
						mockRepo.On(function, mock.Anything, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					case "GetEntryByName":
						mockRepo.On(function, mock.Anything, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					default:
						t.Fatalf("invalid mock return function for CustomError")
					}
				default:
					t.Fatalf("invalid mock return type")
				}
			}

			// Call the method under test
			resp, _ := service.GetEntries(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
				assert.Greater(t, len(resp.Errors), 0, "Expected errors in response")
			} else {
				assert.Nil(t, resp.Errors)
				assert.Equal(t, tt.expectedCount, len(resp.Entries), "Expected entry count to match")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetEntryById(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name          string
		req           *proto.GetEntryRequest
		mockReturn    any // either *model.SnapEntry or *cerror.CustomError
		expectedError bool
		errorCode     string
		expectedEntry *proto.GetEntryResponse
	}{
		{
			name: "Successful retrieval by ID",
			req: &proto.GetEntryRequest{
				Id: stringToPointer(mockUUID.String()),
			},
			mockReturn: &model.SnapEntry{
				ID:          mockUUID,
				Name:        "test-snap",
				Type:        "app",
				Confinement: "strict",
				Base:        "core20",
				Private:     false,
				AccountID:   mockUUID,
				Price:       0,
				Status:      "active",
				CreatedAt:   time.Now(),
				IconURL:     "http://example.com/icon.png",
			},
			expectedError: false,
			expectedEntry: &proto.GetEntryResponse{
				Id:          mockUUID.String(),
				SnapName:    "test-snap",
				Type:        "app",
				Confinement: "strict",
				Base:        "core20",
				Private:     false,
				PublisherId: mockUUID.String(),
				Price:       0,
				Status:      "active",
				IconUrl:     "http://example.com/icon.png",
			},
		},
		{
			name: "Invalid UUID format",
			req: &proto.GetEntryRequest{
				Id: stringToPointer("invalid-uuid"),
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.InvalidField,
		},
		{
			name: "Missing ID",
			req: &proto.GetEntryRequest{
				Id: nil,
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Entry not found",
			req: &proto.GetEntryRequest{
				Id: stringToPointer(mockUUID.String()),
			},
			mockReturn: &cerror.CustomError{
				Code:    cerror.InternalServerError, // In mock, we use InternalServerError for simplicity
				Message: "entry not found",
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch mockReturn := tt.mockReturn.(type) {
			case *model.SnapEntry:
				mockRepo.On("GetEntryById", mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
			case *cerror.CustomError:
				mockRepo.On("GetEntryById", mock.Anything, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
			default:
				// no action needed
			}

			// Call the method under test
			resp, _ := service.GetEntryById(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
				assert.Equal(t, tt.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Nil(t, resp.Errors)
				assert.Equal(t, tt.expectedEntry.Id, resp.Id, "Expected ID to match")
				assert.Equal(t, tt.expectedEntry.SnapName, resp.SnapName, "Expected SnapName to match")
				assert.Equal(t, tt.expectedEntry.Type, resp.Type, "Expected Type to match")
				assert.Equal(t, tt.expectedEntry.Confinement, resp.Confinement, "Expected Confinement to match")
				assert.Equal(t, tt.expectedEntry.Base, resp.Base, "Expected Base to match")
				assert.Equal(t, tt.expectedEntry.Private, resp.Private, "Expected Private to match")
				assert.Equal(t, tt.expectedEntry.PublisherId, resp.PublisherId, "Expected PublisherId to match")
				assert.Equal(t, tt.expectedEntry.Price, resp.Price, "Expected Price to match")
				assert.Equal(t, tt.expectedEntry.Status, resp.Status, "Expected Status to match")
				assert.Equal(t, tt.expectedEntry.IconUrl, resp.IconUrl, "Expected IconUrl to match")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetEntryByName(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name          string
		req           *proto.GetEntryRequest
		mockReturn    any // either *model.SnapEntry or *cerror.CustomError
		expectedError bool
		errorCode     string
		expectedEntry *proto.GetEntryResponse
	}{
		{
			name: "Successful retrieval by Name",
			req: &proto.GetEntryRequest{
				Name: stringToPointer("test-snap"),
			},
			mockReturn: &model.SnapEntry{
				ID:          mockUUID,
				Name:        "test-snap",
				Type:        "app",
				Confinement: "strict",
				Base:        "core20",
				Private:     false,
				AccountID:   mockUUID,
				Price:       0,
				Status:      "active",
				CreatedAt:   time.Now(),
				IconURL:     "http://example.com/icon.png",
			},
			expectedError: false,
			expectedEntry: &proto.GetEntryResponse{
				Id:          mockUUID.String(),
				SnapName:    "test-snap",
				Type:        "app",
				Confinement: "strict",
				Base:        "core20",
				Private:     false,
				PublisherId: mockUUID.String(),
				Price:       0,
				Status:      "active",
				IconUrl:     "http://example.com/icon.png",
			},
		},
		{
			name: "missing name",
			req: &proto.GetEntryRequest{
				Name: nil,
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "empty name",
			req: &proto.GetEntryRequest{
				Name: stringToPointer(""),
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "database error in GetEntryByName",
			req: &proto.GetEntryRequest{
				Name: stringToPointer("test-snap"),
			},
			mockReturn: &cerror.CustomError{
				Code:    cerror.InternalServerError, // In mock, we use InternalServerError for simplicity
				Message: "entry not found",
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch mockReturn := tt.mockReturn.(type) {
			case *model.SnapEntry:
				mockRepo.On("GetEntryByName", mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
			case *cerror.CustomError:
				mockRepo.On("GetEntryByName", mock.Anything, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
			default:
				// no action needed
			}

			// Call the method under test
			resp, _ := service.GetEntryByName(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
				assert.Equal(t, tt.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Nil(t, resp.Errors)
				assert.Equal(t, tt.expectedEntry.Id, resp.Id, "Expected ID to match")
				assert.Equal(t, tt.expectedEntry.SnapName, resp.SnapName, "Expected SnapName to match")
				assert.Equal(t, tt.expectedEntry.Type, resp.Type, "Expected Type to match")
				assert.Equal(t, tt.expectedEntry.Confinement, resp.Confinement, "Expected Confinement to match")
				assert.Equal(t, tt.expectedEntry.Base, resp.Base, "Expected Base to match")
				assert.Equal(t, tt.expectedEntry.Private, resp.Private, "Expected Private to match")
				assert.Equal(t, tt.expectedEntry.PublisherId, resp.PublisherId, "Expected PublisherId to match")
				assert.Equal(t, tt.expectedEntry.Price, resp.Price, "Expected Price to match")
				assert.Equal(t, tt.expectedEntry.Status, resp.Status, "Expected Status to match")
				assert.Equal(t, tt.expectedEntry.IconUrl, resp.IconUrl, "Expected IconUrl to match")
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetRevisions(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name          string
		req           *proto.GetRevisionsRequest
		mockReturn    map[string]any // either *model.SnapRevision or *cerror.CustomError
		expectedError bool
		expectedCount int
	}{
		{
			name: "Successful retrieval by ID",
			req: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{
					{Id: mockUUID.String()},
				},
			},
			mockReturn: map[string]any{
				"GetRevisionById": &model.SnapRevision{
					ID:               mockUUID,
					SnapName:         "test-snap",
					SequenceNumber:   1,
					Architectures:    []string{"amd64"},
					SHA3_384_Encoded: "test-hash",
					Size:             12345,
					CreatedAt:        time.Now(),
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name: "Invalid UUID format",
			req: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{
					{Id: "invalid-uuid"},
				},
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name: "Missing ID and SnapName/Sequence",
			req: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{
					{Id: "", SnapName: "", Sequence: 0},
				},
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name: "Successful retrieval by SnapName and Sequence",
			req: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{
					{SnapName: "test-snap", Sequence: 1},
				},
			},
			mockReturn: map[string]any{
				"GetRevisionByNameAndSequence": &model.SnapRevision{
					ID:               mockUUID,
					SnapName:         "test-snap",
					SequenceNumber:   1,
					Architectures:    []string{"amd64"},
					SHA3_384_Encoded: "test-hash",
					Size:             12345,
					CreatedAt:        time.Now(),
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name: "Revision not found by ID",
			req: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{
					{Id: mockUUID.String()},
				},
			},
			mockReturn: map[string]any{
				"GetRevisionById": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "revision not found",
				},
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name: "Revision not found by SnapName and Sequence",
			req: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{
					{SnapName: "test-snap", Sequence: 1},
				},
			},
			mockReturn: map[string]any{
				"GetRevisionByNameAndSequence": &cerror.CustomError{
					Code:    cerror.ResourceNotFound,
					Message: "revision not found",
				},
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for function, mockReturn := range tt.mockReturn {
				switch mockReturn := mockReturn.(type) {
				case *model.SnapRevision:
					switch function {
					case "GetRevisionById":
						mockRepo.On(function, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					case "GetRevisionByNameAndSequence":
						mockRepo.On(function, mock.Anything, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
					default:
						t.Fatalf("invalid mock return function for SnapRevision")
					}
				case *cerror.CustomError:
					switch function {
					case "GetRevisionById":
						mockRepo.On(function, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					case "GetRevisionByNameAndSequence":
						mockRepo.On(function, mock.Anything, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					default:
						t.Fatalf("invalid mock return function for CustomError")
					}
				default:
					t.Fatalf("invalid mock return type")
				}
			}

			// Call the method under test
			resp, _ := service.GetRevisions(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
				assert.Greater(t, len(resp.Errors), 0, "Expected errors in response")
			} else {
				assert.Nil(t, resp.Errors)
				assert.Equal(t, tt.expectedCount, len(resp.Revisions), "Expected revision count to match")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetRevisionByNameAndSequence(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name           string
		req            *proto.GetRevisionRequest
		mockReturn     map[string]any // either *model.SnapRevision or *cerror.CustomError
		expectedError  bool
		errorCode      string
		expectedResult *proto.GetRevisionResponse
	}{
		{
			name: "Successful retrieval by SnapName and Sequence",
			req: &proto.GetRevisionRequest{
				SnapName: "test-snap",
				Sequence: 1,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &model.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap",
				},
				"GetRevisionByNameAndSequence": &model.SnapRevision{
					ID:               mockUUID,
					SnapName:         "test-snap",
					SequenceNumber:   1,
					Architectures:    []string{"amd64"},
					SHA3_384_Encoded: "test-hash",
					Size:             12345,
					CreatedAt:        time.Now(),
				},
			},
			expectedError: false,
			expectedResult: &proto.GetRevisionResponse{
				Id:              mockUUID.String(),
				SnapName:        "test-snap",
				SequenceNumber:  1,
				Architectures:   []string{"amd64"},
				Sha3_384Encoded: "test-hash",
				Size:            12345,
			},
		},
		{
			name: "Missing SnapName",
			req: &proto.GetRevisionRequest{
				SnapName: "",
				Sequence: 1,
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Missing Sequence",
			req: &proto.GetRevisionRequest{
				SnapName: "test-snap",
				Sequence: 0,
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Entry not found",
			req: &proto.GetRevisionRequest{
				SnapName: "test-snap",
				Sequence: 1,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &cerror.CustomError{
					Code:    cerror.InternalServerError, // In mock, we use InternalServerError for simplicity
					Message: "entry not found",
				},
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
		{
			name: "Revision not found",
			req: &proto.GetRevisionRequest{
				SnapName: "test-snap",
				Sequence: 1,
			},
			mockReturn: map[string]any{
				"GetEntryByName": &model.SnapEntry{
					ID:   mockUUID,
					Name: "test-snap",
				},
				"GetRevisionByNameAndSequence": &cerror.CustomError{
					Code:    cerror.InternalServerError, // In mock, we use InternalServerError for simplicity
					Message: "revision not found",
				},
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for function, mockReturn := range tt.mockReturn {
				switch mockReturn := mockReturn.(type) {
				case *model.SnapEntry:
					mockRepo.On("GetEntryByName", tt.req.SnapName, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
				case *model.SnapRevision:
					mockRepo.On("GetRevisionByNameAndSequence", tt.req.SnapName, tt.req.Sequence, mock.Anything).Return(mockReturn, nil).Once()
				case *cerror.CustomError:
					switch function {
					case "GetEntryByName":
						mockRepo.On("GetEntryByName", tt.req.SnapName, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					case "GetRevisionByNameAndSequence":
						mockRepo.On("GetRevisionByNameAndSequence", tt.req.SnapName, tt.req.Sequence, mock.Anything).Return(nil, mockReturn).Once()
					default:
						t.Fatalf("invalid mock return function for CustomError")
					}
				default:
					t.Fatalf("invalid mock return type")
				}
			}

			// Call the method under test
			resp, _ := service.GetRevisionByNameAndSequence(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
				assert.Equal(t, tt.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Nil(t, resp.Errors)
				assert.Equal(t, tt.expectedResult.Id, resp.Id, "Expected ID to match")
				assert.Equal(t, tt.expectedResult.SnapName, resp.SnapName, "Expected SnapName to match")
				assert.Equal(t, tt.expectedResult.SequenceNumber, resp.SequenceNumber, "Expected SequenceNumber to match")
				assert.Equal(t, tt.expectedResult.Architectures, resp.Architectures, "Expected Architectures to match")
				assert.Equal(t, tt.expectedResult.Sha3_384Encoded, resp.Sha3_384Encoded, "Expected Sha3_384Encoded to match")
				assert.Equal(t, tt.expectedResult.Size, resp.Size, "Expected Size to match")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetEntriesByAccountId(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name           string
		req            *proto.GetEntriesByAccountIdRequest
		mockReturn     any // either []*model.SnapEntry or *cerror.CustomError
		expectedError  bool
		errorCode      string
		expectedResult []*proto.GetEntryResponse
	}{
		{
			name: "Successful retrieval by AccountId",
			req: &proto.GetEntriesByAccountIdRequest{
				AccountId: mockUUID.String(),
			},
			mockReturn: []*model.SnapEntry{
				{
					ID:          mockUUID,
					Name:        "test-snap",
					Type:        "app",
					Confinement: "strict",
					Base:        "core20",
					Private:     false,
					AccountID:   mockUUID,
					Price:       0,
					Status:      "active",
					CreatedAt:   time.Now(),
					IconURL:     "http://example.com/icon.png",
				},
			},
			expectedError: false,
			expectedResult: []*proto.GetEntryResponse{
				{
					Id:          mockUUID.String(),
					SnapName:    "test-snap",
					Type:        "app",
					Confinement: "strict",
					Base:        "core20",
					Private:     false,
					PublisherId: mockUUID.String(),
					Price:       0,
					Status:      "active",
					IconUrl:     "http://example.com/icon.png",
				},
			},
		},
		{
			name: "Invalid AccountId format",
			req: &proto.GetEntriesByAccountIdRequest{
				AccountId: "invalid-uuid",
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.InvalidField,
		},
		{
			name: "Missing AccountId",
			req: &proto.GetEntriesByAccountIdRequest{
				AccountId: "",
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "No entries found for AccountId",
			req: &proto.GetEntriesByAccountIdRequest{
				AccountId: mockUUID.String(),
			},
			mockReturn:     []*model.SnapEntry{},
			expectedError:  false,
			expectedResult: []*proto.GetEntryResponse{},
		},
		{
			name: "Database error in GetEntriesByAccountId",
			req: &proto.GetEntriesByAccountIdRequest{
				AccountId: mockUUID.String(),
			},
			mockReturn: &cerror.CustomError{
				Code:    cerror.InternalServerError,
				Message: "database error",
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch mockReturn := tt.mockReturn.(type) {
			case []*model.SnapEntry:
				accountID, err := uuid.Parse(tt.req.AccountId)
				if err != nil {
					t.Fatalf("invalid UUID format for AccountId: %v", err)
				}
				mockRepo.On("GetEntriesByAccountId", accountID, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
			case *cerror.CustomError:
				accountID, err := uuid.Parse(tt.req.AccountId)
				if err != nil {
					t.Fatalf("invalid UUID format for AccountId: %v", err)
				}
				mockRepo.On("GetEntriesByAccountId", accountID, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
			default:
				// no action needed
			}

			// Call the method under test
			resp, _ := service.GetEntriesByAccountId(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
				assert.Equal(t, tt.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Nil(t, resp.Errors)
				assert.Equal(t, len(tt.expectedResult), len(resp.Entries), "Expected entry count to match")
				for i, entry := range resp.Entries {
					assert.Equal(t, tt.expectedResult[i].Id, entry.Id, "Expected ID to match")
					assert.Equal(t, tt.expectedResult[i].SnapName, entry.SnapName, "Expected SnapName to match")
					assert.Equal(t, tt.expectedResult[i].Type, entry.Type, "Expected Type to match")
					assert.Equal(t, tt.expectedResult[i].Confinement, entry.Confinement, "Expected Confinement to match")
					assert.Equal(t, tt.expectedResult[i].Base, entry.Base, "Expected Base to match")
					assert.Equal(t, tt.expectedResult[i].Private, entry.Private, "Expected Private to match")
					assert.Equal(t, tt.expectedResult[i].PublisherId, entry.PublisherId, "Expected PublisherId to match")
					assert.Equal(t, tt.expectedResult[i].Price, entry.Price, "Expected Price to match")
					assert.Equal(t, tt.expectedResult[i].Status, entry.Status, "Expected Status to match")
					assert.Equal(t, tt.expectedResult[i].IconUrl, entry.IconUrl, "Expected IconUrl to match")
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetRevisionsByEntryIds(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name           string
		req            *proto.GetRevisionsByEntryIdRequests
		mockReturn     map[string]any // either []*model.SnapRevision or *cerror.CustomError
		expectedError  bool
		expectedResult []*proto.GetRevisionsByEntryIdResponse
	}{
		{
			name: "Successful retrieval of revisions by EntryId",
			req: &proto.GetRevisionsByEntryIdRequests{
				Requests: []*proto.GetRevisionsByEntryIdRequest{
					{EntryId: mockUUID.String()},
				},
			},
			mockReturn: map[string]any{
				"GetRevisionsByEntryId": []*model.SnapRevision{
					{
						ID:               mockUUID,
						SnapName:         "test-snap",
						SequenceNumber:   1,
						Architectures:    []string{"amd64"},
						SHA3_384_Encoded: "test-hash",
						Size:             12345,
						CreatedAt:        time.Now(),
					},
				},
			},
			expectedError: false,
			expectedResult: []*proto.GetRevisionsByEntryIdResponse{
				{
					EntryId: mockUUID.String(),
					Revisions: []*proto.GetRevisionResponse{
						{
							Id:              mockUUID.String(),
							SnapName:        "test-snap",
							SequenceNumber:  1,
							Architectures:   []string{"amd64"},
							Sha3_384Encoded: "test-hash",
							Size:            12345,
						},
					},
				},
			},
		},
		{
			name: "Invalid EntryId format",
			req: &proto.GetRevisionsByEntryIdRequests{
				Requests: []*proto.GetRevisionsByEntryIdRequest{
					{EntryId: "invalid-uuid"},
				},
			},
			mockReturn:    nil,
			expectedError: true,
			expectedResult: []*proto.GetRevisionsByEntryIdResponse{
				{
					EntryId: "",
					Errors: []*protoCommon.Error{
						{
							Code:    cerror.InvalidField,
							Message: "invalid UUID format: invalid-uuid",
						},
					},
				},
			},
		},
		{
			name: "Missing EntryId",
			req: &proto.GetRevisionsByEntryIdRequests{
				Requests: []*proto.GetRevisionsByEntryIdRequest{
					{EntryId: ""},
				},
			},
			mockReturn:    nil,
			expectedError: true,
			expectedResult: []*proto.GetRevisionsByEntryIdResponse{
				{
					EntryId: "",
					Errors: []*protoCommon.Error{
						{
							Code:    cerror.MissingField,
							Message: "entry id is required",
						},
					},
				},
			},
		},
		{
			name: "No revisions found for EntryId",
			req: &proto.GetRevisionsByEntryIdRequests{
				Requests: []*proto.GetRevisionsByEntryIdRequest{
					{EntryId: mockUUID.String()},
				},
			},
			mockReturn: map[string]any{
				"GetRevisionsByEntryId": []*model.SnapRevision{},
			},
			expectedError: false,
			expectedResult: []*proto.GetRevisionsByEntryIdResponse{
				{
					EntryId:   mockUUID.String(),
					Revisions: []*proto.GetRevisionResponse{},
				},
			},
		},
		{
			name: "Database error in GetRevisionsByEntryId",
			req: &proto.GetRevisionsByEntryIdRequests{
				Requests: []*proto.GetRevisionsByEntryIdRequest{
					{EntryId: mockUUID.String()},
				},
			},
			mockReturn: map[string]any{
				"GetRevisionsByEntryId": &cerror.CustomError{
					Code:    cerror.InternalServerError, // In mock, we use InternalServerError for simplicity
					Message: "database error",
				},
			},
			expectedError: true,
			expectedResult: []*proto.GetRevisionsByEntryIdResponse{
				{
					EntryId: mockUUID.String(),
					Errors: []*protoCommon.Error{
						{
							Code:    cerror.InternalServerError,
							Message: "database error",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for function, mockReturn := range tt.mockReturn {
				switch mockReturn := mockReturn.(type) {
				case []*model.SnapRevision:
					entryID, err := uuid.Parse(tt.req.Requests[0].EntryId)
					if err != nil {
						t.Fatalf("invalid UUID format for EntryId: %v", err)
					}
					mockRepo.On(function, entryID, mock.Anything).Return(mockReturn, nil).Once()
				case *cerror.CustomError:
					entryID, err := uuid.Parse(tt.req.Requests[0].EntryId)
					if err != nil {
						t.Fatalf("invalid UUID format for EntryId: %v", err)
					}
					mockRepo.On(function, entryID, mock.Anything).Return(nil, mockReturn).Once()
				default:
					// no action needed
				}
			}

			// Call the method under test
			resp, _ := service.GetRevisionsByEntryIds(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Responses)
				for i, response := range resp.Responses {
					assert.Equal(t, tt.expectedResult[i].EntryId, response.EntryId, "Expected EntryId to match")
					assert.Equal(t, len(tt.expectedResult[i].Errors), len(response.Errors), "Expected error count to match")
					for j, err := range response.Errors {
						assert.Equal(t, tt.expectedResult[i].Errors[j].Code, err.Code, "Expected error code to match")
					}
				}
			} else {
				assert.Nil(t, resp.Responses[0].Errors)
				assert.Equal(t, len(tt.expectedResult[0].Revisions), len(resp.Responses[0].Revisions), "Expected revision count to match")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetLatestRevisionByTrackAndChannel(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name           string
		req            *proto.GetLatestRevisionRequest
		mockReturn     any // either *model.SnapRevision or *cerror.CustomError
		expectedError  bool
		errorCode      string
		expectedResult *proto.GetRevisionResponse
	}{
		{
			name: "Successful retrieval of latest revision",
			req: &proto.GetLatestRevisionRequest{
				SnapName: "test-snap",
				Track:    "latest",
				Channel:  "stable",
			},
			mockReturn: &model.SnapRevision{
				ID:               mockUUID,
				SnapName:         "test-snap",
				SequenceNumber:   1,
				Architectures:    []string{"amd64"},
				SHA3_384_Encoded: "test-hash",
				Size:             12345,
				CreatedAt:        time.Now(),
			},
			expectedError: false,
			expectedResult: &proto.GetRevisionResponse{
				Id:              mockUUID.String(),
				SnapName:        "test-snap",
				SequenceNumber:  1,
				Architectures:   []string{"amd64"},
				Sha3_384Encoded: "test-hash",
				Size:            12345,
			},
		},
		{
			name: "Missing SnapName",
			req: &proto.GetLatestRevisionRequest{
				SnapName: "",
				Track:    "latest",
				Channel:  "stable",
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Revision not found",
			req: &proto.GetLatestRevisionRequest{
				SnapName: "test-snap",
				Track:    "latest",
				Channel:  "stable",
			},
			mockReturn: &cerror.CustomError{
				Code:    cerror.InternalServerError, // In mock, we use InternalServerError for simplicity
				Message: "revision not found",
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
		{
			name: "Database error in GetLatestRevisionByTrackAndChannel",
			req: &proto.GetLatestRevisionRequest{
				SnapName: "test-snap",
				Track:    "latest",
				Channel:  "stable",
			},
			mockReturn: &cerror.CustomError{
				Code:    cerror.InternalServerError, // In mock, we use InternalServerError for simplicity
				Message: "database error",
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch mockReturn := tt.mockReturn.(type) {
			case *model.SnapRevision:
				mockRepo.On("GetLatestRevisionByTrackAndChannel", tt.req.SnapName, tt.req.Track, tt.req.Channel, mock.Anything).Return(mockReturn, nil).Once()
			case *cerror.CustomError:
				mockRepo.On("GetLatestRevisionByTrackAndChannel", tt.req.SnapName, tt.req.Track, tt.req.Channel, mock.Anything).Return(nil, mockReturn).Once()
			default:
				// no action needed
			}

			// Call the method under test
			resp, _ := service.GetLatestRevisionByTrackAndChannel(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
				assert.Equal(t, tt.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Nil(t, resp.Errors)
				assert.Equal(t, tt.expectedResult.Id, resp.Id, "Expected ID to match")
				assert.Equal(t, tt.expectedResult.SnapName, resp.SnapName, "Expected SnapName to match")
				assert.Equal(t, tt.expectedResult.SequenceNumber, resp.SequenceNumber, "Expected SequenceNumber to match")
				assert.Equal(t, tt.expectedResult.Architectures, resp.Architectures, "Expected Architectures to match")
				assert.Equal(t, tt.expectedResult.Sha3_384Encoded, resp.Sha3_384Encoded, "Expected Sha3_384Encoded to match")
				assert.Equal(t, tt.expectedResult.Size, resp.Size, "Expected Size to match")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSnapDownload(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)
	mockStream := new(repository.MockSnapDownloadServer)

	mockUUID := uuid.New()

	tests := []struct {
		name          string
		req           *proto.SnapDownloadRequest
		mockReturn    map[string]any // either *model.SnapDownload or *cerror.CustomError
		expectedError bool
		errorCode     string
	}{
		{
			name: "Successful download",
			req: &proto.SnapDownloadRequest{
				RevisionId: mockUUID.String(),
			},
			mockReturn: map[string]any{
				"GetRevisionById": &model.SnapRevision{
					ID:               mockUUID,
					SnapName:         "test-snap",
					SequenceNumber:   1,
					Architectures:    []string{"amd64"},
					SHA3_384_Encoded: "test-hash",
					Size:             12345,
					CreatedAt:        time.Now(),
				},
				"GetSnapFileReader": io.NopCloser(strings.NewReader("dummy snap content")),
				"Send":              nil,
			},
			expectedError: false,
		},
		{
			name: "Missing RevisionId",
			req: &proto.SnapDownloadRequest{
				RevisionId: "",
			},
			mockReturn:    map[string]any{},
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "error getting objectStoreFilePath",
			req: &proto.SnapDownloadRequest{
				RevisionId: mockUUID.String(),
			},
			mockReturn: map[string]any{
				"GetRevisionById": &cerror.CustomError{
					Code:    cerror.InternalServerError,
					Message: "error getting objectStoreFilePath",
				},
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
		{
			name: "error getting file reader",
			req: &proto.SnapDownloadRequest{
				RevisionId: mockUUID.String(),
			},
			mockReturn: map[string]any{
				"GetRevisionById": &model.SnapRevision{
					ID:               mockUUID,
					SnapName:         "test-snap",
					SequenceNumber:   1,
					Architectures:    []string{"amd64"},
					SHA3_384_Encoded: "test-hash",
					Size:             12345,
					CreatedAt:        time.Now(),
				},
				"GetSnapFileReader": fmt.Errorf("error getting file reader"),
				"Send":              nil,
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for function, mockReturn := range tt.mockReturn {
				switch v := mockReturn.(type) {
				case *model.SnapRevision:
					mockRepo.On(function, mock.Anything, mock.Anything).Return(v, nil).Maybe()
				case io.ReadCloser:
					mockObj.On(function, mock.Anything, mock.Anything).Return(v, nil).Maybe()
				case *cerror.CustomError:
					mockRepo.On(function, tt.req.RevisionId, mock.Anything).Return(nil, v).Maybe()
				case error:
					mockObj.On(function, mock.Anything, mock.Anything).Return(nil, v).Maybe()
				case nil:
					// Controleer de Send-aanroep via Run voor de fouten
					mockStream.On("Send", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
						resp := args.Get(0).(*proto.SnapDownloadResponse)
						if tt.expectedError {
							assert.Len(t, resp.Errors, 1)
							assert.Equal(t, tt.errorCode, resp.Errors[0].Code)
						}
					}).Maybe()
				default:
					t.Fatalf("invalid mock return type for %s: %T", function, v)
				}
			}

			// Call the method under test
			_ = service.SnapDownload(tt.req, mockStream)

			if !tt.expectedError {
				mockStream.AssertExpectations(t)
			}
			mockObj.AssertExpectations(t)
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
		mockReturn    map[string]any // either *model.SnapUpload or cerror.CustomError
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
				"AddUpload": &model.SnapUpload{
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
						mockRepo.On(function, mock.Anything, mock.Anything, tt.req.SnapName, tt.req.Status, tt.req.UnscannedFileName, mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
					default:
						t.Fatalf("invalid mock return function for CustomError")
					}
				case *model.SnapUpload:
					switch function {
					case "AddUpload":
						entryID, err := uuid.Parse(tt.req.EntryId)
						if err != nil {
							t.Fatalf("invalid UUID format for EntryId: %v", err)
						}
						accountID, err := uuid.Parse(tt.req.AccountId)
						if err != nil {
							t.Fatalf("invalid UUID format for AccountId: %v", err)
						}
						mockRepo.On(function, entryID, accountID, tt.req.SnapName, tt.req.Status, tt.req.UnscannedFileName, mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
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

func TestUnscannedUpload(t *testing.T) {} // TODO: Implement this test

func TestGetUploadStatus(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name          string
		req           *proto.GetUploadStatusRequest
		mockReturn    any // either *model.SnapUpload or *cerror.CustomError
		expectedError bool
		errorCode     string
		expectedResp  *proto.GetUploadStatusResponse
	}{
		{
			name: "Successful retrieval of upload status",
			req: &proto.GetUploadStatusRequest{
				UploadId: mockUUID.String(),
			},
			mockReturn: &model.SnapUpload{
				ID:       mockUUID,
				Status:   "processed",
				Revision: 1,
			},
			expectedError: false,
			expectedResp: &proto.GetUploadStatusResponse{
				UploadId:  mockUUID.String(),
				Processed: true,
				Revision:  1,
			},
		},
		{
			name: "Upload not processed yet",
			req: &proto.GetUploadStatusRequest{
				UploadId: mockUUID.String(),
			},
			mockReturn: &model.SnapUpload{
				ID:       mockUUID,
				Status:   "pending",
				Revision: 0,
			},
			expectedError: false,
			expectedResp: &proto.GetUploadStatusResponse{
				UploadId:  mockUUID.String(),
				Processed: false,
				Revision:  0,
			},
		},
		{
			name: "Missing UploadId",
			req: &proto.GetUploadStatusRequest{
				UploadId: "",
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Invalid UploadId format",
			req: &proto.GetUploadStatusRequest{
				UploadId: "invalid-uuid",
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.InvalidField,
		},
		{
			name: "Upload not found",
			req: &proto.GetUploadStatusRequest{
				UploadId: mockUUID.String(),
			},
			mockReturn: &cerror.CustomError{
				Code:    cerror.InternalServerError, // In mock, we use InternalServerError for simplicity
				Message: "upload not found",
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch mockReturn := tt.mockReturn.(type) {
			case *model.SnapUpload:
				mockRepo.On("GetUploadById", mock.Anything, mock.Anything).Return(mockReturn, nil).Once()
			case *cerror.CustomError:
				mockRepo.On("GetUploadById", mock.Anything, mock.Anything).Return(nil, mockReturn).Once()
			default:
				// no action needed
			}

			// Call the method under test
			resp, _ := service.GetUploadStatus(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
				assert.Equal(t, tt.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Nil(t, resp.Errors)
				assert.Equal(t, tt.expectedResp.UploadId, resp.UploadId, "Expected UploadId to match")
				assert.Equal(t, tt.expectedResp.Processed, resp.Processed, "Expected Processed status to match")
				assert.Equal(t, tt.expectedResp.Revision, resp.Revision, "Expected Revision to match")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateUploadStatus(t *testing.T) {
	mockRepo := new(repository.MockSnapsRepository)
	mockObj := new(objectstore.MockObjectStore)
	service := NewStoreLogic(mockRepo, &config.Config{}, mockObj)

	mockUUID := uuid.New()

	tests := []struct {
		name          string
		req           *proto.UpdateUploadStatusRequest
		mockReturn    *cerror.CustomError
		expectedError bool
		errorCode     string
		expectedResp  *proto.UpdateUploadStatusResponse
	}{
		{
			name: "Successful status update",
			req: &proto.UpdateUploadStatusRequest{
				UploadId: mockUUID.String(),
				Status:   "processed",
				Revision: 1,
			},
			mockReturn:    nil,
			expectedError: false,
			expectedResp: &proto.UpdateUploadStatusResponse{
				Status: "processed",
			},
		},
		{
			name: "Missing UploadId",
			req: &proto.UpdateUploadStatusRequest{
				UploadId: "",
				Status:   "processed",
				Revision: 1,
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.MissingField,
		},
		{
			name: "Invalid UploadId format",
			req: &proto.UpdateUploadStatusRequest{
				UploadId: "invalid-uuid",
				Status:   "processed",
				Revision: 1,
			},
			mockReturn:    nil,
			expectedError: true,
			errorCode:     cerror.InvalidField,
		},
		{
			name: "Database error during status update",
			req: &proto.UpdateUploadStatusRequest{
				UploadId: mockUUID.String(),
				Status:   "processed",
				Revision: 1,
			},
			mockReturn: &cerror.CustomError{
				Code:    cerror.InternalServerError, // In mock, we use InternalServerError for simplicity
				Message: "database error",
			},
			expectedError: true,
			errorCode:     cerror.InternalServerError,
		},
		{
			name: "errors in request",
			req: &proto.UpdateUploadStatusRequest{
				UploadId: mockUUID.String(),
				Status:   "processed",
				Revision: 1,
				Errors: []*protoCommon.Error{
					{
						Code:    cerror.InternalServerError,
						Message: "internal server error",
					},
				},
			},
			mockReturn:    nil,
			expectedError: true,
			expectedResp: &proto.UpdateUploadStatusResponse{
				Status: "processed",
				Errors: []*protoCommon.Error{
					{
						Code:    cerror.InternalServerError,
						Message: "internal server error",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.UploadId != "" && tt.req.UploadId != "invalid-uuid" {
				parsedUUID, err := uuid.Parse(tt.req.UploadId)
				if err != nil {
					t.Fatalf("failed to parse UploadId: %v", err)
				}
				if tt.mockReturn == nil {
					mockRepo.On("UpdateUploadStatus", parsedUUID, tt.req.Status, tt.req.Revision, mock.Anything).Return(nil).Once()
				} else {
					mockRepo.On("UpdateUploadStatus", parsedUUID, tt.req.Status, tt.req.Revision, mock.Anything).Return(tt.mockReturn).Once()
				}
			}

			// Call the method under test
			resp, _ := service.UpdateUploadStatus(context.Background(), tt.req)

			// Assertions
			if tt.expectedError {
				assert.NotNil(t, resp.Errors)
			} else {
				logrus.Infof("Response: %+v", resp)
				assert.Nil(t, resp.Errors)
				assert.Equal(t, tt.expectedResp.Status, resp.Status, "Expected status to match")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// ====== Helper functions ======
func stringToPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
