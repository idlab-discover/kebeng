package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/lib/pq"
	"github.com/minio/minio-go/v7"
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
	Type           string          `json:"type,omitempty" db:"type"`
	Version        string          `yaml:"version" db:"version"`
	Summary        string          `yaml:"summary" db:"summary"`
	Description    string          `yaml:"description" db:"description"`
	Confinement    string          `json:"confinement,omitempty" db:"confinement"`
	Base           string          `json:"base,omitempty" db:"base"`
	Grade          string          `json:"grade,omitempty" db:"grade"`
	Architectures  pq.StringArray  `json:"architectures,omitempty" db:"architectures"`
	Private        bool            `json:"private,omitempty" db:"private"`
	Status         string          `json:"status,omitempty" db:"status"`
	Price          float64         `json:"price,omitempty" db:"price"`
	Store          string          `json:"store,omitempty" db:"store"`
	IconURL        string          `json:"icon_url,omitempty" db:"icon_url"`
	AccountID      uuid.UUID       `json:"account_id" db:"account_id"`
	Revisions      []*SnapRevision `json:"revisions,omitempty"`
	Tracks         []*SnapTrack    `json:"tracks,omitempty"`
	Channels       []*SnapChannel  `json:"channels,omitempty"`
	Branches       []*SnapBranch   `json:"branches,omitempty"`
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
	SnapName               string         `json:"snap_name" db:"snap_name"`
	BuildAssertionFileName string         `json:"build_assertion_filename,omitempty" db:"build_assertion_filename"`
	SHA3_384_Encoded       string         `json:"sha3_384_encoded,omitempty" db:"sha3_384_encoded"`
	Size                   uint64         `json:"size,omitempty" db:"size"`
	SequenceNumber         uint32         `json:"sequence_number,omitempty" db:"sequence_number"`
	Architectures          pq.StringArray `json:"architectures,omitempty" db:"architectures"`
	MinioFilePath          string         `json:"minio_file_path,omitempty" db:"minio_file_path"`
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
	ID                uuid.UUID         `json:"id" db:"id"`
	CreatedAt         time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt         *time.Time        `json:"deleted_at,omitempty" db:"deleted_at"`
	EntryID           uuid.UUID         `json:"entry_id" db:"entry_id"`
	AccountID         uuid.UUID         `json:"account_id" db:"account_id"`
	UnscannedFileName string            `json:"unscanned_file_name" db:"unscanned_file_name"`
	SnapName          string            `json:"snap_name" db:"snap_name"`
	Status            string            `json:"status" db:"status"`
	StatusDetailsURL  string            `json:"status_details_url" db:"status_details_url"`
	Revision          uint32            `json:"revision" db:"revision"`
	Errors            *cerror.ErrorList `json:"errors" db:"errors"`
}

type Metadata struct {
	*minio.UploadInfo `json:"upload_info" db:"upload_info"`
	SHA3_384_Encoded  string         `json:"sha3_384_encoded" db:"sha3_384_encoded"`
	Name              string         `json:"name" db:"name"`
	Version           string         `json:"version" db:"version"`
	Type              string         `json:"type" db:"type"`
	Summary           string         `json:"summary" db:"summary"`
	Description       string         `json:"description" db:"description"`
	Confinement       string         `json:"confinement" db:"confinement"`
	Base              string         `json:"base" db:"base"`
	Grade             string         `json:"grade" db:"grade"`
	Architectures     pq.StringArray `json:"architectures" db:"architectures"`
	Plugs             Plugs          `json:"plugs" db:"plugs"`
	Slots             Slots          `json:"slots" db:"slots"`
	RefreshControl    []string       `json:"refresh_control" db:"refresh_control"`
}

type SnapMeta struct {
	Name           string         `yaml:"name"`
	Version        string         `yaml:"version"`
	Summary        string         `yaml:"summary"`
	Description    string         `yaml:"description"`
	Architectures  pq.StringArray `yaml:"architectures"`
	Confinement    string         `yaml:"confinement"`
	Grade          string         `yaml:"grade"`
	Base           string         `yaml:"base"`
	Plugs          Plugs          `yaml:"plugs"`
	Slots          Slots          `yaml:"slots"`
	RefreshControl []string       `yaml:"refresh-control"`
}

type Plugs map[string]map[string]interface{}
type Slots map[string]map[string]interface{}
