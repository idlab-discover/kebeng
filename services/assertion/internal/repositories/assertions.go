package repositories

import (
	"gorm.io/gorm"
)

type IAssertionRepository interface {
}

type AssertionRepository struct {
	db *gorm.DB
}

func NewAssertionRepository(db *gorm.DB) *AssertionRepository {
	return &AssertionRepository{db: db}
}
