package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	ALL      = "all"
	ENTRY    = "entry"
	TRACK    = "track"
	CHANNEL  = "channel"
	BRANCH   = "branch"
	REVISION = "revision"
	UPLOAD   = "upload"
	COMMENT  = "comment"
)

// SnapEntry represents the base information for a snap.
type SnapEntry struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
	Name           string          `json:"name" db:"name"`
	Revisions      []*SnapRevision `json:"revisions,omitempty"`
	Tracks         []*SnapTrack    `json:"tracks,omitempty"`
	Channels       []*SnapChannel  `json:"channels,omitempty"`
	Branches       []*SnapBranch   `json:"branches,omitempty"`
	Type           *string         `json:"type,omitempty" db:"type"`
	Confinement    *string         `json:"confinement,omitempty" db:"confinement"`
	Base           *string         `json:"base,omitempty" db:"base"`
	Private        *bool           `json:"private,omitempty" db:"private"`
	AccountID      uuid.UUID       `json:"account_id" db:"account_id"`
	Status         *string         `json:"status,omitempty" db:"status"`
	Price          *float64        `json:"price,omitempty" db:"price"`
	Store          *string         `json:"store,omitempty" db:"store"`
	IconURL        *string         `json:"icon_url,omitempty" db:"icon_url"`
	LatestComments []*SnapComment  `json:"latest_comments,omitempty"`
}

// SnapTrack represents a track (e.g., versions like 2.0, 2.1, 2.2).
type SnapTrack struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time     `json:"deleted_at,omitempty" db:"deleted_at"`
	Name        string         `json:"name" db:"name"`
	SnapEntryID uuid.UUID      `json:"snap_entry_id" db:"entry_id"`
	Channels    []*SnapChannel `json:"channels,omitempty"`
}

// SnapChannel represents a channel (e.g., stable, beta, edge, candidate).
type SnapChannel struct {
	ID          uuid.UUID     `json:"id" db:"id"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time    `json:"deleted_at,omitempty" db:"deleted_at"`
	Name        string        `json:"name" db:"name"`
	SnapTrackID uuid.UUID     `json:"snap_track_id" db:"snap_track_id"`
	SnapEntryID uuid.UUID     `json:"snap_entry_id" db:"entry_id"`
	Branches    []*SnapBranch `json:"branches,omitempty"`
}

// SnapBranch represents a branch (for example, "beta") of a snap.
type SnapBranch struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	SnapRiskID  uuid.UUID `json:"snap_risk_id" db:"snap_risk_id"`
	SnapEntryID uuid.UUID `json:"snap_entry_id" db:"entry_id"`
	RevisionID  uuid.UUID `json:"revision_id" db:"revision_id"`
}

// SnapRevision represents a specific version (revision) of a snap.
type SnapRevision struct {
	ID                     uuid.UUID      `json:"id" db:"id"`
	CreatedAt              time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at" db:"updated_at"`
	DeletedAt              *time.Time     `json:"deleted_at,omitempty" db:"deleted_at"`
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

// SnapComment represents a comment on a snap.
type SnapComment struct {
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

type SnapUpload struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	EntryID          uuid.UUID  `json:"entry_id" db:"entry_id"`
	AccountID        uuid.UUID  `json:"account_id" db:"account_id"`
	SnapName         string     `json:"snap_name" db:"snap_name"`
	Status           string     `json:"status" db:"status"`
	StatusDetailsURL string     `json:"status_details_url" db:"status_details_url"`
}
