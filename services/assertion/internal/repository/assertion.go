package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type IAssertionRepository interface {
	AddAssertion(snapEntryId uuid.UUID, assertionString string) (*model.Assertion, *cerror.CustomError)
	AddAccountKeyAssertion(el *cerror.ErrorList, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name string, revision uint32, account_id uuid.UUID, since time.Time, until time.Time, body []byte, body_length uint64, signature string) (*model.AccountKeyAssertion, *cerror.CustomError)
	AddSnapRevisionAssertion(el *cerror.ErrorList, authority_id, snap_sha3_384, sign_key_SHA3_384 string, developer_id, snap_entry_id uuid.UUID, snap_revision_sequence_number uint32, snap_size uint64, timestamp time.Time, signature string) (*model.SnapRevisionAssertion, *cerror.CustomError)
	AddSnapDeclarationAssertion(el *cerror.ErrorList, authorityID, signKey, snapID, snapName, publisherID string, revision, series uint32, timestamp time.Time, refreshControl []string, aliases []model.Alias, plugs map[string]*model.Plug, slots map[string]*model.Slot, signature string) (*model.SnapDeclarationAssertion, *cerror.CustomError)

	GetAccountKeyAssertionByName(el *cerror.ErrorList, name string) (*model.AccountKeyAssertion, *cerror.CustomError)
	GetLatestAccountKeyAssertion(el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError)
	GetSnapRevisionAssertionBySHA3_384(el *cerror.ErrorList, snap_sha3_384 string) (*model.SnapRevisionAssertion, *cerror.CustomError)
	GetSnapDeclarationAssertionBySnapID(el *cerror.ErrorList, snapID string) (*model.SnapDeclarationAssertion, *cerror.CustomError)
	GetLatestSnapDeclarationAssertion(el *cerror.ErrorList, snapID string) (*model.SnapDeclarationAssertion, *cerror.CustomError)
}

type AssertionRepository struct {
	db *sqlx.DB
}

func NewAssertionRepository(db *sqlx.DB) IAssertionRepository {
	return &AssertionRepository{db: db}
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

func (r *AssertionRepository) AddAccountKeyAssertion(el *cerror.ErrorList, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name string, revision uint32, account_id uuid.UUID, since time.Time, until time.Time, body []byte, body_length uint64, signature string) (*model.AccountKeyAssertion, *cerror.CustomError) {
	query := `
		INSERT INTO account_key_assertion (authority_id, public_key_SHA3_384, sign_key_SHA3_384, name, revision, account_id, since, until, body, body_length, signature) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) 
		RETURNING id`
	assertion := &model.AccountKeyAssertion{}

	err := r.db.Get(assertion, query, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name, revision, account_id, since, until, body, body_length, signature)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to save account key assertion in database: %v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

func (r *AssertionRepository) AddSnapRevisionAssertion(el *cerror.ErrorList, authority_id, snap_sha3_384, sign_key_SHA3_384 string, developer_id, snap_entry_id uuid.UUID, snap_revision_sequence_number uint32, snap_size uint64, timestamp time.Time, signature string) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	query := `
		INSERT INTO snap_revision_assertion (authority_id, snap_sha3_384, sign_key_SHA3_384, developer_id, snap_entry_id, snap_revision_sequence_number, timestamp, snap_size, signature) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		RETURNING id
	`
	assertion := &model.SnapRevisionAssertion{}
	err := r.db.Get(assertion, query, authority_id, snap_sha3_384, sign_key_SHA3_384, developer_id, snap_entry_id, snap_revision_sequence_number, timestamp, snap_size, signature)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to save snap revision assertion in database: %v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
}

// TODO: implement this correctly and decide how to store in db
func (r *AssertionRepository) AddSnapDeclarationAssertion(el *cerror.ErrorList, authorityID, signKey, snapID, snapName, publisherID string, revision, series uint32, timestamp time.Time, refreshControl []string, aliases []model.Alias, plugs map[string]*model.Plug, slots map[string]*model.Slot, signature string) (*model.SnapDeclarationAssertion, *cerror.CustomError) {
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

func (r *AssertionRepository) GetAccountKeyAssertionByName(el *cerror.ErrorList, name string) (*model.AccountKeyAssertion, *cerror.CustomError) {
	query := `
		SELECT id, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name, revision, account_id, since, until, body_length, signature 
		FROM account_key_assertion 
		WHERE name = $1
	`
	assertion := &model.AccountKeyAssertion{}

	err := r.db.Get(assertion, query, name)
	if err != nil {
		logrus.Errorf("failed to get account key assertion by name:%s, err: %v", name, err)
		el.AddCustomError(cerror.ConvertError(err, fmt.Sprintf("failed to get account key assertion by name:%s, err: %v", name, err)))
		return nil, cerror.ConvertError(err, fmt.Sprintf("failed to get account key assertion by name: %s, err:  %v", name, err))
	}

	return assertion, nil
}

func (r *AssertionRepository) GetLatestAccountKeyAssertion(el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError) {
	query := `
		SELECT id, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name, revision, account_id, since, until, body_length, signature 
		FROM account_key_assertion 
		WHERE account_id = $1 ORDER BY revision DESC LIMIT 1
	`
	assertion := &model.AccountKeyAssertion{}

	err := r.db.Get(assertion, query, account_id)
	if err != nil {
		logrus.Errorf("failed to get latest account key assertion by account id: %s, err: %v", account_id.String(), err)
		el.AddCustomError(cerror.ConvertError(err, fmt.Sprintf("failed to get latest account key assertion by account id: %s, err: %v", account_id.String(), err)))
		return nil, cerror.ConvertError(err, fmt.Sprintf("failed to get latest account key assertion by account id: %s, err: %v", account_id.String(), err))
	}

	return assertion, nil
}

func (r *AssertionRepository) GetSnapRevisionAssertionBySHA3_384(el *cerror.ErrorList, snap_sha3_384 string) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	query := `
		SELECT id, authority_id, snap_sha3_384, developer_id, snap_entry_id, snap_revision_sequence_number, timestamp, signature 
		FROM snap_revision_assertion 
		WHERE snap_sha3_384 = $1
	`
	assertion := &model.SnapRevisionAssertion{}
	err := r.db.Get(assertion, query, snap_sha3_384)
	if err != nil {
		cerr := cerror.ConvertError(err, fmt.Sprintf("failed to get snap revision assertion by SHA3_384: %s, err: %v", snap_sha3_384, err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return nil, cerr
	}

	return assertion, nil
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
