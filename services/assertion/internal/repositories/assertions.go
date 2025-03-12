package repositories

import (
	"github.com/idlab-discover/kebeng/services/assertion/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type IAssertionRepository interface {
	AddAssertion(assertionString string) (*models.Assertion, error)
}

type AssertionRepository struct {
	db *gorm.DB
}

func NewAssertionRepository(db *gorm.DB) *AssertionRepository {
	return &AssertionRepository{db: db}
}

func (r *AssertionRepository) AddAssertion(assertionString string) (*models.Assertion, error) {
	assertion := models.Assertion{
		Assertion: assertionString,
	}

	db := r.db.Save(&assertion)
	if db.Error != nil {
		logrus.Errorf("Failed to save assertion in datase: %v", db.Error)
		return nil, db.Error
	}
	if db.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return &assertion, nil
}
