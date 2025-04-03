package logic

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/google/uuid"
// 	cerror "github.com/idlab-discover/kebeng/common/cerror"
// 	"github.com/idlab-discover/kebeng/services/store/internal/config"
// 	"github.com/idlab-discover/kebeng/services/store/internal/models"
// 	"github.com/idlab-discover/kebeng/services/store/internal/repository"
// 	proto "github.com/idlab-discover/kebeng/services/store/proto"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// // Helper function to return time pointers
// func ptrTime(t time.Time) *time.Time {
// 	return &t
// }

// func TestRegisterSnapName(t *testing.T) {
// 	mockRepo := new(repository.MockSnapsRepository)
// 	service := NewStoreLogic(mockRepo, &config.Config{})

// 	mockUUID := uuid.New()
// 	tests := []struct {
// 		name          string
// 		req           *proto.RegisterSnapNameRequest
// 		mockReturn    interface{} // either *models.SnapEntry or cerror.CustomError
// 		expectedError bool
// 		errorCode     string
// 	}{
// 		{
// 			name: "Successful registration",
// 			req: &proto.RegisterSnapNameRequest{
// 				SnapName:  "test-snap-name",
// 				AccountId: "test-account-id",
// 			},
// 			mockReturn: &models.SnapEntry{
// 				ID:        uuid.New(),
// 				Name:      "test-snap-name",
// 				AccountID: mockUUID,
// 				CreatedAt: time.Now(),
// 				UpdatedAt: time.Now(),
// 			},
// 			expectedError: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if entry, ok := tt.mockReturn.(*models.SnapEntry); ok {
// 				mockRepo.On("GetEntryByName", tt.req.SnapName, mock.Anything).Return(nil, nil).Once()
// 				mockRepo.On("RegisterSnap", tt.req.SnapName, mock.Anything).Return(entry, nil).Once()
// 			} else if err, ok := tt.mockReturn.(*cerror.CustomError); ok {
// 				mockRepo.On("GetEntryByName", tt.req.SnapName, mock.Anything).Return(nil, err).Once()
// 				mockRepo.On("RegisterSnap", tt.req.SnapName, mock.Anything).Return(nil, err).Once()
// 			} else {
// 				t.Fatalf("Invalid mock return type")
// 			}

// 			// Call the method under test
// 			resp, _ := service.RegisterSnapName(context.Background(), tt.req)

// 			if tt.expectedError {
// 				assert.NotNil(t, resp.Errors)
// 				assert.GreaterOrEqual(t, 1, len(resp.Errors), "Expected a single error")
// 				assert.Equal(t, tt.errorCode, resp.Errors[0].Code, "Expected error code to match")
// 			} else {
// 				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors in response")
// 				assert.NotNil(t, resp, "Expected a valid entry")
// 				assert.Equal(t, tt.req.SnapName, resp.SnapName)
// 			}
// 		})
// 	}

// }
