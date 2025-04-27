package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
)

type RequestStoreDeviceNonceRequest struct {
}

type RequestStoreDeviceNonceResponse struct {
	Nonce string `json:"nonce"`
}

type FindSnapsRequest struct {
	Name string `json:"name"`
}

type RegisterSnapNameRequest struct {
	SnapName  string `json:"snap_name" binding:"required"` //TODO: check wheter this gets handled correctly
	IsPrivate bool   `json:"is_private" default:"false"`
	Store     string `json:"store" default:"default_store"`
}

type RegisterSnapNameResponse struct {
	SnapId   string `json:"snap_id"`
	SnapName string `json:"snap_name"`
}

type SnapBuildAssertionRequest struct {
	Assertion []byte `json:"assertion"`
}

type RefreshSnapRequest struct {
	Context []struct {
		SnapID          string `json:"snap-id"`
		InstanceKey     string `json:"instance-key"`
		Revision        int    `json:"revision"`
		TrackingChannel string `json:"tracking-channel"`
		Epoch           struct {
			Read  []int `json:"read"`
			Write []int `json:"write"`
		} `json:"epoch"`
		RefreshedDate string `json:"refreshed-date"`
	} `json:"context"`
	Actions []*Action `json:"actions"`
	Fields  []string  `json:"fields"`
}

type Action struct {
	Action      string `json:"action"`
	InstanceKey string `json:"instance-key"`
	Name        string `json:"name"`
	Channel     string `json:"channel"`
	Epoch       *struct {
		Read  []int `json:"read"`
		Write []int `json:"write"`
	} `json:"epoch"`
}

type SnapBuildAssertionResp struct {
	AuthorityId     string           `json:"authority_id"`
	Grade           string           `json:"grade"`
	SignKeySha3_384 string           `json:"sign_key_sha3_384"`
	SnapId          string           `json:"snap_id"`
	SnapSha3_384    string           `json:"snap_sha3_384"`
	SnapSize        int              `json:"snap_size"`
	Timestamp       string           `json:"timestamp"`
	Revision        string           `json:"revision"`
	Type            string           `json:"type"`
	Errors          cerror.ErrorList `json:"error_list"`
}

type RefreshSnapResponses struct {
	Responses []*RefreshSnapResult `json:"results"`
}

type RefreshSnapResult struct {
	Result      string       `json:"result,omitempty"`
	InstanceKey string       `json:"instance-key,omitempty"`
	SnapId      string       `json:"snap-id,omitempty"`
	Name        string       `json:"name,omitempty"`
	Snap        *RefreshSnap `json:"snap,omitempty"`
}

type RefreshSnap struct {
	Architectures []string   `json:"architectures,omitempty"`
	SnapId        string     `json:"snap-id,omitempty"`
	Name          string     `json:"name,omitempty"`
	Publisher     *Publisher `json:"publisher,omitempty"`
	Download      *Download  `json:"download,omitempty"`
	Version       string     `json:"version,omitempty"`
	Confinement   string     `json:"confinement,omitempty"`
	Revision      uint32     `json:"revision,omitempty"`
	Type          string     `json:"type,omitempty"`
	Base          string     `json:"base,omitempty"`
}

type Download struct {
	URL      *string `json:"url,omitempty"`
	Sha3_384 *string `json:"sha3-384,omitempty"`
	Size     *uint64 `json:"size,omitempty"`
}

// SnapRevision represents a snap revision in the store
type SnapRevision struct {
	Revision      int      `json:"revision"`
	Version       string   `json:"version"`
	Status        string   `json:"status"`
	Architectures []string `json:"architectures"`
	Channels      []string `json:"channels"`
}

// SnapComment represents a comment in the context of an under-review or revoked name
type SnapComment struct {
	Author struct {
		ID          uuid.UUID `json:"id"`
		DisplayName string    `json:"display-name"`
		Username    string    `json:"username"`
		Validation  string    `json:"validation"`
	} `json:"author"`
	Since   time.Time `json:"since"`
	Reason  string    `json:"reason"`
	Comment string    `json:"comment"`
}

// Snap represents a snap owned or collaborated on by the user
type Snap struct {
	Status          string         `json:"status"`
	Price           float64        `json:"price,omitempty"`
	Since           time.Time      `json:"since"`
	SnapID          string         `json:"snap-id"`
	Store           string         `json:"store"`
	Private         bool           `json:"private"`
	IconURL         string         `json:"icon_url,omitempty" default:""`
	Publisher       Publisher      `json:"publisher"`
	LatestComments  []SnapComment  `json:"latest_comments"`
	LatestRevisions []SnapRevision `json:"latest_revisions"`
}

// Store represents a store object accessible by the user
type Store struct {
	Name  string    `json:"name"`
	ID    uuid.UUID `json:"id"`
	Roles []string  `json:"roles"`
}

type SnapPushRequest struct {
	Name              string   `json:"name"`
	DryRun            bool     `json:"dry_run"`
	UnscannedFileName string   `json:"updown_id"`
	Series            string   `json:"series"`
	BinaryFileSize    int64    `json:"binary_filesize"`
	SourceUploaded    bool     `json:"source_uploaded"`
	Channels          []string `json:"channels"`
}

type UnscannedUploadRequest struct {
	Data string `json:"data"`
}
