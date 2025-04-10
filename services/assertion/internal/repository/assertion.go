package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type IAssertionRepository interface {
	AddAssertion(snapEntryId uuid.UUID, assertionString string) (*model.Assertion, *cerror.CustomError)
	AddAccountKeyAssertion(authority_id, public_key_SHA3_384, sign_key_SHA3_384, name string, revision uint32, account_id uuid.UUID, since time.Time, body_length uint64) (*model.AccountKeyAssertion, *cerror.CustomError)
}

type AssertionRepository struct {
	db *sqlx.DB
}

func NewAssertionRepository(db *sqlx.DB) IAssertionRepository {
	return &AssertionRepository{db: db}
}

func (r *AssertionRepository) AddAssertion(snapEntryId uuid.UUID, assertionString string) (*model.Assertion, *cerror.CustomError) {
	query := `INSERT INTO assertions (snap_entry_id, assertion) VALUES ($1, $2) RETURNING id, assertion`
	assertion := &model.Assertion{}

	err := r.db.Get(assertion, query, snapEntryId, assertionString)
	if err != nil {
		logrus.Errorf("Failed to save assertion in database: %v", err)
		return nil, cerror.NewCustomError(cerror.DatabaseError, "Failed to save assertion in database")
	}

	return assertion, nil
}

// TODO: implement
func (r *AssertionRepository) AddAccountKeyAssertion(authority_id, public_key_SHA3_384, sign_key_SHA3_384, name string, revision uint32, account_id uuid.UUID, since time.Time, body_length uint64) (*model.AccountKeyAssertion, *cerror.CustomError) {
	return nil, nil
}
