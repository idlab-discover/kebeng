package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type IAssertionRepository interface {
	AddAssertion(snapEntryId uuid.UUID, assertionString string) (*model.Assertion, *cerror.CustomError)
	AddAccountKeyAssertion(el *cerror.ErrorList, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name string, revision uint32, account_id uuid.UUID, since time.Time, body_length uint64, signature string) (*model.AccountKeyAssertion, *cerror.CustomError)
	AddSnapRevisionAssertion(el *cerror.ErrorList, authority_id, snap_sha3_384, sign_key_SHA3_384 string, developer_id, snap_entry_id uuid.UUID, snap_revision_sequence_number uint32, snap_size uint64, timestamp time.Time, signature string) (*model.SnapRevisionAssertion, *cerror.CustomError)

	GetAccountKeyAssertionByAccountId(el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError)
	GetSnapRevisionAssertionBySnapEntryId(el *cerror.ErrorList, snap_entry_id uuid.UUID) (*model.SnapRevisionAssertion, *cerror.CustomError)
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

func (r *AssertionRepository) AddAccountKeyAssertion(el *cerror.ErrorList, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name string, revision uint32, account_id uuid.UUID, since time.Time, body_length uint64, signature string) (*model.AccountKeyAssertion, *cerror.CustomError) {
	query := `INSERT INTO account_key_assertion (authority_id, public_key_SHA3_384, sign_key_SHA3_384, name, revision, account_id, since, body_length, signature) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	assertion := &model.AccountKeyAssertion{}

	err := r.db.Get(assertion, query, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name, revision, account_id, since, body_length, signature)
	if err != nil {
		logrus.Errorf("failed to save account key assertion in database: %v", err)
		el.AddCustomError(cerror.ConvertError(err, fmt.Sprintf("failed to save account key assertion in database: %v", err)))
		return nil, cerror.ConvertError(err, fmt.Sprintf("failed to save account key assertion in database: %v", err))
	}

	return assertion, nil
}

func (r *AssertionRepository) AddSnapRevisionAssertion(el *cerror.ErrorList, authority_id, snap_sha3_384, sign_key_SHA3_384 string, developer_id, snap_entry_id uuid.UUID, snap_revision_sequence_number uint32, snap_size uint64, timestamp time.Time, signature string) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	query := `INSERT INTO snap_revision_assertion (authority_id, snap_sha3_384, sign_key_SHA3_384, developer_id, snap_entry_id, snap_revision_sequence_number, timestamp, snap_size, signature) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	assertion := &model.SnapRevisionAssertion{}
	err := r.db.Get(assertion, query, authority_id, snap_sha3_384, sign_key_SHA3_384, developer_id, snap_entry_id, snap_revision_sequence_number, timestamp, snap_size, signature)
	if err != nil {
		logrus.Errorf("failed to save snap revision assertion in database: %v", err)
		el.AddCustomError(cerror.ConvertError(err, fmt.Sprintf("failed to save snap revision assertion in database: %v", err)))
		return nil, cerror.ConvertError(err, fmt.Sprintf("failed to save snap revision assertion in database: %v", err))
	}

	return assertion, nil
}

func (r *AssertionRepository) GetAccountKeyAssertionByAccountId(el *cerror.ErrorList, account_id uuid.UUID) (*model.AccountKeyAssertion, *cerror.CustomError) {
	query := `SELECT id, authority_id, public_key_SHA3_384, sign_key_SHA3_384, name, revision, account_id, since, body_length, signature FROM account_key_assertion WHERE account_id = $1`
	assertion := &model.AccountKeyAssertion{}

	err := r.db.Get(assertion, query, account_id)
	if err != nil {
		logrus.Errorf("failed to get account key assertion by id: %v", err)
		el.AddCustomError(cerror.ConvertError(err, fmt.Sprintf("failed to get account key assertion by id: %v", err)))
		return nil, cerror.ConvertError(err, fmt.Sprintf("failed to get account key assertion by id: %v", err))
	}

	return assertion, nil
}

func (r *AssertionRepository) GetSnapRevisionAssertionBySnapEntryId(el *cerror.ErrorList, snap_entry_id uuid.UUID) (*model.SnapRevisionAssertion, *cerror.CustomError) {
	query := `SELECT id, authority_id, snap_sha3_384, developer_id, snap_entry_id, snap_revision_sequence_number, timestamp, signature FROM snap_revision_assertion WHERE snap_entry_id = $1`
	assertion := &model.SnapRevisionAssertion{}
	err := r.db.Get(assertion, query, snap_entry_id)
	if err != nil {
		logrus.Errorf("failed to get snap revision assertion by id: %v", err)
		el.AddCustomError(cerror.ConvertError(err, fmt.Sprintf("failed to get snap revision assertion by id: %v", err)))
		return nil, cerror.ConvertError(err, fmt.Sprintf("failed to get snap revision assertion by id: %v", err))
	}

	return assertion, nil
}
