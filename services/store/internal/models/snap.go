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
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/snap"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Entry = base information, first entry point, global information...
type SnapEntry struct {
	gorm.Model
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name           string    `json:"name"`
	Revisions      []SnapRevision
	Type           string
	Confinement    string
	Base           string
	Private        bool
	Uploads        []SnapUpload
	AccountID      uuid.UUID
	Status         string        // NOT YET IMPLEMENTED
	Price          float64       // NOT YET IMPLEMENTED
	Store          string        // NOT YET IMPLEMENTED
	IconURL        string        // NOT YET IMPLEMENTED
	LatestComments []SnapComment // NOT YET IMPLEMENTED
}

// Track = latest, or things like 2.0, 2.1, 2.2
type SnapTrack struct {
	gorm.Model
	Name string

	SnapEntryID uuid.UUID
	SnapEntry   SnapEntry

	Risks []SnapChannel
}

// Channel = stable, beta, edge, candidate
type SnapChannel struct {
	gorm.Model
	ID          string `gorm:"primaryKey"`
	Name        string
	SnapTrackID uint
	SnapEntryID uuid.UUID
	SnapEntry   SnapEntry

	// TODO: fix this -- currently this is monotonically incrementing across ALL revisions, it should just be a given snap
	RevisionID string
	Revision   SnapRevision

	Branches []SnapBranch
}

// keep this for later... maybe
type SnapBranch struct {
	gorm.Model
	ID          string `gorm:"primaryKey"`
	Name        string
	SnapRiskID  uint
	SnapEntryID uuid.UUID
	SnapEntry   SnapEntry

	RevisionID uint
	Revision   SnapRevision
}

// Revision = a specific version of a snap, not necessarily a release
type SnapRevision struct {
	gorm.Model
	ID             string `gorm:"primaryKey"`
	SnapFilename   string
	SnapEntryID    uuid.UUID
	SHA3_384       string
	SHA3384Encoded string `gorm:"column:sha3_384_encoded"`
	Size           uint64
	SequenceNumber uint
}

// SnapUpload = a specific upload of a snap, with a specific file, info about file, etc
type SnapUpload struct {
	gorm.Model
	ID       string `gorm:"primaryKey"`
	Name     string
	UpDownID string
	Filesize uint
	// Channels is a comma-separated string of channels
	Channels    string
	SnapEntryID uuid.UUID
	SnapEntry   SnapEntry
}

type SnapComment struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AuthorID    uuid.UUID
	Since       time.Time
	Reason      string
	Comment     string
	SnapEntryID uuid.UUID
	SnapEntry   SnapEntry
}

func (se *SnapEntry) ToStoreSnap(snapRevision *SnapRevision) (*responses.StoreSnap, error) {
	downloadURL := fmt.Sprintf(viper.GetString(configkey.StoreAPIURL)+"/download/snaps/%s", snapRevision.SnapFilename)
	base := snapRevision.SnapFilename
	obs := objectstore.NewObjectStore()
	h := crypto.SHA3_384.New()
	objectPtr, err := obs.MinioClient.GetObject(context.Background(), "snaps", base, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	bytes, _ := io.ReadAll(objectPtr)
	h.Write(bytes)
	actualSha3 := fmt.Sprintf("%x", h.Sum(nil))

	logrus.Infof("Snap: %s, Revision: %d, URL: %s, SHA3: %s", se.Name, snapRevision.SequenceNumber, downloadURL, actualSha3)

	storeSnap := &responses.StoreSnap{
		Name:     se.Name,
		Type:     snap.Type(se.Type),
		SnapID:   se.ID.String(),
		Revision: int(snapRevision.SequenceNumber),
		Download: responses.StoreSnapDownload{
			Sha3_384: actualSha3,
			Size:     snapRevision.Size,
			URL:      downloadURL,
		},
		Confinement: se.Confinement,
		Base:        &se.Base,
	}

	return storeSnap, nil
}
