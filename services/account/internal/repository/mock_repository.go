package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/account/internal/models"
)

// MockAccountRepository is a mock implementation of IAccountRepository for testing.
type MockAccountRepository struct {
	mock.Mock
}

var _ IAccountRepository = (*MockAccountRepository)(nil)

// Mock CreateAccount
func (m *MockAccountRepository) CreateAccount(ctx context.Context, account *models.Account) (*models.Account, *cerror.CustomError) {
	args := m.Called(ctx, account)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Account), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock UpdateAccount
func (m *MockAccountRepository) UpdateAccount(ctx context.Context, account *models.Account) (*models.Account, *cerror.CustomError) {
	args := m.Called(ctx, account)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Account), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock DeleteAccount
func (m *MockAccountRepository) DeleteAccount(ctx context.Context, accountID uuid.UUID) *cerror.CustomError {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*cerror.CustomError)
}

// Mock GetAccountByEmail
func (m *MockAccountRepository) GetAccountByEmail(ctx context.Context, email string, associations []string) (*models.Account, *cerror.CustomError) {
	args := m.Called(ctx, email, associations)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Account), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock GetAccountByID
func (m *MockAccountRepository) GetAccountByID(ctx context.Context, accountID uuid.UUID, associations []string) (*models.Account, *cerror.CustomError) {
	args := m.Called(ctx, accountID, associations)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Account), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock GetAccountByUsername
func (m *MockAccountRepository) GetAccountByUsername(ctx context.Context, username string, associations []string) (*models.Account, *cerror.CustomError) {
	args := m.Called(ctx, username, associations)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Account), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock AddKeyToAccountByEmail
func (m *MockAccountRepository) AddKeyToAccountByEmail(ctx context.Context, name, sha3384, encodedPublicKey, accountEmail string) (*models.Key, *cerror.CustomError) {
	args := m.Called(ctx, name, sha3384, encodedPublicKey, accountEmail)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Key), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock GetKeyBySHA3384
func (m *MockAccountRepository) GetKeyBySHA3384(ctx context.Context, sha3384 string) (*models.Key, *cerror.CustomError) {
	args := m.Called(ctx, sha3384)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Key), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock GetKeysByAccountID
func (m *MockAccountRepository) GetKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]*models.Key, *cerror.CustomError) {
	args := m.Called(ctx, accountID)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.Key), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock GetSSHKeysByAccountID
func (m *MockAccountRepository) GetSSHKeysByAccountID(ctx context.Context, accountID uuid.UUID) ([]models.SSHKey, *cerror.CustomError) {
	args := m.Called(ctx, accountID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.SSHKey), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock FilterKeys
func (m *MockAccountRepository) FilterKeys(ctx context.Context, filter *models.Key, takeFirst bool) (*models.Key, *cerror.CustomError) {
	args := m.Called(ctx, filter, takeFirst)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Key), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// Mock FilterAccounts
func (m *MockAccountRepository) FilterAccounts(ctx context.Context, filter *models.Account, takeFirst bool) ([]*models.Account, *cerror.CustomError) {
	args := m.Called(ctx, filter, takeFirst)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.Account), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}
