package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ALL     = "all"
	KEY     = "key"
	SSHKEY  = "sshkey"
	ACCOUNT = "account"
)

type Key struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	Name             string     `json:"name" db:"name"`
	SHA3384          string     `json:"sha3384" db:"sha3384"` // Should be unique
	EncodedPublicKey string     `json:"encoded_public_key" db:"encoded_public_key"`
	AccountID        uuid.UUID  `json:"account_id" db:"account_id"`
	Account          *Account   `json:"-" db:"-"`
	Until            *time.Time `json:"until" db:"until"`
	CreatedAt        *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at" db:"deleted_at"`
}

type SSHKey struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	PublicKeyString string     `json:"public_key_string" db:"public_key_string"` // should be unique
	AccountID       uuid.UUID  `json:"account_id" db:"account_id"`
	CreatedAt       *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at" db:"deleted_at"`
}

type Account struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	DisplayName  string     `json:"display_name" db:"display_name"`
	Username     string     `json:"username" db:"username"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"password_hash" db:"password_hash"`
	CreatedAt    *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" db:"deleted_at"`
	Validation   *string    `json:"validation" db:"validation"`
	SSHKeys      []SSHKey   // associations (handled separately)
}
