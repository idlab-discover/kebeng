package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/snapcore/snapd/asserts"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

const (
	ACCOUNT         = "account_assertion"
	ACCOUNTKEY      = "account_key_assertion"
	SNAPREVISION    = "snap_revision_assertion"
	SNAPDECLARATION = "snap_declaration_assertion"
	SNAPBUILD       = "snap_build_assertion"
)

type IAssertionRepository interface {
	AddAssertion(snapEntryId uuid.UUID, assertionString string) (*model.Assertion, *cerror.CustomError)
	AddAccountKeyAssertion(el *cerror.ErrorList, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name string, revision uint32, account_id uuid.UUID, since time.Time, until time.Time, body []byte, body_length uint64, signature string) (*model.AccountKeyAssertion, *cerror.CustomError)
	AddSnapRevisionAssertion(el *cerror.ErrorList, authority_id, snap_sha3_384, sign_key_SHA3_384 string, developer_id, snap_entry_id uuid.UUID, snap_revision_sequence_number uint32, snap_size uint64, timestamp time.Time, signature string) (*model.SnapRevisionAssertion, *cerror.CustomError)
	AddSnapDeclarationAssertion(el *cerror.ErrorList, authorityID, sign_key_SHA3_384, snapID, snapName, publisherID string, revision uint32, series string, timestamp time.Time, refreshControl []string, aliases []model.Alias, plugs model.Plugs, slots model.Slots, signature string) (*model.SnapDeclarationAssertion, *cerror.CustomError)
	AddSnapBuildAssertion(el *cerror.ErrorList, authority_id, sign_key_SHA3_384 string, snap_id, account_id uuid.UUID, grade string, snap_sha3_384 string, snap_size uint64, signature string, timestamp time.Time) (*model.SnapBuildAssertion, *cerror.CustomError)
	AddAccountAssertion(el *cerror.ErrorList, authority_id, displayName, username, validation string, accountID uuid.UUID, revision uint32, timestamp time.Time, sign_key_SHA3_384, signature string) (*model.AccountAssertion, *cerror.CustomError)

	GetAccountKeyAssertionByPublicKeySha(el *cerror.ErrorList, public_key_SHA3_384 string) (*model.AccountKeyAssertion, *cerror.CustomError)
	GetLatestAccountKeyAssertion(el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError)
	GetSnapRevisionAssertionBySHA3_384(el *cerror.ErrorList, snap_sha3_384 string) (*model.SnapRevisionAssertion, *cerror.CustomError)
	GetSnapDeclarationAssertionBySnapID(el *cerror.ErrorList, snapID string) (*model.SnapDeclarationAssertion, *cerror.CustomError)
	GetLatestSnapDeclarationAssertion(el *cerror.ErrorList, snapID string) (*model.SnapDeclarationAssertion, *cerror.CustomError)
	GetAccountAssertionByAccountID(el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountAssertion, *cerror.CustomError)
	GetLatestAccountAssertionByAccountID(el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountAssertion, *cerror.CustomError)
}

type AssertionRepository struct {
	db                   *sqlx.DB
	mongoClient          *mongo.Client
	assertionCollections map[string]*mongo.Collection
}

func NewAssertionRepository(db *sqlx.DB, mongoClient *mongo.Client) IAssertionRepository {
	dbName := "assertion" // Maybe this should be a config value

	return &AssertionRepository{
		db:          db,
		mongoClient: mongoClient,
		assertionCollections: map[string]*mongo.Collection{
			ACCOUNT:         mongoClient.Database(dbName).Collection(ACCOUNT),
			ACCOUNTKEY:      mongoClient.Database(dbName).Collection(ACCOUNTKEY),
			SNAPREVISION:    mongoClient.Database(dbName).Collection(SNAPREVISION),
			SNAPDECLARATION: mongoClient.Database(dbName).Collection(SNAPDECLARATION),
			SNAPBUILD:       mongoClient.Database(dbName).Collection(SNAPBUILD),
		},
	}
}

// TODO: eventually remove this and use correct assertion, leaving this here now for backwards compatibility
func (r *AssertionRepository) AddAssertion(snapEntryId uuid.UUID, assertionString string) (*model.Assertion, *cerror.CustomError) {
	query := `INSERT INTO assertions (snap_entry_id, assertion) VALUES ($1, $2) RETURNING id, assertion`
	assertion := &model.Assertion{}

	err := r.db.Get(assertion, query, snapEntryId, assertionString)
	if err != nil {
		logrus.Errorf("failed to save assertion in database: %v", err)
		return nil, cerror.NewCustomError(cerror.DatabaseError, "failed to save assertion in database")
	}

	return assertion, nil
}

func (r *AssertionRepository) AddAccountKeyAssertion(
	el *cerror.ErrorList,
	authorityID, publicKeySHA3_384, signKeySHA3_384, name string,
	revision uint32,
	accountID uuid.UUID,
	since, until time.Time,
	body []byte,
	bodyLength uint64,
	signature string,
) (*model.AccountKeyAssertion, *cerror.CustomError) {
	assertion := &model.AccountKeyAssertion{
		Type:                     asserts.AccountKeyType.Name,
		RevisionSequenceNumber:   revision,
		PublicKeySha3_384Encoded: publicKeySHA3_384,
		Signature:                signature,
	}

	_, err := r.assertionCollections[ACCOUNTKEY].InsertOne(context.Background(), assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert account key assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddSnapRevisionAssertion(
	el *cerror.ErrorList,
	authorityID, snapSHA3_384, signKeySHA3_384 string,
	developerID, snapEntryID uuid.UUID,
	snapRevisionSequenceNumber uint32,
	snapSize uint64,
	timestamp time.Time,
	signature string,
) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	assertion := &model.SnapRevisionAssertion{
		Type:         asserts.SnapRevisionType.Name,
		SnapSHA3_384: snapSHA3_384,
		Signature:    signature,
	}

	_, err := r.assertionCollections[SNAPREVISION].InsertOne(context.Background(), assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert snap revision assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddSnapDeclarationAssertion(el *cerror.ErrorList, authorityID, signKey, snapID, snapName, publisherID string, revision uint32, series string, timestamp time.Time, refreshControl []string, aliases []model.Alias, plugs model.Plugs, slots model.Slots, signature string) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	// start transaction
	assertion := &model.SnapDeclarationAssertion{
		Type:      asserts.SnapDeclarationType.Name,
		SnapID:    snapID,
		Revision:  revision,
		Signature: signature,
	}

	_, err := r.assertionCollections[SNAPDECLARATION].InsertOne(context.Background(), assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert snap declaration assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddSnapBuildAssertion(
	el *cerror.ErrorList,
	authorityID, signKeySHA3_384 string,
	snapID, accountID uuid.UUID,
	grade, snapSHA3_384 string,
	snapSize uint64,
	signature string,
	timestamp time.Time,
) (*model.SnapBuildAssertion, *cerror.CustomError) {
	assertion := &model.SnapBuildAssertion{
		Type:            asserts.SnapBuildType.Name,
		SignKeySHA3_384: signKeySHA3_384,
		SnapEntryID:     snapID,
		DeveloperID:     accountID,
		SnapSHA3_384:    snapSHA3_384,
		Signature:       signature,
	}

	_, err := r.assertionCollections[SNAPBUILD].InsertOne(context.Background(), assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert snap build assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddAccountAssertion(
	el *cerror.ErrorList,
	authorityID, displayName, username, validation string,
	accountID uuid.UUID,
	revision uint32,
	timestamp time.Time,
	signKeySHA3_384, signature string,
) (*model.AccountAssertion, *cerror.CustomError) {
	assertion := &model.AccountAssertion{
		Type:      asserts.AccountType.Name,
		AccountID: accountID,
		Revision:  revision,
		Signature: signature,
	}

	_, err := r.assertionCollections[ACCOUNT].InsertOne(context.Background(), assertion)
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to insert account assertion in MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetAccountKeyAssertionByPublicKeySha(
	el *cerror.ErrorList,
	publicKeySHA3_384 string,
) (*model.AccountKeyAssertion, *cerror.CustomError) {
	filter := bson.M{"publickeysha3_384encoded": publicKeySHA3_384}

	assertion, err := findOne[model.AccountKeyAssertion](r.assertionCollections[ACCOUNTKEY], filter, nil)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			cerr := cerror.NewCustomError(cerror.ResourceNotFound, "no matching AccountKeyAssertion found")
			el.AddCustomError(cerr)
			return nil, cerr
		}
		cerr := cerror.ConvertError(err, "failed to retrieve account key assertion from MongoDB")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetLatestAccountKeyAssertion(el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError) {
	filter := bson.M{"accountid": accountID}
	opts := options.FindOne().SetSort(bson.D{{Key: "revisionsequencenumber", Value: -1}})

	assertion, err := findOne[model.AccountKeyAssertion](r.assertionCollections[ACCOUNTKEY], filter, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("no account key assertion found for account id: %s", accountID.String()))
			el.AddCustomError(cerr)
			return nil, cerr
		}
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to get latest account key assertion by account id: %s", accountID.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetSnapRevisionAssertionBySHA3_384(el *cerror.ErrorList, snap_sha3_384 string) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	filter := bson.M{"snapsha3_384": snap_sha3_384}

	assertion, err := findOne[model.SnapRevisionAssertion](r.assertionCollections[SNAPREVISION], filter, nil)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("no snap revision assertion found for SHA3_384: %s", snap_sha3_384))
			el.AddCustomError(cerr)
			return nil, cerr
		}
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve snap revision assertion from MongoDB for SHA3_384: %s", snap_sha3_384))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetSnapDeclarationAssertionBySnapID(el *cerror.ErrorList, snapID string) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	filter := bson.M{"snapid": snapID}

	assertion, err := findOne[model.SnapDeclarationAssertion](r.assertionCollections[SNAPDECLARATION], filter, nil)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("no snap declaration assertion found for snap id: %s", snapID))
			el.AddCustomError(cerr)
			return nil, cerr
		}
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve snap declaration assertion from MongoDB for snap id: %s", snapID))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetLatestSnapDeclarationAssertion(el *cerror.ErrorList, snapID string) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	filter := bson.M{"snapid": snapID}
	opts := options.FindOne().SetSort(bson.D{{Key: "revision", Value: -1}})

	assertion, err := findOne[model.SnapDeclarationAssertion](r.assertionCollections[SNAPDECLARATION], filter, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("no snap declaration assertion found for snap id: %s", snapID))
			el.AddCustomError(cerr)
			return nil, cerr
		}
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve latest snap declaration assertion from MongoDB for snap id: %s", snapID))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetAccountAssertionByAccountID(el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountAssertion, *cerror.CustomError) {
	filter := bson.M{"accountid": accountID}

	assertion, err := findOne[model.AccountAssertion](r.assertionCollections[ACCOUNT], filter, nil)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("no account assertion found for account id: %s", accountID.String()))
			el.AddCustomError(cerr)
			return nil, cerr
		}
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve account assertion from MongoDB for account id: %s", accountID.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) GetLatestAccountAssertionByAccountID(el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountAssertion, *cerror.CustomError) {
	filter := bson.M{"accountid": accountID}
	opts := options.FindOne().SetSort(bson.D{{Key: "revision", Value: -1}})

	assertion, err := findOne[model.AccountAssertion](r.assertionCollections[ACCOUNT], filter, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			cerr := cerror.NewCustomError(cerror.ResourceNotFound, fmt.Sprintf("no account assertion found for account id: %s", accountID.String()))
			el.AddCustomError(cerr)
			return nil, cerr
		}
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to retrieve latest account assertion from MongoDB for account id: %s", accountID.String()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

// ============= HELPERS =============
func findOne[T any](collection *mongo.Collection, filter interface{}, opts *options.FindOneOptions) (*T, error) {
	var result T
	err := collection.FindOne(context.Background(), filter, opts).Decode(&result)
	return &result, err
}
