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
	ID                         uuid.UUID  `json:"id" db:"id"`
	CreatedAt                  time.Time  `json:"created_at" db:"created_at"`
	DeletedAt                  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	Type                       string     `json:"type" db:"type"`
	AuthorityID                string     `json:"authority_id" db:"authority_id"`
	SnapRevisionSequenceNumber uint32     `json:"revision" db:"revision"`
	PublicKeySHA3_384          string     `json:"public_key_sha3_384" db:"public_key_sha3_384"`
	AccountID                  uuid.UUID  `json:"account_id" db:"account_id"`
	Name                       string     `json:"name" db:"name"`
	Since                      time.Time  `json:"since" db:"since"`
	BodyLength                 uint64     `json:"body_length" db:"body_length"`
	Body                       []byte     `json:"body" db:"body"` // TODO: figure out what (idk what this is) "base64 encoded version prefixed public key packet" means
	SignKeySHA3_384            string     `json:"sign_key_sha3_384" db:"sign_key_sha3_384"`
	Signature                  string     `json:"signature" db:"signature"`
}

type SnapRevisionAssertion struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	Type                       string    `json:"type" db:"type"`
	AuthorityID                string    `json:"authority_id" db:"authority_id"`
	SnapSHA3_384               string    `json:"snap_sha3_384" db:"snap_sha3_384"`
	DeveloperID                uuid.UUID `json:"developer_id" db:"developer_id"`
	SnapEntryID                uuid.UUID `json:"snap_entry_id" db:"snap_entry_id"`
	SnapRevisionSequenceNumber uint32    `json:"snap_revision_sequence_number" db:"snap_revision_sequence_number"`
	SnapSize                   uint64    `json:"snap_size" db:"snap_size"`
	Timestamp                  time.Time `json:"timestamp" db:"timestamp"`
	SignKeySHA3_384            string    `json:"sign_key_sha3_384" db:"sign_key_sha3_384"`
	Signature                  string    `json:"signature" db:"signature"`
}
