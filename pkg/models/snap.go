package models

import (
	"context"
	"crypto"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/config/configkey"
	"github.com/idlab-discover/kebeng/pkg/objectstore"
	"github.com/idlab-discover/kebeng/pkg/store/responses"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/snap"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type SnapTrack struct {
	gorm.Model
	Name string

	SnapEntryID uuid.UUID
	SnapEntry   SnapEntry

	Risks []SnapChannel
}

type SnapChannel struct {
	gorm.Model
	Name        string
	SnapTrackID uint
	SnapEntryID uuid.UUID
	SnapEntry   SnapEntry
   
   // why store RevisionID when we have Revision?
	// TODO: fix this -- currently this is monotonically incrementing across ALL revisions, it should just be a given snap
	RevisionID uint
   // TODO: can have multiple revisions
	Revision   SnapRevision

	Branches []SnapBranch
}

// optional so going to leave this out for now
type SnapBranch struct {
	gorm.Model
	Name        string
	SnapRiskID  uint
	SnapEntryID uuid.UUID
	SnapEntry   SnapEntry

	RevisionID uint
	Revision   SnapRevision
}

type SnapEntry struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `json:"name"`
	Revisions   []SnapRevision
	Type        string
	Confinement string
	Base        string
	Uploads     []SnapUpload

	AccountID uuid.UUID
	Account   Account

	CreatedAt time.Time
}

type SnapRevision struct {
	gorm.Model
	SnapFilename   string
	SnapEntryID    uuid.UUID
	SHA3_384       string
	SHA3384Encoded string `gorm:"column:sha3_384_encoded"`
	Size           uint64
}

// created directly when something is uploaded afterwards assertions checks and a snap revision is created
type SnapUpload struct {
	gorm.Model
	Name     string
	UpDownID string
	Filesize uint
	// Channels is a comma-separated string of channels
	Channels    string
	SnapEntryID uuid.UUID
	SnapEntry   SnapEntry
}

// used as an object to pass to the GetSnap function to filter snaps on attributes
type SnapFilter struct {
	Name        string
	Type        string
	Confinement string
	Base        string
   // add more attributes if want to filtere on them
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

	logrus.Infof("Snap: %s, Revision: %d, URL: %s, SHA3: %s", se.Name, snapRevision.ID, downloadURL, actualSha3)

	storeSnap := &responses.StoreSnap{
		Name:     se.Name,
		Type:     snap.Type(se.Type),
		SnapID:   se.ID,
		Revision: int(snapRevision.ID),
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


func SnapFilterScope(filter *SnapFilter) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if filter == nil {
			return db
		}
		if filter.Name != "" {
			db = db.Where("name = ?", filter.Name)
		}
		if filter.Type != "" {
			db = db.Where("type = ?", filter.Type)
		}
		if filter.Confinement != "" {
			db = db.Where("confinement = ?", filter.Confinement)
		}
		if filter.Base != "" {
			db = db.Where("base = ?", filter.Base)
		}
		return db
	}
}
