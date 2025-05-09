package repository

import (
	"context"
	"encoding/json"
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
	"github.com/lib/pq"
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
	tx, err := r.db.Beginx()
	if err != nil {
		cerr := cerror.ConvertError(err, "failed to begin transaction")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// marshal plugs/slots into JSON
	plugsJSON, err := json.Marshal(plugs)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to marshal plugs: %v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}
	slotsJSON, err := json.Marshal(slots)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to marshal slots: %v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	parent := &model.SnapDeclarationAssertion{}
	const parentInsertQuery = `
    INSERT INTO snap_declaration_assertion
      (authority_id, sign_key_sha3_384,
       snap_id,    snap_name,      publisher_id,
       revision,   series,         timestamp,
       refresh_control, plugs,      slots,
       signature)
    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
    RETURNING
      id, authority_id, sign_key_sha3_384,
      snap_id, snap_name, publisher_id,
      revision, series, timestamp,
      refresh_control, plugs, slots,
      signature, created_at
  `
	err = tx.Get(parent, parentInsertQuery,
		authorityID, signKey,
		snapID, snapName, publisherID,
		revision, series, timestamp,
		pq.Array(refreshControl),
		plugsJSON, slotsJSON,
		signature,
	)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to insert snap_declaration_assertion: %v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	// insert aliases
	for _, a := range aliases {
		if _, err = tx.Exec(`
      INSERT INTO alias (assertion_id, name, target)
      VALUES ($1,$2,$3)
    `, parent.ID, a.Name, a.Target); err != nil {
			cerr := cerror.ConvertError(err, fmt.Sprintf("failed to insert alias %q: %v", a.Name, err))
			logrus.Error(cerr)
			el.AddCustomError(cerr)
			return nil, cerr
		}
	}

	// commit
	if err = tx.Commit(); err != nil {
		cerr := cerror.ConvertError(err, "failed to commit transaction")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	parent.Aliases = aliases
	parent.Plugs = plugs
	parent.Slots = slots

	return parent, nil
}

func (r *AssertionRepository) AddSnapBuildAssertion(el *cerror.ErrorList, authority_id, sign_key_SHA3_384 string, snap_id, account_id uuid.UUID, grade string, snap_sha3_384 string, snap_size uint64, signature string, timestamp time.Time) (*model.SnapBuildAssertion, *cerror.CustomError) {
	query := `
		INSERT INTO snap_build_assertion (authority_id, sign_key_SHA3_384, snap_id, account_id, grade, snap_sha3_384, snap_size, signature, timestamp) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		RETURNING id
	`
	assertion := &model.SnapBuildAssertion{}
	err := r.db.Get(assertion, query, authority_id, sign_key_SHA3_384, snap_id, account_id, grade, snap_sha3_384, snap_size, signature, timestamp)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to save snap build assertion in database: %v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddAccountAssertion(el *cerror.ErrorList, authority_id, displayName, username, validation string, accountID uuid.UUID, revision uint32, timestamp time.Time, sign_key_SHA3_384, signature string) (*model.AccountAssertion, *cerror.CustomError) {
	query := `
		INSERT INTO account_assertion (authority_id, display_name, username, validation, account_id, revision, timestamp, sign_key_SHA3_384, signature)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	  RETURNING id, authority_id, display_name, username, validation, account_id,
		revision, timestamp, sign_key_sha3_384, signature
	`
	assertion := &model.AccountAssertion{}
	err := r.db.Get(assertion, query, authority_id, displayName, username, validation, accountID, revision, timestamp, sign_key_SHA3_384, signature)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to save account assertion in database: %v", err))
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

	var result model.AccountKeyAssertion
	err := r.assertionCollections[ACCOUNTKEY].FindOne(context.Background(), filter).Decode(&result)
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

	return &result, nil
}

func (r *AssertionRepository) GetLatestAccountKeyAssertion(el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError) {
	filter := bson.M{"accountid": accountID}
	opts := options.FindOne().SetSort(bson.D{{Key: "revisionsequencenumber", Value: -1}})

	var result model.AccountKeyAssertion
	err := r.assertionCollections[ACCOUNTKEY].FindOne(context.Background(), filter, opts).Decode(&result)
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

	return &result, nil
}

func (r *AssertionRepository) GetSnapRevisionAssertionBySHA3_384(el *cerror.ErrorList, snap_sha3_384 string) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	filter := bson.M{"snapsha3_384": snap_sha3_384}

	var result model.SnapRevisionAssertion
	err := r.assertionCollections[SNAPREVISION].FindOne(context.Background(), filter).Decode(&result)
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

	return &result, nil
}

func (r *AssertionRepository) GetSnapDeclarationAssertionBySnapID(el *cerror.ErrorList, assertionID string) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	const parentQuery = `
        SELECT
            id, authority_id, sign_key_sha3_384, snap_id, snap_name, publisher_id, revision, series, timestamp,
            refresh_control, plugs, slots, signature, created_at
        FROM snap_declaration_assertion
        WHERE snap_id = $1
    `

	var assertion model.SnapDeclarationAssertion
	if err := r.db.Get(&assertion, parentQuery, assertionID); err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to load snap_declaration_assertion %q: %v", assertionID, err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	// now load aliases
	const aliasQuery = `
        SELECT name, target
          FROM alias
         WHERE assertion_id = $1
         ORDER BY name
    `
	var aliases []model.Alias
	if err := r.db.Select(&aliases, aliasQuery, assertion.ID); err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to load aliases for assertion %q: %v", assertionID, err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}
	assertion.Aliases = aliases

	return &assertion, nil
}

func (r *AssertionRepository) GetLatestSnapDeclarationAssertion(el *cerror.ErrorList, snapID string) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
	query := `
		SELECT id, authority_id, sign_key_sha3_384, snap_id, snap_name, publisher_id, revision, series, timestamp, refresh_control, plugs, slots, signature 
		FROM snap_declaration_assertion 
		WHERE snap_id = $1 ORDER BY revision DESC LIMIT 1
	`
	assertion := &model.SnapDeclarationAssertion{}

	err := r.db.Get(assertion, query, snapID)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to get latest snap declaration assertion by snap id: %s, err: %v", snapID, err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerror.ConvertError(err, fmt.Sprintf("failed to get latest snap declaration assertion by snap id: %s, err: %v", snapID, err))
	}

	return assertion, nil
}

func (r *AssertionRepository) GetAccountAssertionByAccountID(el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountAssertion, *cerror.CustomError) {
	query := `
		SELECT id, authority_id, display_name, username, validation, account_id, revision, timestamp, sign_key_SHA3_384, signature 
		FROM account_assertion 
		WHERE account_id = $1
	`
	assertion := &model.AccountAssertion{}

	err := r.db.Get(assertion, query, accountID)
	if err != nil {
		logrus.Errorf("failed to get account assertion by account id: %s, err: %v", accountID.String(), err)
		el.AddCustomError(cerror.ConvertError(err, fmt.Sprintf("failed to get account assertion by account id: %s, err: %v", accountID.String(), err)))
		return nil, cerror.ConvertError(err, fmt.Sprintf("failed to get account assertion by account id: %s, err: %v", accountID.String(), err))
	}

	return assertion, nil
}

func (r *AssertionRepository) GetLatestAccountAssertionByAccountID(el *cerror.ErrorList, accountID uuid.UUID) (*model.AccountAssertion, *cerror.CustomError) {
	query := `
		SELECT id, authority_id, display_name, username, validation, account_id, revision, timestamp, sign_key_SHA3_384, signature 
		FROM account_assertion 
		WHERE account_id = $1 ORDER BY revision DESC LIMIT 1
	`
	assertion := &model.AccountAssertion{}

	err := r.db.Get(assertion, query, accountID)
	if err != nil {
		logrus.Errorf("failed to get latest account assertion by account id: %s, err: %v", accountID.String(), err)
		el.AddCustomError(cerror.ConvertError(err, fmt.Sprintf("failed to get latest account assertion by account id: %s, err: %v", accountID.String(), err)))
		return nil, cerror.ConvertError(err, fmt.Sprintf("failed to get latest account assertion by account id: %s, err: %v", accountID.String(), err))
	}

	return assertion, nil
}
