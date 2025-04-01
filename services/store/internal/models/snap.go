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

// Entry = base information, first entry point, global information...
type SnapEntry struct {
	ID             uuid.UUID  `db:"id"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
	DeletedAt      *time.Time `db:"deleted_at"`
	Name           string     `json:"name" db:"name"`
	Revisions      []*SnapRevision
	Type           *string `json:"type" db:"type"`
	Confinement    *string `json:"confinement" db:"confinement"`
	Base           *string `json:"base" db:"base"`
	Private        *bool   `json:"private" db:"private"`
	Uploads        []*SnapUpload
	AccountID      uuid.UUID      `db:"account_id"`
	Status         *string        `db:"status"`   // NOT YET IMPLEMENTED
	Price          *float64       `db:"price"`    // NOT YET IMPLEMENTED
	Store          *string        `db:"store"`    // NOT YET IMPLEMENTED
	IconURL        *string        `db:"icon_url"` // NOT YET IMPLEMENTED
	LatestComments []*SnapComment // NOT YET IMPLEMENTED

}

// Track = latest, or things like 2.0, 2.1, 2.2
type SnapTrack struct {
	ID          uuid.UUID  `db:"id"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
	Name        string     `json:"name" db:"name"`
	SnapEntryID uuid.UUID  `db:"entry_id"`
	SnapEntry   *SnapEntry
	Channels    []*SnapChannel
}

// Channel = stable, beta, edge, candidate
type SnapChannel struct {
	ID          uuid.UUID  `db:"id"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
	Name        string     `json:"name" db:"name"`
	SnapTrackID uuid.UUID  `db:"snap_track_id"`
	SnapEntryID uuid.UUID  `db:"entry_id"`
	SnapEntry   *SnapEntry

	RevisionID uuid.UUID `db:"revision_id"`
	Revision   *SnapRevision

	Branches []*SnapBranch
}

// keep this for later... maybe
type SnapBranch struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `json:"name" db:"name"`
	SnapRiskID  uuid.UUID `db:"snap_risk_id"`
	SnapEntryID uuid.UUID `db:"entry_id"`
	SnapEntry   *SnapEntry

	RevisionID uuid.UUID `db:"revision_id"`
	Revision   *SnapRevision
}

// Revision = a specific version of a snap, not necessarily a release
type SnapRevision struct {
	ID                     uuid.UUID      `db:"id"`
	CreatedAt              time.Time      `db:"created_at"`
	UpdatedAt              time.Time      `db:"updated_at"`
	DeletedAt              *time.Time     `db:"deleted_at"`
	SnapName               *string        `db:"snap_name"`
	BuildAssertionFileName *string        `db:"build_assertion_filename"`
	SnapEntryID            uuid.UUID      `db:"entry_id"`
	SHA3_384               *string        `db:"sha3_384"`
	SHA3_384_Encoded       *string        `db:"sha3_384_encoded"`
	Size                   *uint64        `db:"size"`
	SequenceNumber         *uint          `db:"sequence_number"`
	Architectures          pq.StringArray `db:"architectures"` // TODO: check if this is supposed to be stored here
	Status                 *string        `db:"status"`
	Version                *string        `db:"version"`
}

// SnapUpload = a specific upload of a snap, with a specific file, info about file, etc
type SnapUpload struct {
	ID        uuid.UUID  `db:"id"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
	UpDownID  string     `db:"up_down_id"`
	Filesize  uint       `db:"filesize"`
	// Channels is a comma-separated string of channels
	Channels    pq.StringArray `db:"channels"`
	SnapEntryID uuid.UUID      `db:"entry_id"`
	SnapEntry   *SnapEntry
}

type SnapComment struct {
	ID          uuid.UUID  `db:"id"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
	AuthorID    uuid.UUID  `db:"author_id"`
	Since       time.Time  `db:"since"`
	Reason      string     `json:"reason"`
	Comment     string     `json:"comment"`
	SnapEntryID uuid.UUID  `db:"entry_id"`
	SnapEntry   *SnapEntry
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
