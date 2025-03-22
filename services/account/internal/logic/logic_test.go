package logic

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/account/internal/config"
	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/idlab-discover/kebeng/services/account/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function to return time pointers
func ptrTime(t time.Time) *time.Time {
	return &t
}

func TestCreateAccountLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	testCases := []struct {
		name        string
		req         *proto.CreateAccountRequest
		mockReturn  interface{} // Either a *models.Account or a cerror.CustomError
		expectError bool
		errorCode   string
	}{
		{
			name: "Successful account creation",
			req: &proto.CreateAccountRequest{
				DisplayName: "Alice",
				Username:    "alice123",
				Email:       "alice@example.com",
			},
			mockReturn: &models.Account{
				ID:          uuid.New(),
				DisplayName: "Alice",
				Username:    "alice123",
				Email:       "alice@example.com",
				CreatedAt:   ptrTime(time.Now()),
				UpdatedAt:   ptrTime(time.Now()),
			},
			expectError: false,
		},
		{
			name: "Duplicate username",
			req: &proto.CreateAccountRequest{
				DisplayName: "Alice",
				Username:    "alice123", // Already exists
				Email:       "alice2@example.com",
			},
			mockReturn:  cerror.NewCustomError(cerror.AlreadyRegistered, "username already taken"),
			expectError: true,
			errorCode:   cerror.AlreadyRegistered,
		},
		{
			name: "Invalid email",
			req: &proto.CreateAccountRequest{
				DisplayName: "Alice",
				Username:    "alicevalid",
				Email:       "", // Invalid email
			},
			mockReturn:  cerror.NewCustomError(cerror.InvalidField, "email is required"),
			expectError: true,
			errorCode:   cerror.InvalidField,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock before calling the function
			if account, ok := tc.mockReturn.(*models.Account); ok {
				mockRepo.On("CreateAccount", mock.Anything, mock.Anything).Return(account, nil).Once()
			} else if err, ok := tc.mockReturn.(*cerror.CustomError); ok {
				mockRepo.On("CreateAccount", mock.Anything, mock.Anything).Return(nil, err).Once()
			} else {
				t.Fatalf("Unexpected type in mockReturn: %T", tc.mockReturn)
			}

			// Call the service function
			resp, _ := service.CreateAccount(context.Background(), tc.req)

			// Assertions
			if tc.expectError {
				assert.NotNil(t, resp.Errors, "Expected an error response")
				assert.GreaterOrEqual(t, 1, len(resp.Errors), "Expected a single error")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors in response")
				assert.NotNil(t, resp, "Expected a valid account")
				assert.Equal(t, tc.req.DisplayName, resp.DisplayName)
				assert.Equal(t, tc.req.Username, resp.Username)
				assert.Equal(t, tc.req.Email, resp.Email)
			}

			// Ensure the expected method was called
			mockRepo.AssertCalled(t, "CreateAccount", mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateAccountLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	// string because proto uses a string
	validID := uuid.New().String()

	testCases := []struct {
		name        string
		req         *proto.UpdateAccountRequest
		mockReturn  interface{} // Either *models.Account or *cerror.CustomError
		expectError bool
		errorCode   string
	}{
		{
			name: "Successful account update",
			req: &proto.UpdateAccountRequest{
				Id:          validID,
				DisplayName: "UpdatedAlice",
				Username:    "alice_updated",
				Email:       "alice_updated@example.com",
			},
			mockReturn: &models.Account{
				ID:          uuid.MustParse(validID),
				DisplayName: "UpdatedAlice",
				Username:    "alice_updated",
				Email:       "alice_updated@example.com",
				CreatedAt:   ptrTime(time.Now().Add(-time.Hour)),
				UpdatedAt:   ptrTime(time.Now()),
			},
			expectError: false,
		},
		{
			name: "Invalid UUID",
			req: &proto.UpdateAccountRequest{
				Id:          "not-a-valid-uuid",
				DisplayName: "SomeName",
				Username:    "someusername",
				Email:       "some@example.com",
			},
			// no repository call will be made because uuid.Parse fails
			mockReturn:  nil,
			expectError: true,
			errorCode:   cerror.BadRequest,
		},
		{
			name: "Repository update error",
			req: &proto.UpdateAccountRequest{
				Id:          validID,
				DisplayName: "Alice",
				Username:    "alice",
				Email:       "alice@example.com",
			},
			mockReturn:  cerror.NewCustomError(cerror.ResourceNotFound, "account not found"),
			expectError: true,
			errorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tc := range testCases {
		// Create a fresh mock repository and service for each sub-test.
		t.Run(tc.name, func(t *testing.T) {

			// If the UUID is valid, set up repository expectation.
			if _, err := uuid.Parse(tc.req.Id); err == nil {
				// Setup expectation for UpdateAccount call.
				if account, ok := tc.mockReturn.(*models.Account); ok {
					mockRepo.On("UpdateAccount", mock.Anything, mock.AnythingOfType("*models.Account")).
						Return(account, nil).Once()
				} else if errVal, ok := tc.mockReturn.(*cerror.CustomError); ok {
					mockRepo.On("UpdateAccount", mock.Anything, mock.AnythingOfType("*models.Account")).
						Return(nil, errVal).Once()
				}
			}

			// Call the UpdateAccount service function.
			resp, _ := service.UpdateAccount(context.Background(), tc.req)
			t.Logf("DEBUG: performed UpdateAccount, resp = %+v", resp)

			// Assertions
			if tc.expectError {
				assert.Equal(t, 1, len(resp.Errors), "Expected an error response")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error in response")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors in response")
				// Assuming convertToProtoAccount maps the fields directly.
				assert.NotNil(t, resp, "Expected a valid account")
				assert.Equal(t, tc.req.DisplayName, resp.DisplayName)
				assert.Equal(t, tc.req.Username, resp.Username)
				assert.Equal(t, tc.req.Email, resp.Email)
			}

			// Verify that the expected method was called if a valid UUID was provided.
			if _, err := uuid.Parse(tc.req.Id); err == nil {
				mockRepo.AssertCalled(t, "UpdateAccount", mock.Anything, mock.AnythingOfType("*models.Account"))
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestDeleteAccountLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	// Use a valid UUID string for tests that require it.
	validID := uuid.New().String()

	testCases := []struct {
		name        string
		req         *proto.DeleteAccountRequest
		mockReturn  interface{} // nil means success, or a *cerror.CustomError for failure.
		expectError bool
		errorCode   string
	}{
		{
			name: "Successful deletion",
			req: &proto.DeleteAccountRequest{
				Id: validID,
			},
			mockReturn:  nil,
			expectError: false,
		},
		{
			name: "Invalid UUID",
			req: &proto.DeleteAccountRequest{
				Id: "not-a-valid-uuid",
			},
			// No repository call will be made because uuid.Parse fails.
			mockReturn:  nil,
			expectError: true,
			errorCode:   cerror.BadRequest,
		},
		{
			name: "Repository deletion error",
			req: &proto.DeleteAccountRequest{
				Id: validID,
			},
			mockReturn:  cerror.NewCustomError(cerror.ResourceNotFound, "account not found"),
			expectError: true,
			errorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// If the UUID is valid, set up the repository expectation.
			if _, err := uuid.Parse(tc.req.Id); err == nil {
				if tc.mockReturn == nil {
					mockRepo.On("DeleteAccount", mock.Anything, mock.AnythingOfType("uuid.UUID")).
						Return(nil).Once()
				} else if errVal, ok := tc.mockReturn.(*cerror.CustomError); ok {
					mockRepo.On("DeleteAccount", mock.Anything, mock.AnythingOfType("uuid.UUID")).
						Return(errVal).Once()
				}
			}

			// Call the DeleteAccount service function.
			resp, _ := service.DeleteAccount(context.Background(), tc.req)
			t.Logf("DEBUG: performed DeleteAccount, resp = %+v", resp)

			// Assertions.
			if tc.expectError {
				assert.False(t, resp.Success, "Expected success to be false")
				assert.NotNil(t, resp.Errors, "Expected errors in response")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.True(t, resp.Success, "Expected success to be true")
				assert.Nil(t, resp.Errors, "Did not expect errors")
			}

			// If a valid UUID was provided, verify the repository call.
			if _, err := uuid.Parse(tc.req.Id); err == nil {
				mockRepo.AssertCalled(t, "DeleteAccount", mock.Anything, mock.AnythingOfType("uuid.UUID"))
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestGetAccountByEmailLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	testCases := []struct {
		name        string
		req         *proto.GetAccountByEmailRequest
		mockReturn  interface{} // Either *models.Account or *cerror.CustomError.
		expectError bool
		errorCode   string
	}{
		{
			name: "Successful account retrieval",
			req: &proto.GetAccountByEmailRequest{
				Email: "alice@example.com",
			},
			mockReturn: &models.Account{
				ID:          uuid.New(),
				DisplayName: "Alice",
				Username:    "alice123",
				Email:       "alice@example.com",
				CreatedAt:   ptrTime(time.Now().Add(-time.Hour)),
				UpdatedAt:   ptrTime(time.Now()),
			},
			expectError: false,
		},
		{
			name: "Repository error: account not found",
			req: &proto.GetAccountByEmailRequest{
				Email: "notfound@example.com",
			},
			mockReturn:  cerror.NewCustomError(cerror.ResourceNotFound, "account not found"),
			expectError: true,
			errorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Set up the repository expectation for GetAccountByEmail.
			if account, ok := tc.mockReturn.(*models.Account); ok {
				mockRepo.On("GetAccountByEmail", mock.Anything, tc.req.Email, []string{models.ALL}).
					Return(account, nil).Once()
			} else if errVal, ok := tc.mockReturn.(*cerror.CustomError); ok {
				mockRepo.On("GetAccountByEmail", mock.Anything, tc.req.Email, []string{models.ALL}).
					Return(nil, errVal).Once()
			} else {
				t.Fatalf("Unexpected type in mockReturn: %T", tc.mockReturn)
			}

			// Call the GetAccountByEmail service function.
			resp, _ := service.GetAccountByEmail(context.Background(), tc.req)
			t.Logf("DEBUG: performed GetAccountByEmail, resp = %+v", resp)

			// Assertions.
			if tc.expectError {
				assert.NotNil(t, resp.Errors, "Expected error response")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors")
				assert.NotNil(t, resp, "Expected a valid account")
				// Optionally, compare additional fields.
				assert.Equal(t, tc.req.Email, resp.Email)
			}

			// Verify the repository call.
			mockRepo.AssertCalled(t, "GetAccountByEmail", mock.Anything, tc.req.Email, []string{models.ALL})
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetAccountByIDLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	// Use a valid UUID string for tests that require it.
	validID := uuid.New().String()

	testCases := []struct {
		name        string
		req         *proto.GetAccountByIDRequest
		mockReturn  interface{} // Either *models.Account or *cerror.CustomError
		expectError bool
		errorCode   string
	}{
		{
			name: "Successful account retrieval",
			req: &proto.GetAccountByIDRequest{
				Id: validID,
			},
			mockReturn: &models.Account{
				ID:          uuid.MustParse(validID),
				DisplayName: "randomtestUserNameUser",
				Username:    "randomtestUserNameusername",
				Email:       "randomtestUserName@example.com",
				CreatedAt:   ptrTime(time.Now().Add(-time.Hour)),
				UpdatedAt:   ptrTime(time.Now()),
			},
			expectError: false,
		},
		{
			name: "Invalid UUID",
			req: &proto.GetAccountByIDRequest{
				Id: "invalid-uuid",
			},
			// No repository call will be made because uuid.Parse fails.
			mockReturn:  nil,
			expectError: true,
			errorCode:   cerror.BadRequest,
		},
		{
			name: "Repository error",
			req: &proto.GetAccountByIDRequest{
				Id: validID,
			},
			mockReturn:  cerror.NewCustomError(cerror.ResourceNotFound, "account not found"),
			expectError: true,
			errorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// If uuid.Parse succeeds, set up repository expectation.
			if _, err := uuid.Parse(tc.req.Id); err == nil {
				if account, ok := tc.mockReturn.(*models.Account); ok {
					mockRepo.On("GetAccountByID", mock.Anything, mock.AnythingOfType("uuid.UUID"), []string{models.ALL}).
						Return(account, nil).Once()
				} else if errVal, ok := tc.mockReturn.(*cerror.CustomError); ok {
					mockRepo.On("GetAccountByID", mock.Anything, mock.AnythingOfType("uuid.UUID"), []string{models.ALL}).
						Return(nil, errVal).Once()
				}
			}

			// Call the service function.
			resp, _ := service.GetAccountByID(context.Background(), tc.req)
			t.Logf("DEBUG: performed GetAccountByID, resp = %+v", resp)

			// Assertions.
			if tc.expectError {
				assert.NotNil(t, resp.Errors, "Expected an error response")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
				if len(resp.Errors) > 0 {
					assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
				}
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors")
				assert.NotNil(t, resp, "Expected a valid account")
				// Compare the returned account ID with the request.
				assert.Equal(t, tc.req.Id, resp.Id, "Expected account id to match")
				// Optionally, compare other fields.
				if account, ok := tc.mockReturn.(*models.Account); ok {
					assert.Equal(t, account.DisplayName, resp.DisplayName)
					assert.Equal(t, account.Username, resp.Username)
					assert.Equal(t, account.Email, resp.Email)
				} else {
					t.Fatalf("Unexpected type in mockReturn: %T", tc.mockReturn)
				}
			}

			// If a valid UUID was provided, verify that the repository call was made.
			if _, err := uuid.Parse(tc.req.Id); err == nil {
				mockRepo.AssertCalled(t, "GetAccountByID", mock.Anything, mock.AnythingOfType("uuid.UUID"), []string{models.ALL})
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestGetAccountsByIdsLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	// Create two valid UUID strings.
	validID1 := uuid.New().String()
	validID2 := uuid.New().String()

	// Define expected account models for valid IDs.
	validAccount1 := &models.Account{
		ID:          uuid.MustParse(validID1),
		DisplayName: "User1",
		Username:    "user1",
		Email:       "user1@example.com",
		CreatedAt:   ptrTime(time.Now().Add(-time.Hour)),
		UpdatedAt:   ptrTime(time.Now()),
	}
	validAccount2 := &models.Account{
		ID:          uuid.MustParse(validID2),
		DisplayName: "User2",
		Username:    "user2",
		Email:       "user2@example.com",
		CreatedAt:   ptrTime(time.Now().Add(-time.Hour)),
		UpdatedAt:   ptrTime(time.Now()),
	}

	testCases := []struct {
		name             string
		req              *proto.GetAccountsByIdsRequest
		repoResponses    map[string]interface{} // maps ID string to either *models.Account or *cerror.CustomError
		expectError      bool
		errorCode        string
		expectedAccounts int // expected number of accounts in the response (if no early error)
	}{
		{
			name: "Successful retrieval for two valid IDs",
			req: &proto.GetAccountsByIdsRequest{
				Ids: []string{validID1, validID2},
			},
			repoResponses: map[string]interface{}{
				validID1: validAccount1,
				validID2: validAccount2,
			},
			expectError:      false,
			expectedAccounts: 2,
		},
		{
			name: "Invalid UUID in request",
			req: &proto.GetAccountsByIdsRequest{
				Ids: []string{"invalid-uuid", validID1},
			},
			// No repository calls should occur since at least one ID is invalid.
			repoResponses:    nil,
			expectError:      true,
			errorCode:        cerror.BadRequest,
			expectedAccounts: 0,
		},
		{
			name: "Repository error for one account",
			req: &proto.GetAccountsByIdsRequest{
				Ids: []string{validID1, validID2},
			},
			repoResponses: map[string]interface{}{
				validID1: validAccount1,
				validID2: cerror.NewCustomError(cerror.ResourceNotFound, "account not found"),
			},
			// Even if one account is returned successfully, the presence of an error should cause the response to include errors.
			expectError:      true,
			errorCode:        cerror.ResourceNotFound,
			expectedAccounts: 1,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {

			// Track if all IDs in the request are valid.
			allIDsValid := true
			for _, id := range tc.req.Ids {
				if _, err := uuid.Parse(id); err != nil {
					allIDsValid = false
					break
				}
			}

			// If all IDs are valid, set up repository expectations.
			if allIDsValid && tc.repoResponses != nil {
				for _, id := range tc.req.Ids {
					parsed, _ := uuid.Parse(id)
					if respVal, ok := tc.repoResponses[id]; ok {
						switch v := respVal.(type) {
						case *models.Account:
							mockRepo.On("GetAccountByID", mock.Anything, parsed, []string{models.ALL}).
								Return(v, nil).Once()
						case *cerror.CustomError:
							mockRepo.On("GetAccountByID", mock.Anything, parsed, []string{models.ALL}).
								Return(nil, v).Once()
						}
					}
				}
			}

			// Call the service function.
			resp, _ := service.GetAccountsByIds(context.Background(), tc.req)
			t.Logf("DEBUG: performed GetAccountsByIds, resp = %+v", resp)

			// If not all IDs are valid, we expect an error response and no repository calls.
			if !allIDsValid {
				assert.NotNil(t, resp.Errors, "Expected error response due to invalid UUID")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				// Otherwise, if all IDs are valid:
				if tc.expectError {
					assert.NotNil(t, resp.Errors, "Expected error response")
					assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
					assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
				} else {
					assert.Equal(t, 0, len(resp.Errors), "Did not expect errors")
					assert.Equal(t, tc.expectedAccounts, len(resp.Accounts), "Expected number of accounts to match")
				}
				// Verify that for each valid UUID a repository call was made.
				for _, id := range tc.req.Ids {
					if parsed, err := uuid.Parse(id); err == nil {
						mockRepo.AssertCalled(t, "GetAccountByID", mock.Anything, parsed, []string{models.ALL})
					}
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetAccountByUsernameLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	testCases := []struct {
		name        string
		req         *proto.GetAccountByUsernameRequest
		mockReturn  interface{} // Either a *models.Account or a *cerror.CustomError
		expectError bool
		errorCode   string
	}{
		{
			name: "Successful account retrieval",
			req: &proto.GetAccountByUsernameRequest{
				Username: "user123",
			},
			mockReturn: &models.Account{
				ID:          uuid.New(),
				DisplayName: "User 123",
				Username:    "user123",
				Email:       "user123@example.com",
				CreatedAt:   ptrTime(time.Now().Add(-time.Hour)),
				UpdatedAt:   ptrTime(time.Now()),
			},
			expectError: false,
		},
		{
			name: "Repository error",
			req: &proto.GetAccountByUsernameRequest{
				Username: "nonexistent",
			},
			mockReturn:  cerror.NewCustomError(cerror.ResourceNotFound, "account not found"),
			expectError: true,
			errorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Setup expectation on the mock repository.
			if account, ok := tc.mockReturn.(*models.Account); ok {
				mockRepo.On("GetAccountByUsername", mock.Anything, tc.req.Username, []string{models.ALL}).
					Return(account, nil).Once()
			} else if errVal, ok := tc.mockReturn.(*cerror.CustomError); ok {
				mockRepo.On("GetAccountByUsername", mock.Anything, tc.req.Username, []string{models.ALL}).
					Return(nil, errVal).Once()
			} else {
				t.Fatalf("Unexpected type in mockReturn: %T", tc.mockReturn)
			}

			// Call the service function.
			resp, _ := service.GetAccountByUsername(context.Background(), tc.req)
			t.Logf("DEBUG: performed GetAccountByUsername, resp = %+v", resp)

			// Assertions.
			if tc.expectError {
				assert.NotNil(t, resp.Errors, "Expected an error response")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors in response")
				assert.NotNil(t, resp, "Expected a valid account")
				assert.Equal(t, tc.req.Username, resp.Username, "Expected username to match")
				// Optionally, compare additional fields like DisplayName and Email.
			}

			// Verify that the repository method was called as expected.
			mockRepo.AssertCalled(t, "GetAccountByUsername", mock.Anything, tc.req.Username, []string{models.ALL})
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAddKeyLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	testCases := []struct {
		name        string
		req         *proto.AddKeyRequest
		mockReturn  interface{} // Either a *models.Key or a *cerror.CustomError
		expectError bool
		errorCode   string
	}{
		{
			name: "Successful key addition",
			req: &proto.AddKeyRequest{
				KeyName:          "TestKey",
				Sha3384:          "sha123",
				EncodedPublicKey: "encodedPublicKey",
				AccountEmail:     "test@example.com",
			},
			mockReturn: &models.Key{
				Name:             "TestKey",
				SHA3384:          "sha123",
				EncodedPublicKey: "encodedPublicKey",
			},
			expectError: false,
		},
		{
			name: "Repository error when adding key",
			req: &proto.AddKeyRequest{
				KeyName:          "TestKey",
				Sha3384:          "sha123",
				EncodedPublicKey: "encodedPublicKey",
				AccountEmail:     "test@example.com",
			},
			mockReturn:  cerror.NewCustomError(cerror.ResourceNotFound, "account not found"),
			expectError: true,
			errorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Setup the mock expectation.
			if key, ok := tc.mockReturn.(*models.Key); ok {
				mockRepo.On("AddKeyToAccountByEmail", mock.Anything, tc.req.KeyName, tc.req.Sha3384, tc.req.EncodedPublicKey, tc.req.AccountEmail).
					Return(key, nil).Once()
			} else if errVal, ok := tc.mockReturn.(*cerror.CustomError); ok {
				mockRepo.On("AddKeyToAccountByEmail", mock.Anything, tc.req.KeyName, tc.req.Sha3384, tc.req.EncodedPublicKey, tc.req.AccountEmail).
					Return(nil, errVal).Once()
			} else {
				t.Fatalf("Unexpected type in mockReturn: %T", tc.mockReturn)
			}

			// Call the service function.
			resp, _ := service.AddKey(context.Background(), tc.req)
			t.Logf("DEBUG: performed AddKey, resp = %+v", resp)

			// Assertions.
			if tc.expectError {
				assert.NotNil(t, resp.Errors, "Expected an error response")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors in response")
				assert.Equal(t, tc.req.KeyName, resp.Name, "Expected key name to match")
				assert.Equal(t, tc.req.Sha3384, resp.Sha3384, "Expected SHA3384 to match")
				assert.Equal(t, tc.req.EncodedPublicKey, resp.EncodedPublicKey, "Expected encoded public key to match")
			}

			// Verify that the repository method was called.
			mockRepo.AssertCalled(t, "AddKeyToAccountByEmail", mock.Anything, tc.req.KeyName, tc.req.Sha3384, tc.req.EncodedPublicKey, tc.req.AccountEmail)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetKeyLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	testCases := []struct {
		name        string
		req         *proto.GetKeyBySHA3384Request
		mockReturn  interface{} // Either a *models.Key or a *cerror.CustomError
		expectError bool
		errorCode   string
	}{
		{
			name: "Successful key retrieval",
			req: &proto.GetKeyBySHA3384Request{
				Sha3384: "sha123",
			},
			mockReturn: &models.Key{
				Name:             "TestKey",
				SHA3384:          "sha123",
				EncodedPublicKey: "encodedPublicKey",
			},
			expectError: false,
		},
		{
			name: "Repository error when getting key",
			req: &proto.GetKeyBySHA3384Request{
				Sha3384: "sha123",
			},
			mockReturn:  cerror.NewCustomError(cerror.ResourceNotFound, "key not found"),
			expectError: true,
			errorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Setup the mock expectation.
			if key, ok := tc.mockReturn.(*models.Key); ok {
				mockRepo.On("GetKeyBySHA3384", mock.Anything, tc.req.Sha3384).
					Return(key, nil).Once()
			} else if errVal, ok := tc.mockReturn.(*cerror.CustomError); ok {
				mockRepo.On("GetKeyBySHA3384", mock.Anything, tc.req.Sha3384).
					Return(nil, errVal).Once()
			} else {
				t.Fatalf("Unexpected type in mockReturn: %T", tc.mockReturn)
			}

			// Call the service function.
			resp, _ := service.GetKey(context.Background(), tc.req)
			t.Logf("DEBUG: performed GetKey, resp = %+v", resp)

			// Assertions.
			if tc.expectError {
				assert.NotNil(t, resp.Errors, "Expected an error response")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors in response")
				assert.Equal(t, tc.req.Sha3384, resp.Sha3384, "Expected SHA3384 to match")
				// Optionally, also check that the key name and encoded public key match.
				if key, ok := tc.mockReturn.(*models.Key); ok {
					assert.Equal(t, key.Name, resp.Name, "Expected key name to match")
					assert.Equal(t, key.EncodedPublicKey, resp.EncodedPublicKey, "Expected encoded public key to match")
				}
			}

			// Verify that the repository method was called.
			mockRepo.AssertCalled(t, "GetKeyBySHA3384", mock.Anything, tc.req.Sha3384)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetKeysByAccountIdLogic(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})
	// Create a valid account ID string and its parsed form.
	validID := uuid.New().String()
	parsedID := uuid.MustParse(validID)

	// Define two sample key models.
	key1 := &models.Key{
		Name:             "KeyOne",
		SHA3384:          "sha1",
		EncodedPublicKey: "encodedKey1",
		CreatedAt:        ptrTime(time.Now().Add(-2 * time.Hour)),
		Until:            ptrTime(time.Now().Add(24 * time.Hour)),
	}
	key2 := &models.Key{
		Name:             "KeyTwo",
		SHA3384:          "sha2",
		EncodedPublicKey: "encodedKey2",
		CreatedAt:        ptrTime(time.Now().Add(-3 * time.Hour)),
		Until:            ptrTime(time.Now().Add(48 * time.Hour)),
	}

	testCases := []struct {
		name             string
		req              *proto.GetKeysByAccountIdRequest
		repoReturn       interface{} // Either []*models.Key or *cerror.CustomError
		expectError      bool
		errorCode        string
		expectedKeyCount int
	}{
		{
			name: "Successful retrieval",
			req: &proto.GetKeysByAccountIdRequest{
				AccountId: validID,
			},
			repoReturn:       []*models.Key{key1, key2},
			expectError:      false,
			expectedKeyCount: 2,
		},
		{
			name: "Invalid UUID",
			req: &proto.GetKeysByAccountIdRequest{
				AccountId: "invalid-uuid",
			},
			repoReturn:       nil, // no repository call is made
			expectError:      true,
			errorCode:        cerror.BadRequest,
			expectedKeyCount: 0,
		},
		{
			name: "Repository error",
			req: &proto.GetKeysByAccountIdRequest{
				AccountId: validID,
			},
			repoReturn:       cerror.NewCustomError(cerror.ResourceNotFound, "no keys found"),
			expectError:      true,
			errorCode:        cerror.ResourceNotFound,
			expectedKeyCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// If the account ID is valid, set up the repository expectation.
			if _, err := uuid.Parse(tc.req.AccountId); err == nil {
				if keys, ok := tc.repoReturn.([]*models.Key); ok {
					mockRepo.On("GetKeysByAccountID", mock.Anything, parsedID).
						Return(keys, nil).Once()
				} else if errVal, ok := tc.repoReturn.(*cerror.CustomError); ok {
					mockRepo.On("GetKeysByAccountID", mock.Anything, parsedID).
						Return(nil, errVal).Once()
				}
			}

			// Call the service function.
			resp, _ := service.GetKeysByAccountId(context.Background(), tc.req)
			t.Logf("DEBUG: performed GetKeysByAccountId, resp = %+v", resp)

			// Assertions.
			if tc.expectError {
				assert.NotNil(t, resp.Errors, "Expected an error response")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
				assert.Equal(t, 0, len(resp.Keys), "Expected no keys in error response")
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors")
				assert.Equal(t, tc.expectedKeyCount, len(resp.Keys), "Expected number of keys to match")
				// Optionally, check that key fields are populated.
				if len(resp.Keys) > 0 {
					assert.NotEmpty(t, resp.Keys[0].Name, "Expected key name to be set")
					assert.NotEmpty(t, resp.Keys[0].Sha3384, "Expected SHA3384 to be set")
					assert.NotEmpty(t, resp.Keys[0].EncodedPublicKey, "Expected encoded public key to be set")
					// You can also check the Timestamps if desired.
				}
			}

			// If a valid UUID was provided, verify that the repository call was made.
			if _, err := uuid.Parse(tc.req.AccountId); err == nil {
				mockRepo.AssertCalled(t, "GetKeysByAccountID", mock.Anything, parsedID)
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestPatchAccountByEmail(t *testing.T) {
	mockRepo := new(repository.MockAccountRepository)
	service := NewAccountService(mockRepo, &config.Config{})

	// Create dummy error values.
	getAccountErr := cerror.NewCustomError(cerror.ResourceNotFound, "account not found")
	updateAccountErr := cerror.NewCustomError(cerror.InternalServerError, "failed to update account")

	id := uuid.New()
	testCases := []struct {
		name                   string
		req                    *proto.PatchAccountByEmailRequest
		originalAccount        *models.Account
		updateAccountReturn    interface{} // Either *models.Account or *cerror.CustomError
		expectError            bool
		errorCode              string
		expectedShortNamespace string
	}{
		{
			name: "Successful patch with username provided",
			req: &proto.PatchAccountByEmailRequest{
				Email:    "test@example.com",
				Username: "new_username",
			},
			originalAccount: &models.Account{
				ID:       id,
				Email:    "test@example.com",
				Username: "old_username",
			},
			updateAccountReturn: &models.Account{
				ID:       id,
				Email:    "test@example.com",
				Username: "new_username",
			},
			expectError:            false,
			expectedShortNamespace: "new_username",
		},
		{
			name: "Successful patch with empty username (no update)",
			req: &proto.PatchAccountByEmailRequest{
				Email:    "test@example.com",
				Username: "",
			},
			originalAccount: &models.Account{
				ID:       id,
				Email:    "test@example.com",
				Username: "old_username",
			},
			// When no username is provided, the account remains unchanged.
			updateAccountReturn: &models.Account{
				ID:       id,
				Email:    "test@example.com",
				Username: "old_username",
			},
			expectError:            false,
			expectedShortNamespace: "old_username",
		},
		{
			name: "Error in GetAccountByEmail",
			req: &proto.PatchAccountByEmailRequest{
				Email:    "nonexistent@example.com",
				Username: "whatever",
			},
			originalAccount:        nil, // not used
			updateAccountReturn:    nil, // UpdateAccount is never called.
			expectError:            true,
			errorCode:              getAccountErr.GetCode(),
			expectedShortNamespace: "",
		},
		{
			name: "Error in UpdateAccount",
			req: &proto.PatchAccountByEmailRequest{
				Email:    "test@example.com",
				Username: "new_username",
			},
			originalAccount: &models.Account{
				ID:       id,
				Email:    "test@example.com",
				Username: "old_username",
			},
			updateAccountReturn:    updateAccountErr,
			expectError:            true,
			errorCode:              updateAccountErr.GetCode(),
			expectedShortNamespace: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up repository expectation for GetAccountByEmail if an original account is expected.
			if tc.originalAccount != nil {
				mockRepo.On("GetAccountByEmail", mock.Anything, tc.req.Email, ([]string)(nil)).
					Return(tc.originalAccount, nil).Once()
			} else {
				mockRepo.On("GetAccountByEmail", mock.Anything, tc.req.Email, ([]string)(nil)).
					Return(nil, getAccountErr).Once()
			}

			// If GetAccountByEmail succeeded, set expectation for UpdateAccount.
			if tc.originalAccount != nil {
				if upd, ok := tc.updateAccountReturn.(*models.Account); ok {
					// Verify that the account passed to UpdateAccount has the expected username.
					mockRepo.On("UpdateAccount", mock.Anything, mock.MatchedBy(func(a *models.Account) bool {
						if tc.req.Username != "" {
							return a.Username == tc.req.Username
						}
						return a.Username == tc.originalAccount.Username
					})).
						Return(upd, nil).Once()
				} else if errVal, ok := tc.updateAccountReturn.(*cerror.CustomError); ok {
					mockRepo.On("UpdateAccount", mock.Anything, mock.Anything).
						Return(nil, errVal).Once()
				}
			}

			// Call the service function.
			resp, _ := service.PatchAccountByEmail(context.Background(), tc.req)
			t.Logf("DEBUG: performed PatchAccountByEmail, resp = %+v", resp)

			// Assertions.
			if tc.expectError {
				assert.NotNil(t, resp.Errors, "Expected an error response")
				assert.GreaterOrEqual(t, len(resp.Errors), 1, "Expected at least one error")
				assert.Equal(t, tc.errorCode, resp.Errors[0].Code, "Expected error code to match")
				assert.Equal(t, "", resp.ShortNamespace, "Expected no short namespace on error")
			} else {
				assert.Equal(t, 0, len(resp.Errors), "Did not expect errors")
				assert.Equal(t, tc.expectedShortNamespace, resp.ShortNamespace, "Expected short namespace to match")
			}

			// Verify that all expectations were met.
			mockRepo.AssertExpectations(t)
		})
	}
}
