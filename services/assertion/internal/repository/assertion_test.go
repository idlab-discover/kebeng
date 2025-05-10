package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestAddAccountKeyAssertion_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			repository.ACCOUNTKEY: mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, nil)

	el := &cerror.ErrorList{}
	result, cerr := repo.AddAccountKeyAssertion(ctx, el, "auth-id", "public-key-sha3-384", "sign-key-sha3-384", "name", 1, uuid.New(), time.Now(), time.Now().Add(24*time.Hour), nil, 0, "signature")

	assert.Nil(t, cerr)
	assert.NotNil(t, result)
	mockCol.AssertExpectations(t)
}

func TestAddAccountKeyAssertion_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			repository.ACCOUNTKEY: mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, errors.New("mock error"))

	el := &cerror.ErrorList{}
	result, cerr := repo.AddAccountKeyAssertion(ctx, el, "auth-id", "public-key-sha3-384", "sign-key-sha3-384", "name", 1, uuid.New(), time.Now(), time.Now().Add(24*time.Hour), nil, 0, "signature")

	assert.NotNil(t, cerr)
	assert.Nil(t, result)
	mockCol.AssertExpectations(t)
}

func TestAddSnapRevisionAssertion_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_revision_assertion": mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, nil)

	el := &cerror.ErrorList{}
	result, cerr := repo.AddSnapRevisionAssertion(ctx, el, "auth-id", "snap-sha3-384", "sign-key", uuid.New(), uuid.New(), 1, 123, time.Now(), "signature")

	assert.Nil(t, cerr)
	assert.NotNil(t, result)
	mockCol.AssertExpectations(t)
}

func TestAddSnapRevisionAssertion_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_revision_assertion": mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, errors.New("mock error"))

	el := &cerror.ErrorList{}
	result, cerr := repo.AddSnapRevisionAssertion(ctx, el, "auth-id", "snap-sha3-384", "sign-key", uuid.New(), uuid.New(), 1, 123, time.Now(), "signature")

	assert.NotNil(t, cerr)
	assert.Nil(t, result)
	mockCol.AssertExpectations(t)
}

func TestAddSnapDeclarationAssertion_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_declaration_assertion": mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, nil)

	el := &cerror.ErrorList{}
	result, cerr := repo.AddSnapDeclarationAssertion(ctx, el, "auth-id", "sign-key", "snap-id", "snap-name", "publisher-id", 1, "series", time.Now(), []string{"refresh"}, nil, nil, nil, "signature")

	assert.Nil(t, cerr)
	assert.NotNil(t, result)
	mockCol.AssertExpectations(t)
}

func TestAddSnapDeclarationAssertion_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_declaration_assertion": mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, errors.New("mock error"))

	el := &cerror.ErrorList{}
	result, cerr := repo.AddSnapDeclarationAssertion(ctx, el, "auth-id", "sign-key", "snap-id", "snap-name", "publisher-id", 1, "series", time.Now(), []string{"refresh"}, nil, nil, nil, "signature")

	assert.NotNil(t, cerr)
	assert.Nil(t, result)
	mockCol.AssertExpectations(t)
}

func TestAddSnapBuildAssertion_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_build_assertion": mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, nil)

	el := &cerror.ErrorList{}
	result, cerr := repo.AddSnapBuildAssertion(ctx, el, "auth-id", "sign-key", uuid.New(), uuid.New(), "grade", "snap-sha3-384", 123, "signature", time.Now())

	assert.Nil(t, cerr)
	assert.NotNil(t, result)
	mockCol.AssertExpectations(t)
}

func TestAddSnapBuildAssertion_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_build_assertion": mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, errors.New("mock error"))

	el := &cerror.ErrorList{}
	result, cerr := repo.AddSnapBuildAssertion(ctx, el, "auth-id", "sign-key", uuid.New(), uuid.New(), "grade", "snap-sha3-384", 123, "signature", time.Now())

	assert.NotNil(t, cerr)
	assert.Nil(t, result)
	mockCol.AssertExpectations(t)
}

func TestAddAccountAssertion_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_assertion": mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, nil)

	el := &cerror.ErrorList{}
	result, cerr := repo.AddAccountAssertion(ctx, el, "auth-id", "display-name", "username", "validation", uuid.New(), 1, time.Now(), "sign-key", "signature")

	assert.Nil(t, cerr)
	assert.NotNil(t, result)
	mockCol.AssertExpectations(t)
}

func TestAddAccountAssertion_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_assertion": mockCol,
		},
	}
	ctx := context.Background()
	mockCol.On("InsertOne", mock.Anything, mock.Anything).Return(nil, errors.New("mock error"))

	el := &cerror.ErrorList{}
	result, cerr := repo.AddAccountAssertion(ctx, el, "auth-id", "display-name", "username", "validation", uuid.New(), 1, time.Now(), "sign-key", "signature")

	assert.NotNil(t, cerr)
	assert.Nil(t, result)
	mockCol.AssertExpectations(t)
}

func TestGetAccountKeyAssertionByPublicKeySha_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			repository.ACCOUNTKEY: mockCol,
		},
	}

	expected := model.AccountKeyAssertion{
		ID:                       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		PublicKeySha3_384Encoded: "public-key-sha3-384",
		AccountID:                uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Name:                     "test",
		Signature:                "signed-assertion",
	}

	fakeResult := &FakeSingleResult{Doc: expected}
	mockCol.
		On("FindOne", mock.Anything, bson.M{"publickeysha3_384encoded": "public-key-sha3-384"}).
		Return(fakeResult)

	el := cerror.NewErrorList()
	got, cerr := repo.GetAccountKeyAssertionByPublicKeySha(context.Background(), el, "public-key-sha3-384")

	assert.Nil(t, cerr)
	assert.Equal(t, expected.PublicKeySha3_384Encoded, got.PublicKeySha3_384Encoded)
	assert.Equal(t, expected.AccountID, got.AccountID)
	assert.Equal(t, expected.Name, got.Name)
	assert.Equal(t, expected.Signature, got.Signature)

	mockCol.AssertExpectations(t)
}

func TestGetAccountKeyAssertionByPublicKeySha_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			repository.ACCOUNTKEY: mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, bson.M{"publickeysha3_384encoded": "public-key-sha3-384"}).
		Return(&FakeSingleResult{Err: errors.New("mock error")})

	el := cerror.NewErrorList()
	got, cerr := repo.GetAccountKeyAssertionByPublicKeySha(context.Background(), el, "public-key-sha3-384")

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetAccountKeyAssertionByPublicKeySha_NotFound(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			repository.ACCOUNTKEY: mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, bson.M{"publickeysha3_384encoded": "public-key-sha3-384"}).
		Return(&FakeSingleResult{Err: mongo.ErrNoDocuments})

	el := cerror.NewErrorList()
	got, cerr := repo.GetAccountKeyAssertionByPublicKeySha(context.Background(), el, "public-key-sha3-384")

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetLatestAccountKeyAssertion_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_key_assertion": mockCol,
		},
	}

	expected := model.AccountKeyAssertion{
		ID:                       uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		PublicKeySha3_384Encoded: "public-key-sha3-384",
		AccountID:                uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		Name:                     "latest-test",
		Signature:                "latest-signed-assertion",
	}

	fakeResult := &FakeSingleResult{Doc: expected}
	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(fakeResult)

	el := cerror.NewErrorList()
	got, cerr := repo.GetLatestAccountKeyAssertion(context.Background(), el, uuid.MustParse("33333333-3333-3333-3333-333333333333"))

	assert.Nil(t, cerr)
	assert.Equal(t, expected.PublicKeySha3_384Encoded, got.PublicKeySha3_384Encoded)
	assert.Equal(t, expected.AccountID, got.AccountID)
	assert.Equal(t, expected.Name, got.Name)
	assert.Equal(t, expected.Signature, got.Signature)

	mockCol.AssertExpectations(t)
}

func TestGetLatestAccountKeyAssertion_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_key_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(&FakeSingleResult{Err: errors.New("mock error")})

	el := cerror.NewErrorList()
	got, cerr := repo.GetLatestAccountKeyAssertion(context.Background(), el, uuid.MustParse("33333333-3333-3333-3333-333333333333"))

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetLatestAccountKeyAssertion_NotFound(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_key_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(&FakeSingleResult{Err: mongo.ErrNoDocuments})

	el := cerror.NewErrorList()
	got, cerr := repo.GetLatestAccountKeyAssertion(context.Background(), el, uuid.MustParse("33333333-3333-3333-3333-333333333333"))

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetSnapRevisionAssertionBySHA3_384_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_revision_assertion": mockCol,
		},
	}

	expected := model.SnapRevisionAssertion{
		ID:           uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		SnapSHA3_384: "snap-sha3-384",
		Signature:    "signed-assertion",
	}

	fakeResult := &FakeSingleResult{Doc: expected}
	mockCol.
		On("FindOne", mock.Anything, bson.M{"snapsha3_384": "snap-sha3-384"}).
		Return(fakeResult)

	el := cerror.NewErrorList()
	got, cerr := repo.GetSnapRevisionAssertionBySHA3_384(context.Background(), el, "snap-sha3-384")

	assert.Nil(t, cerr)
	assert.Equal(t, expected.SnapSHA3_384, got.SnapSHA3_384)
	assert.Equal(t, expected.Signature, got.Signature)

	mockCol.AssertExpectations(t)
}

func TestGetSnapRevisionAssertionBySHA3_384_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_revision_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, bson.M{"snapsha3_384": "snap-sha3-384"}).
		Return(&FakeSingleResult{Err: errors.New("mock error")})

	el := cerror.NewErrorList()
	got, cerr := repo.GetSnapRevisionAssertionBySHA3_384(context.Background(), el, "snap-sha3-384")

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetSnapRevisionAssertionBySHA3_384_NotFound(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_revision_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, bson.M{"snapsha3_384": "snap-sha3-384"}).
		Return(&FakeSingleResult{Err: mongo.ErrNoDocuments})

	el := cerror.NewErrorList()
	got, cerr := repo.GetSnapRevisionAssertionBySHA3_384(context.Background(), el, "snap-sha3-384")

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetSnapDeclarationAssertionBySnapID_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_declaration_assertion": mockCol,
		},
	}

	expected := model.SnapDeclarationAssertion{
		ID:        uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		SnapID:    "snap-id",
		Signature: "signed-assertion",
	}

	fakeResult := &FakeSingleResult{Doc: expected}
	mockCol.
		On("FindOne", mock.Anything, bson.M{"snapid": "snap-id"}).
		Return(fakeResult)

	el := cerror.NewErrorList()
	got, cerr := repo.GetSnapDeclarationAssertionBySnapID(context.Background(), el, "snap-id")

	assert.Nil(t, cerr)
	assert.Equal(t, expected.SnapID, got.SnapID)
	assert.Equal(t, expected.Signature, got.Signature)

	mockCol.AssertExpectations(t)
}

func TestGetSnapDeclarationAssertionBySnapID_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_declaration_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, bson.M{"snapid": "snap-id"}).
		Return(&FakeSingleResult{Err: errors.New("mock error")})

	el := cerror.NewErrorList()
	got, cerr := repo.GetSnapDeclarationAssertionBySnapID(context.Background(), el, "snap-id")

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetSnapDeclarationAssertionBySnapID_NotFound(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_declaration_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, bson.M{"snapid": "snap-id"}).
		Return(&FakeSingleResult{Err: mongo.ErrNoDocuments})

	el := cerror.NewErrorList()
	got, cerr := repo.GetSnapDeclarationAssertionBySnapID(context.Background(), el, "snap-id")

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetLatestSnapDeclarationAssertion_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_declaration_assertion": mockCol,
		},
	}

	expected := model.SnapDeclarationAssertion{
		ID:        uuid.MustParse("77777777-7777-7777-7777-777777777777"),
		SnapID:    "snap-id",
		Signature: "latest-signed-assertion",
	}

	fakeResult := &FakeSingleResult{Doc: expected}
	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(fakeResult)

	el := cerror.NewErrorList()
	got, cerr := repo.GetLatestSnapDeclarationAssertion(context.Background(), el, "snap-id")

	assert.Nil(t, cerr)
	assert.Equal(t, expected.SnapID, got.SnapID)
	assert.Equal(t, expected.Signature, got.Signature)

	mockCol.AssertExpectations(t)
}

func TestGetLatestSnapDeclarationAssertion_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_declaration_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(&FakeSingleResult{Err: errors.New("mock error")})

	el := cerror.NewErrorList()
	got, cerr := repo.GetLatestSnapDeclarationAssertion(context.Background(), el, "snap-id")

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetLatestSnapDeclarationAssertion_NotFound(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"snap_declaration_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(&FakeSingleResult{Err: mongo.ErrNoDocuments})

	el := cerror.NewErrorList()
	got, cerr := repo.GetLatestSnapDeclarationAssertion(context.Background(), el, "snap-id")

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetAccountAssertionByAccountID_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_assertion": mockCol,
		},
	}

	expected := model.AccountAssertion{
		ID:          uuid.MustParse("88888888-8888-8888-8888-888888888888"),
		AccountID:   uuid.MustParse("99999999-9999-9999-9999-999999999999"),
		DisplayName: "test-account",
		Signature:   "signed-assertion",
	}

	fakeResult := &FakeSingleResult{Doc: expected}
	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(fakeResult)

	el := cerror.NewErrorList()
	got, cerr := repo.GetAccountAssertionByAccountID(context.Background(), el, uuid.MustParse("99999999-9999-9999-9999-999999999999"))

	assert.Nil(t, cerr)
	assert.Equal(t, expected.AccountID, got.AccountID)
	assert.Equal(t, expected.DisplayName, got.DisplayName)
	assert.Equal(t, expected.Signature, got.Signature)

	mockCol.AssertExpectations(t)
}

func TestGetAccountAssertionByAccountID_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(&FakeSingleResult{Err: errors.New("mock error")})

	el := cerror.NewErrorList()
	got, cerr := repo.GetAccountAssertionByAccountID(context.Background(), el, uuid.MustParse("99999999-9999-9999-9999-999999999999"))

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetAccountAssertionByAccountID_NotFound(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(&FakeSingleResult{Err: mongo.ErrNoDocuments})

	el := cerror.NewErrorList()
	got, cerr := repo.GetAccountAssertionByAccountID(context.Background(), el, uuid.MustParse("99999999-9999-9999-9999-999999999999"))

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetLatestAccountAssertionByAccountID_Succes(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_assertion": mockCol,
		},
	}

	expected := model.AccountAssertion{
		ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		AccountID:   uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		DisplayName: "latest-account",
		Signature:   "latest-signed-assertion",
	}

	fakeResult := &FakeSingleResult{Doc: expected}
	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(fakeResult)

	el := cerror.NewErrorList()
	got, cerr := repo.GetLatestAccountAssertionByAccountID(context.Background(), el, uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))

	assert.Nil(t, cerr)
	assert.Equal(t, expected.AccountID, got.AccountID)
	assert.Equal(t, expected.DisplayName, got.DisplayName)
	assert.Equal(t, expected.Signature, got.Signature)

	mockCol.AssertExpectations(t)
}

func TestGetLatestAccountAssertionByAccountID_Failure(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(&FakeSingleResult{Err: errors.New("mock error")})

	el := cerror.NewErrorList()
	got, cerr := repo.GetLatestAccountAssertionByAccountID(context.Background(), el, uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

func TestGetLatestAccountAssertionByAccountID_NotFound(t *testing.T) {
	mockCol := new(repository.MockCollection)
	repo := &repository.AssertionRepository{
		AssertionCollections: map[string]repository.ICollection{
			"account_assertion": mockCol,
		},
	}

	mockCol.
		On("FindOne", mock.Anything, mock.Anything).
		Return(&FakeSingleResult{Err: mongo.ErrNoDocuments})

	el := cerror.NewErrorList()
	got, cerr := repo.GetLatestAccountAssertionByAccountID(context.Background(), el, uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))

	assert.NotNil(t, cerr)
	assert.Nil(t, got)

	mockCol.AssertExpectations(t)
}

// ============ Helper functions ============
// test_helpers.go (in je repository_test package)

type FakeSingleResult struct {
	Doc interface{}
	Err error
}

func (f *FakeSingleResult) Decode(v interface{}) error {
	if f.Err != nil {
		return f.Err
	}
	data, _ := bson.Marshal(f.Doc)
	return bson.Unmarshal(data, v)
}
