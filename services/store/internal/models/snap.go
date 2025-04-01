package models

import (
	"context"
	"crypto"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/pkg/store/responses"
	"github.com/idlab-discover/kebeng/services/store/internal/config/configkey"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	"github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/snap"
	"github.com/spf13/viper"
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
	SnapName               *string        `json:"snap_name,omitempty" db:"snap_name"`
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

func (se *SnapEntry) ToStoreSnap(snapRevision *SnapRevision) (*responses.StoreSnap, error) {
	downloadURL := fmt.Sprintf(viper.GetString(configkey.StoreAPIURL)+"/download/snaps/%s", snapRevision.SnapName)
	base := snapRevision.SnapName
	obs := objectstore.NewObjectStore()
	h := crypto.SHA3_384.New()
	objectPtr, err := obs.MinioClient.GetObject(context.Background(), "snaps", *base, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	bytes, _ := io.ReadAll(objectPtr)
	h.Write(bytes)
	actualSha3 := fmt.Sprintf("%x", h.Sum(nil))

	logrus.Infof("Snap: %s, Revision: %d, URL: %s, SHA3: %s", se.Name, snapRevision.SequenceNumber, downloadURL, actualSha3)

	storeSnap := &responses.StoreSnap{
		Name:     se.Name,
		Type:     snap.Type(*se.Type),
		SnapID:   se.ID,
		Revision: int(*snapRevision.SequenceNumber),
		Download: responses.StoreSnapDownload{
			Sha3_384: actualSha3,
			Size:     *snapRevision.Size,
			URL:      downloadURL,
		},
		Confinement: *se.Confinement,
		Base:        se.Base,
	}

	return storeSnap, nil
}
