package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ############# ACCOUNT SERVICE #############
type TestKey struct {
	ID               uuid.UUID    `json:"id" db:"id"`
	Name             string       `json:"name" db:"name"`
	SHA3384          string       `json:"sha3384" db:"sha3384"` // Should be unique
	EncodedPublicKey string       `json:"encoded_public_key" db:"encoded_public_key"`
	AccountID        uuid.UUID    `json:"account_id" db:"account_id"`
	Account          *TestAccount `json:"-" db:"-"`
	Until            *time.Time   `json:"until" db:"until"`
	CreatedAt        *time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt        *time.Time   `json:"updated_at" db:"updated_at"`
	DeletedAt        *time.Time   `json:"deleted_at" db:"deleted_at"`
}

type TestSSHKey struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	PublicKeyString string     `json:"public_key_string" db:"public_key_string"` // should be unique
	AccountID       uuid.UUID  `json:"account_id" db:"account_id"`
	CreatedAt       *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at" db:"deleted_at"`
}

type TestAccount struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	DisplayName  string       `json:"display_name" db:"display_name"`
	Username     string       `json:"username" db:"username"`
	Email        string       `json:"email" db:"email"`
	PasswordHash string       `json:"password_hash" db:"password_hash"`
	CreatedAt    *time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt    *time.Time   `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time   `json:"deleted_at" db:"deleted_at"`
	Validation   *string      `json:"validation" db:"validation"`
	SSHKeys      []TestSSHKey // associations (handled separately)
}

// ############# ASSERTION SERVICE #############

type TestAccountKeyAssertion struct {
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
	Until                      time.Time  `json:"until" db:"until"`
	BodyLength                 uint64     `json:"body_length" db:"body_length"`
	Body                       []byte     `json:"body" db:"body"` // TODO: figure out what (idk what this is) "base64 encoded version prefixed public key packet" means
	SignKeySHA3_384            string     `json:"sign_key_sha3_384" db:"sign_key_sha3_384"`
	Signature                  string     `json:"signature" db:"signature"`
}

type TestSnapRevisionAssertion struct {
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

// ############# STORE SERVICE #############

type TestSnapEntry struct {
	ID             uuid.UUID           `json:"id" db:"id"`
	CreatedAt      time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time          `json:"deleted_at,omitempty" db:"deleted_at"`
	Name           string              `json:"name" db:"name"`
	Revisions      []*TestSnapRevision `json:"revisions,omitempty"`
	Tracks         []*TestSnapTrack    `json:"tracks,omitempty"`
	Channels       []*TestSnapChannel  `json:"channels,omitempty"`
	Branches       []*TestSnapBranch   `json:"branches,omitempty"`
	Type           *string             `json:"type,omitempty" db:"type"`
	Confinement    *string             `json:"confinement,omitempty" db:"confinement"`
	Base           *string             `json:"base,omitempty" db:"base"`
	Private        *bool               `json:"private,omitempty" db:"private"`
	AccountID      uuid.UUID           `json:"account_id" db:"account_id"`
	Status         *string             `json:"status,omitempty" db:"status"`
	Price          *float64            `json:"price,omitempty" db:"price"`
	Store          *string             `json:"store,omitempty" db:"store"`
	IconURL        *string             `json:"icon_url,omitempty" db:"icon_url"`
	LatestComments []*TestSnapComment  `json:"latest_comments,omitempty"`
}

// TestSnapTrack represents a track (e.g., versions like 2.0, 2.1, 2.2).
type TestSnapTrack struct {
	ID          uuid.UUID          `json:"id" db:"id"`
	CreatedAt   time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time         `json:"deleted_at,omitempty" db:"deleted_at"`
	Name        string             `json:"name" db:"name"`
	SnapEntryID uuid.UUID          `json:"snap_entry_id" db:"entry_id"`
	Channels    []*TestSnapChannel `json:"channels,omitempty"`
}

// TestSnapChannel represents a channel (e.g., stable, beta, edge, candidate).
type TestSnapChannel struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time        `json:"deleted_at,omitempty" db:"deleted_at"`
	Name        string            `json:"name" db:"name"`
	SnapTrackID uuid.UUID         `json:"snap_track_id" db:"snap_track_id"`
	SnapEntryID uuid.UUID         `json:"snap_entry_id" db:"entry_id"`
	Branches    []*TestSnapBranch `json:"branches,omitempty"`
}

// TestSnapBranch represents a branch (for example, "beta") of a snap.
type TestSnapBranch struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	SnapRiskID  uuid.UUID `json:"snap_risk_id" db:"snap_risk_id"`
	SnapEntryID uuid.UUID `json:"snap_entry_id" db:"entry_id"`
	RevisionID  uuid.UUID `json:"revision_id" db:"revision_id"`
}

// TestSnapRevision represents a specific version (revision) of a snap.
type TestSnapRevision struct {
	ID                     uuid.UUID      `json:"id" db:"id"`
	CreatedAt              time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at" db:"updated_at"`
	DeletedAt              *time.Time     `json:"deleted_at,omitempty" db:"deleted_at"`
	SnapName               *string        `json:"snap_name" db:"snap_name"`
	BuildAssertionFileName *string        `json:"build_assertion_filename,omitempty" db:"build_assertion_filename"`
	SHA3_384               *string        `json:"sha3_384,omitempty" db:"sha3_384"`
	SHA3_384_Encoded       *string        `json:"sha3_384_encoded,omitempty" db:"sha3_384_encoded"`
	Size                   *uint64        `json:"size,omitempty" db:"size"`
	SequenceNumber         *uint          `json:"sequence_number,omitempty" db:"sequence_number"`
	Architectures          pq.StringArray `json:"architectures,omitempty" db:"architectures"`
	Status                 *string        `json:"status,omitempty" db:"status"`
	Version                *string        `json:"version,omitempty" db:"version"`
	SnapEntryID            uuid.UUID      `json:"snap_entry_id" db:"entry_id"`
	SnapTrackID            uuid.UUID      `json:"snap_track_id" db:"snap_track_id"`
	SnapChannelID          uuid.UUID      `json:"snap_channel_id" db:"snap_channel_id"`
	SnapBranchID           uuid.UUID      `json:"snap_branch_id" db:"snap_branch_id"`
}

// TestSnapComment represents a comment on a snap.
type TestSnapComment struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	AuthorID    uuid.UUID  `json:"author_id" db:"author_id"`
	Since       time.Time  `json:"since" db:"since"`
	Reason      string     `json:"reason" db:"reason"`
	Comment     string     `json:"comment" db:"comment"`
	SnapEntryID uuid.UUID  `json:"snap_entry_id" db:"entry_id"`
}

type TestSnapUpload struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	EntryID           uuid.UUID  `json:"entry_id" db:"entry_id"`
	AccountID         uuid.UUID  `json:"account_id" db:"account_id"`
	UnscannedFileName string     `json:"unscanned_file_name" db:"unscanned_file_name"`
	SnapName          string     `json:"snap_name" db:"snap_name"`
	Status            string     `json:"status" db:"status"`
	StatusDetailsURL  string     `json:"status_details_url" db:"status_details_url"`
}
