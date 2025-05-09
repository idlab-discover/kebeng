package repository

import (
	"context"
	"time"

	"github.com/idlab-discover/kebeng/services/assertion/internal/model"

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

// AddAssertion mocks AddAssertion method.
func (m *MockAssertionRepository) AddAssertion(ctx context.Context, snapEntryId uuid.UUID, assertionString string) (*model.Assertion, *cerror.CustomError) {
	args := m.Called(ctx, snapEntryId, assertionString)
	if args.Get(0) != nil {
		return args.Get(0).(*model.Assertion), nil
	}
	return nil, args.Get(1).(*cerror.CustomError)
}

// AddAccountKeyAssertion mocks AddAccountKeyAssertion method.
func (m *MockAssertionRepository) AddAccountKeyAssertion(ctx context.Context, el *cerror.ErrorList, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name string, revision uint32, account_id uuid.UUID, since, until time.Time, body []byte, body_length uint64, signature string) (*model.AccountKeyAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name, revision, account_id, since, until, body, body_length, signature)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountKeyAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// AddSnapRevisionAssertion mocks AddSnapRevisionAssertion method.
func (m *MockAssertionRepository) AddSnapRevisionAssertion(ctx context.Context, el *cerror.ErrorList, authority_id, snap_sha3_384, sign_key_SHA3_384 string, developer_id, snap_entry_id uuid.UUID, snap_revision_sequence_number uint32, snap_size uint64, timestamp time.Time, signature string) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, authority_id, snap_sha3_384, sign_key_SHA3_384, developer_id, snap_entry_id, snap_revision_sequence_number, snap_size, timestamp, signature)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapRevisionAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockAssertionRepository) AddSnapDeclarationAssertion(ctx context.Context, el *cerror.ErrorList, authorityID, signKey, snapID, snapName, publisherID string, revision uint32, series string, timestamp time.Time, refreshControl []string, aliases []model.Alias, plugs model.Plugs, slots model.Slots, signature string) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, authorityID, signKey, snapID, snapName, publisherID, revision, series, timestamp, refreshControl, aliases, plugs, slots, signature)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapDeclarationAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockAssertionRepository) AddAccountAssertion(ctx context.Context, el *cerror.ErrorList, authority_id, displayName, username, validation string, accountID uuid.UUID, revision uint32, timestamp time.Time, sign_key_SHA3_384, signature string) (*model.AccountAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, authority_id, displayName, username, validation, accountID, revision, timestamp, sign_key_SHA3_384, signature)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

// GetAccountKeyAssertionByPublicKeySha mocks GetAccountKeyAssertionByPublicKeySha method.
func (m *MockAssertionRepository) GetAccountKeyAssertionByPublicKeySha(ctx context.Context, el *cerror.ErrorList, name string) (*model.AccountKeyAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, name)
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

func (m *MockAssertionRepository) GetSnapDeclarationAssertionBySnapID(ctx context.Context, el *cerror.ErrorList, id string) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapDeclarationAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockAssertionRepository) GetLatestSnapDeclarationAssertion(ctx context.Context, el *cerror.ErrorList, snapID string) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, snapID)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapDeclarationAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockAssertionRepository) GetAccountAssertionByAccountID(ctx context.Context, el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, accountID)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockAssertionRepository) GetLatestAccountAssertionByAccountID(ctx context.Context, el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, accountID)
	if args.Get(0) != nil {
		return args.Get(0).(*model.AccountAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}

func (m *MockAssertionRepository) AddSnapBuildAssertion(ctx context.Context, el *cerror.ErrorList, authority_id, sign_key_SHA3_384 string, snap_id, account_id uuid.UUID, grade string, snap_sha3_384 string, snap_size uint64, signature string, timestamp time.Time) (*model.SnapBuildAssertion, *cerror.CustomError) {
	args := m.Called(ctx, el, authority_id, sign_key_SHA3_384, snap_id, account_id, grade, snap_sha3_384, snap_size, signature, timestamp)
	if args.Get(0) != nil {
		return args.Get(0).(*model.SnapBuildAssertion), nil
	}
	el.AddCustomError(args.Get(1).(*cerror.CustomError))
	return nil, args.Get(1).(*cerror.CustomError)
}
