package model

import (
	"time"

	"github.com/google/uuid"
)

type Assertion struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	Assertion string     `json:"assertion"`
}

type AccountKeyAssertion struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	Type              string     `json:"type" db:"type"`
	AuthorityID       string     `json:"authority_id" db:"authority_id"`
	Revision          uint32     `json:"revision" db:"revision"`
	PublicKeySHA3_384 string     `json:"public_key_sha3_384" db:"public_key_sha3_384"`
	AccountID         uuid.UUID  `json:"account_id" db:"account_id"`
	Name              string     `json:"name" db:"name"`
	Since             time.Time  `json:"since" db:"since"`
	BodyLength        uint64     `json:"body_length" db:"body_length"`
	SignKeySHA3_384   string     `json:"sign_key_sha3_384" db:"sign_key_sha3_384"`
}
