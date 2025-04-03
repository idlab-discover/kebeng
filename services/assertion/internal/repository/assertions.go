package repository

import (
	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/assertion/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type IAssertionRepository interface {
	AddAssertion(snapEntryId uuid.UUID, assertionString string) (*models.Assertion, *cerror.CustomError)
}

type AssertionRepository struct {
	db *sqlx.DB
}

func NewAssertionRepository(db *sqlx.DB) IAssertionRepository {
	return &AssertionRepository{db: db}
}

func (r *AssertionRepository) AddAssertion(snapEntryId uuid.UUID, assertionString string) (*models.Assertion, *cerror.CustomError) {
	query := `INSERT INTO assertions (snap_entry_id, assertion) VALUES ($1, $2) RETURNING id, assertion`
	assertion := &models.Assertion{}

	err := r.db.Get(assertion, query, snapEntryId, assertionString)
	if err != nil {
		logrus.Errorf("Failed to save assertion in database: %v", err)
		return nil, cerror.NewCustomError(cerror.DatabaseError, "Failed to save assertion in database")
	}

	return assertion, nil
}
