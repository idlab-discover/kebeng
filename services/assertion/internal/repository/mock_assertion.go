package repository

import (
	"context"

	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/stretchr/testify/mock"
)

// Ensure that MockAssertionRepository implements the interface.
var _ IAssertionRepository = (*MockAssertionRepository)(nil)

// MockAssertionRepository is a mock implementation of IAssertionRepository.
type MockAssertionRepository struct {
	mock.Mock
}

// AddAccountKeyAssertion mocks AddAccountKeyAssertion method.
func (m *MockAssertionRepository) AddAccountKeyAssertion(ctx context.Context, el *cerror.ErrorList, public_key_SHA3_384, signature string, revision uint32, account_id uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, public_key_SHA3_384, signature, revision, account_id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountKeyAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// AddSnapRevisionAssertion mocks AddSnapRevisionAssertion method.
func (m *MockAssertionRepository) AddSnapRevisionAssertion(ctx context.Context, el *cerror.ErrorList, snap_sha3_384, signature string, developer_id, snap_entry_id uuid.UUID) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, snap_sha3_384, signature, developer_id, snap_entry_id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapRevisionAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// AddSnapDeclarationAssertion mocks AddSnapDeclarationAssertion method.
func (m *MockAssertionRepository) AddSnapDeclarationAssertion(ctx context.Context, el *cerror.ErrorList, signature string, revision uint32, snapEntryId, publisherId uuid.UUID) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, signature, revision, snapEntryId, publisherId)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapDeclarationAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// AddSnapBuildAssertion mocks AddSnapBuildAssertion method.
func (m *MockAssertionRepository) AddSnapBuildAssertion(ctx context.Context, el *cerror.ErrorList, signature string, snap_entry_id, account_id uuid.UUID) (*model.SnapBuildAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, signature, snap_entry_id, account_id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapBuildAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// AddAccountAssertion mocks AddAccountAssertion method.
func (m *MockAssertionRepository) AddAccountAssertion(ctx context.Context, el *cerror.ErrorList, signature string, revision uint32, account_id uuid.UUID) (*model.AccountAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, signature, revision, account_id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// GetAccountKeyAssertionByPublicKeySha mocks GetAccountKeyAssertionByPublicKeySha method.
func (m *MockAssertionRepository) GetAccountKeyAssertionByPublicKeySha(ctx context.Context, el *cerror.ErrorList, public_key_SHA3_384 string) (*model.AccountKeyAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, public_key_SHA3_384)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountKeyAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// GetLatestAccountKeyAssertion mocks GetLatestAccountKeyAssertion method.
func (m *MockAssertionRepository) GetLatestAccountKeyAssertion(ctx context.Context, el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, account_id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountKeyAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// GetSnapRevisionAssertionBySHA3_384 mocks GetSnapRevisionAssertionBySHA3_384 method.
func (m *MockAssertionRepository) GetSnapRevisionAssertionBySHA3_384(ctx context.Context, el *cerror.ErrorList, snap_sha3_384 string) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, snap_sha3_384)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapRevisionAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// GetSnapDeclarationAssertionBySnapEntryID mocks GetSnapDeclarationAssertionBySnapEntryID method.
func (m *MockAssertionRepository) GetSnapDeclarationAssertionBySnapEntryID(ctx context.Context, el *cerror.ErrorList, snap_entry_id uuid.UUID) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, snap_entry_id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapDeclarationAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// GetLatestSnapDeclarationAssertion mocks GetLatestSnapDeclarationAssertion method.
func (m *MockAssertionRepository) GetLatestSnapDeclarationAssertion(ctx context.Context, el *cerror.ErrorList, snap_entry_id uuid.UUID) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, snap_entry_id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapDeclarationAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// GetAccountAssertionByAccountID mocks GetAccountAssertionByAccountID method.
func (m *MockAssertionRepository) GetAccountAssertionByAccountID(ctx context.Context, el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, account_id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// GetLatestAccountAssertionByAccountID mocks GetLatestAccountAssertionByAccountID method.
func (m *MockAssertionRepository) GetLatestAccountAssertionByAccountID(ctx context.Context, el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, account_id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// =========================
type MockCollection struct {
	mock.Mock
}

func (m *MockCollection) InsertOne(ctx context.Context, doc interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, doc)
	return nil, args.Error(1)
}

func (m *MockCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) SingleResult {
	args := m.Called(ctx, filter)
	return args.Get(0).(SingleResult)
}
