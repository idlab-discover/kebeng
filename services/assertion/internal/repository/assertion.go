package repository

import (
	"context"
	"fmt"

	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/snapcore/snapd/asserts"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/sirupsen/logrus"
)

const (
	ACCOUNT         = "account_assertion"
	ACCOUNTKEY      = "account_key_assertion"
	SNAPREVISION    = "snap_revision_assertion"
	SNAPDECLARATION = "snap_declaration_assertion"
	SNAPBUILD       = "snap_build_assertion"
)

type ICollection interface {
	InsertOne(context.Context, interface{}, ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	FindOne(context.Context, interface{}, ...*options.FindOneOptions) SingleResult
}

type IAssertionRepository interface {
	AddAccountKeyAssertion(ctx context.Context, el *cerror.ErrorList, public_key_SHA3_384, signature string, revision uint32, account_id uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError)
	AddSnapRevisionAssertion(ctx context.Context, el *cerror.ErrorList, snap_sha3_384, signature string, developer_id, snap_entry_id uuid.UUID) (*model.SnapRevisionAssertion, *cerror.CustomError)
	AddSnapDeclarationAssertion(ctx context.Context, el *cerror.ErrorList, signature string, revision uint32, snapEntryId, publisherId uuid.UUID) (*model.SnapDeclarationAssertion, *cerror.CustomError)
	AddSnapBuildAssertion(ctx context.Context, el *cerror.ErrorList, signature string, snap_entry_id, account_id uuid.UUID) (*model.SnapBuildAssertion, *cerror.CustomError)
	AddAccountAssertion(ctx context.Context, el *cerror.ErrorList, signature string, revision uint32, account_id uuid.UUID) (*model.AccountAssertion, *cerror.CustomError)

	GetAccountKeyAssertionByPublicKeySha(ctx context.Context, el *cerror.ErrorList, public_key_SHA3_384 string) (*model.AccountKeyAssertion, *cerror.CustomError)
	GetLatestAccountKeyAssertion(ctx context.Context, el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError)
	GetSnapRevisionAssertionBySHA3_384(ctx context.Context, el *cerror.ErrorList, snap_sha3_384 string) (*model.SnapRevisionAssertion, *cerror.CustomError)
	GetSnapDeclarationAssertionBySnapEntryID(ctx context.Context, el *cerror.ErrorList, snap_entry_id uuid.UUID) (*model.SnapDeclarationAssertion, *cerror.CustomError)
	GetLatestSnapDeclarationAssertion(ctx context.Context, el *cerror.ErrorList, snap_entry_id uuid.UUID) (*model.SnapDeclarationAssertion, *cerror.CustomError)
	GetAccountAssertionByAccountID(ctx context.Context, el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountAssertion, *cerror.CustomError)
	GetLatestAccountAssertionByAccountID(ctx context.Context, el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountAssertion, *cerror.CustomError)
}

type AssertionRepository struct {
	mongoClient          *mongo.Client
	AssertionCollections map[string]ICollection
}

func NewAssertionRepository(cfg *config.Config, mongoClient *mongo.Client) IAssertionRepository {
	dbName := cfg.MongoDBDB

	return &AssertionRepository{
		mongoClient: mongoClient,
		AssertionCollections: map[string]ICollection{
			ACCOUNT:         &MongoCollection{col: mongoClient.Database(dbName).Collection(ACCOUNT)},
			ACCOUNTKEY:      &MongoCollection{col: mongoClient.Database(dbName).Collection(ACCOUNTKEY)},
			SNAPREVISION:    &MongoCollection{col: mongoClient.Database(dbName).Collection(SNAPREVISION)},
			SNAPDECLARATION: &MongoCollection{col: mongoClient.Database(dbName).Collection(SNAPDECLARATION)},
			SNAPBUILD:       &MongoCollection{col: mongoClient.Database(dbName).Collection(SNAPBUILD)},
		},
	}
}

func (r *AssertionRepository) AddAccountKeyAssertion(
	ctx context.Context,
	el *cerror.ErrorList,
	publicKeySHA3_384, signature string,
	revision uint32,
	accountID uuid.UUID,
) (*model.AccountKeyAssertion, *cerror.CustomError) {
	assertion := &model.AccountKeyAssertion{
		ID:                       uuid.New(),
		AccountID:                accountID,
		Type:                     asserts.AccountKeyType.Name,
		RevisionSequenceNumber:   revision,
		PublicKeySha3_384Encoded: publicKeySHA3_384,
		Signature:                signature,
	}

	_, err := r.AssertionCollections[ACCOUNTKEY].InsertOne(ctx, assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert account key assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddSnapRevisionAssertion(
	ctx context.Context,
	el *cerror.ErrorList,
	snapSHA3_384, signature string,
	developerID, snapEntryID uuid.UUID,
) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	assertion := &model.SnapRevisionAssertion{
		ID:           uuid.New(),
		Type:         asserts.SnapRevisionType.Name,
		SnapEntryID:  snapEntryID,
		DeveloperID:  developerID,
		SnapSHA3_384: snapSHA3_384,
		Signature:    signature,
	}

	_, err := r.AssertionCollections[SNAPREVISION].InsertOne(ctx, assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert snap revision assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddSnapDeclarationAssertion(
	ctx context.Context,
	el *cerror.ErrorList,
	signature string,
	revision uint32,
	snapEntryId, publisherId uuid.UUID,
) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	assertion := &model.SnapDeclarationAssertion{
		ID:          uuid.New(),
		Type:        asserts.SnapDeclarationType.Name,
		Revision:    revision,
		SnapEntryID: snapEntryId,
		Signature:   signature,
	}

	_, err := r.AssertionCollections[SNAPDECLARATION].InsertOne(ctx, assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert snap declaration assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddSnapBuildAssertion(
	ctx context.Context,
	el *cerror.ErrorList,
	signature string,
	snapEntryID, accountID uuid.UUID,
) (*model.SnapBuildAssertion, *cerror.CustomError) {
	assertion := &model.SnapBuildAssertion{
		ID:          uuid.New(),
		SnapEntryID: snapEntryID,
		DeveloperID: accountID,
		Type:        asserts.SnapBuildType.Name,
		Signature:   signature,
	}

	_, err := r.AssertionCollections[SNAPBUILD].InsertOne(ctx, assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert snap build assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddAccountAssertion(
	ctx context.Context,
	el *cerror.ErrorList,
	signature string,
	revision uint32,
	accountID uuid.UUID,
) (*model.AccountAssertion, *cerror.CustomError) {
	assertion := &model.AccountAssertion{
		ID:        uuid.New(),
		Type:      asserts.AccountType.Name,
		AccountID: accountID,
		Revision:  revision,
		Signature: signature,
	}

	_, err := r.AssertionCollections[ACCOUNT].InsertOne(ctx, assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert account assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetAccountKeyAssertionByPublicKeySha(
	ctx context.Context,
	el *cerror.ErrorList,
	publicKeySHA3_384 string,
) (*model.AccountKeyAssertion, *cerror.CustomError) {
	filter := bson.M{"publickeysha3_384encoded": publicKeySHA3_384}

	assertion, err := findOne[model.AccountKeyAssertion](ctx, r.AssertionCollections[ACCOUNTKEY], filter, nil)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to retrieve account key assertion from MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetLatestAccountKeyAssertion(
	ctx context.Context,
	el *cerror.ErrorList,
	accountID uuid.UUID,
) (*model.AccountKeyAssertion, *cerror.CustomError) {
	filter := bson.M{"accountid": accountID}
	opts := options.FindOne().SetSort(bson.D{{Key: "revisionsequencenumber", Value: -1}})

	assertion, err := findOne[model.AccountKeyAssertion](ctx, r.AssertionCollections[ACCOUNTKEY], filter, opts)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to get latest account key assertion by account id: %s", accountID.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetSnapRevisionAssertionBySHA3_384(
	ctx context.Context,
	el *cerror.ErrorList,
	snapSHA3_384 string,
) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	filter := bson.M{"snapsha3_384": snapSHA3_384}

	assertion, err := findOne[model.SnapRevisionAssertion](ctx, r.AssertionCollections[SNAPREVISION], filter, nil)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve snap revision assertion from MongoDB for SHA3_384: %s", snapSHA3_384))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetSnapDeclarationAssertionBySnapEntryID(
	ctx context.Context,
	el *cerror.ErrorList,
	snapID uuid.UUID,
) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	filter := bson.M{"snapid": snapID}

	assertion, err := findOne[model.SnapDeclarationAssertion](ctx, r.AssertionCollections[SNAPDECLARATION], filter, nil)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve snap declaration assertion from MongoDB for snap id: %s", snapID))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetLatestSnapDeclarationAssertion(
	ctx context.Context,
	el *cerror.ErrorList,
	snapEntryID uuid.UUID,
) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	filter := bson.M{"snapid": snapEntryID}
	opts := options.FindOne().SetSort(bson.D{{Key: "revision", Value: -1}})

	assertion, err := findOne[model.SnapDeclarationAssertion](ctx, r.AssertionCollections[SNAPDECLARATION], filter, opts)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve latest snap declaration assertion from MongoDB for snap id: %s", snapEntryID))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetAccountAssertionByAccountID(
	ctx context.Context,
	el *cerror.ErrorList,
	accountID uuid.UUID,
) (*model.AccountAssertion, *cerror.CustomError) {
	filter := bson.M{"accountid": accountID}

	assertion, err := findOne[model.AccountAssertion](ctx, r.AssertionCollections[ACCOUNT], filter, nil)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve account assertion from MongoDB for account id: %s", accountID.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetLatestAccountAssertionByAccountID(
	ctx context.Context,
	el *cerror.ErrorList,
	accountID uuid.UUID,
) (*model.AccountAssertion, *cerror.CustomError) {
	filter := bson.M{"accountid": accountID}
	opts := options.FindOne().SetSort(bson.D{{Key: "revision", Value: -1}})

	assertion, err := findOne[model.AccountAssertion](ctx, r.AssertionCollections[ACCOUNT], filter, opts)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve latest account assertion from MongoDB for account id: %s", accountID.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

// ============= HELPERS =============
func findOne[T any](ctx context.Context, collection ICollection, filter interface{}, opts *options.FindOneOptions) (*T, error) {
	var result T
	err := collection.FindOne(ctx, filter, opts).Decode(&result)
	return &result, err
}
