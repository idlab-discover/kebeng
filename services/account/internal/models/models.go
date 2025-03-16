package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Key struct {
	gorm.Model
	ID               uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name             string
	SHA3384          string `gorm:"unique"`
	EncodedPublicKey string
	AccountID        uuid.UUID
	Account          Account
	Until            time.Time
}

type SSHKey struct {
	gorm.Model
	ID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PublicKeyString string    `gorm:"unique"`
	AccountID       uuid.UUID
	Account         Account
}

type Account struct {
	ID           uuid.UUID  `db:"id"`
	DisplayName  string     `db:"display_name"`
	Username     string     `db:"username"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
	Validation   string     `db:"validation"`
	SSHKeys      []SSHKey   // associations (handled separately)
}
