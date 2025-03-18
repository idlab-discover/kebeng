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
	ID               uuid.UUID  `db:"id"`
	Name             string     `db:"name"`
	SHA3384          string     `db:"sha3384"` // Should be unique
	EncodedPublicKey string     `db:"encoded_public_key"`
	AccountID        uuid.UUID  `db:"account_id"`
	Account          *Account   `db:"-"`
	Until            time.Time  `db:"until"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
	DeletedAt        *time.Time `db:"deleted_at"`
}

type SSHKey struct {
	ID              uuid.UUID  `db:"id"`
	PublicKeyString string     `db:"public_key_string"` // should be unique
	AccountID       uuid.UUID  `db:"account_id"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	DeletedAt       *time.Time `db:"deleted_at"`
}

type Account struct {
	ID           uuid.UUID  `db:"id"`
	DisplayName  string     `db:"display_name"`
	Username     string     `db:"username"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	CreatedAt    *time.Time `db:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
	Validation   *string    `db:"validation"`
	SSHKeys      []SSHKey   // associations (handled separately)
}
