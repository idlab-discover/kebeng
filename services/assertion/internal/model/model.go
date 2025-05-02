package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Assertion struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	Assertion string     `json:"assertion"`
}

type AccountKeyAssertion struct {
	ID                       uuid.UUID  `json:"id" db:"id"`
	CreatedAt                time.Time  `json:"created_at" db:"created_at"`
	DeletedAt                *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	Type                     string     `json:"type" db:"type"`
	AuthorityID              string     `json:"authority_id" db:"authority_id"`
	RevisionSequenceNumber   uint32     `json:"revision" db:"revision"`
	PublicKeySha3_384Encoded string     `json:"public_key_sha3_384" db:"public_key_sha3_384"`
	AccountID                uuid.UUID  `json:"account_id" db:"account_id"`
	Name                     string     `json:"name" db:"name"`
	Since                    time.Time  `json:"since" db:"since"`
	Until                    time.Time  `json:"until" db:"until"`
	BodyLength               uint64     `json:"body_length" db:"body_length"`
	Body                     []byte     `json:"body" db:"body"` // TODO: figure out what (idk what this is) "base64 encoded version prefixed public key packet" means
	SignKeySHA3_384          string     `json:"sign_key_sha3_384" db:"sign_key_sha3_384"`
	Signature                string     `json:"signature" db:"signature"`
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

type SnapDeclarationAssertion struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	DeletedAt       *time.Time     `json:"deleted_at,omitempty" db:"deleted_at"`
	Type            string         `json:"type" db:"type"`
	AuthorityID     string         `json:"authority_id" db:"authority_id"`
	Revision        uint32         `json:"revision" db:"revision"`
	Series          string         `json:"series" db:"series"`
	SnapID          string         `json:"snap_id" db:"snap_id"`
	SnapName        string         `json:"snap_name" db:"snap_name"`
	PublisherID     string         `json:"publisher_id" db:"publisher_id"`
	Timestamp       time.Time      `json:"timestamp" db:"timestamp"`
	RefreshControl  pq.StringArray `json:"refresh_control" db:"refresh_control"`
	Aliases         []Alias        `json:"aliases" db:"aliases"`
	Plugs           PlugMap        `json:"plugs" db:"plugs"`
	Slots           SlotMap        `json:"slots" db:"slots"`
	SignKeySHA3_384 string         `json:"sign_key_sha3_384" db:"sign_key_sha3_384"`
	Signature       string         `json:"signature" db:"signature"`
}

type Alias struct {
	Name   string `json:"name" db:"name"`
	Target string `json:"target" db:"target"`
}

// Same as models in client
// NOTE: maybe change this to use an enum? with states like Allowed, Denied and NotSet and can be expanded later on to support more complex objects
// bools are pointers so that we can use nil to indicate that the field is not set
type Plug struct {
	AllowInstallation   *bool
	DenyInstallation    *bool
	AllowConnection     *bool
	DenyConnection      *bool
	AllowAutoConnection *bool
	DenyAutoConnection  *bool
}

type PlugMap map[string]*Plug

type Slot struct {
	AllowInstallation   *bool
	DenyInstallation    *bool
	AllowConnection     *bool
	DenyConnection      *bool
	AllowAutoConnection *bool
	DenyAutoConnection  *bool
}

type SlotMap map[string]*Slot

type AccountAssertion struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	Type            string     `json:"type" db:"type"`
	AuthorityID     string     `json:"authority_id" db:"authority_id"`
	Revision        uint32     `json:"revision" db:"revision"`
	AccountID       uuid.UUID  `json:"account_id" db:"account_id"`
	DisplayName     string     `json:"display_name" db:"display_name"`
	Username        string     `json:"username" db:"username"`
	Validation      string     `json:"validation" db:"validation"`
	Timestamp       time.Time  `json:"timestamp" db:"timestamp"`
	SignKeySHA3_384 string     `json:"sign_key_sha3_384" db:"sign_key_sha3_384"`
	Signature       string     `json:"signature" db:"signature"`
}
