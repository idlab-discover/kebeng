package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Assertion struct {
	*gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Assertion string    `json:"assertion"`
}
