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

type FindSnapResponse struct {
	Results []FindSnapResult `json:"results"`
}

func NewFindSnapResponse() *FindSnapResponse {
	return &FindSnapResponse{
		Results: make([]FindSnapResult, 0),
	}
}

type FindSnapResult struct {
	Name     string       `json:"name"`
	SnapID   string       `json:"snap-id"`
	Snap     Snap         `json:"snap"`
	Revision SnapRevision `json:"revision"`
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
	Context             []*Context     `json:"context"`
	Actions             []*Action      `json:"actions"`
	Fields              []string       `json:"fields"`
	AssertionMaxFormats map[string]int `json:"assertion-max-formats,omitempty"`
}

type Context struct {
	SnapID           string     `json:"snap-id"`
	InstanceKey      string     `json:"instance-key"`
	Revision         int        `json:"revision"`
	TrackingChannel  string     `json:"tracking-channel"`
	Epoch            Epoch      `json:"epoch"`
	RefreshedDate    *time.Time `json:"refreshed-date,omitempty"`
	IgnoreValidation bool       `json:"ignore-validation,omitempty"`
	CohortKey        string     `json:"cohort-key,omitempty"`
}

type Action struct {
	Action           string     `json:"action"`
	InstanceKey      string     `json:"instance-key,omitempty"`
	Name             string     `json:"name,omitempty"`
	SnapID           string     `json:"snap-id"`
	Channel          string     `json:"channel,omitempty"`
	Revision         int        `json:"revision,omitempty"`
	CohortKey        string     `json:"cohort-key,omitempty"`
	IgnoreValidation bool       `json:"ignore-validation,omitempty"`
	Epoch            Epoch      `json:"epoch,omitempty"`
	Key              string     `json:"key,omitempty"`
	Assertions       []any      `json:"assertions,omitempty"`
	ValidationSets   [][]string `json:"validation-sets,omitempty"`
}

type Epoch struct {
	Read  []uint32 `json:"read"`
	Write []uint32 `json:"write"`
}

type RefreshSnapResults struct {
	Results []*RefreshSnapResult `json:"results"`
}

type RefreshSnapResult struct {
	Result              string       `json:"result,omitempty"`
	InstanceKey         string       `json:"instance-key,omitempty"`
	SnapId              string       `json:"snap-id,omitempty"`
	CohortKey           string       `json:"cohort-key,omitempty"`
	Name                string       `json:"name,omitempty"`
	Snap                *RefreshSnap `json:"snap,omitempty"`
	Key                 string       `json:"key"`
	AssertionStreamURLs []string     `json:"assertion-stream-urls"`
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

type Category struct {
	Featured bool   `json:"featured"`
	Name     string `json:"name"`
}

type Download struct {
	URL      *string `json:"url,omitempty"`
	Sha3_384 *string `json:"sha3-384,omitempty"`
	Size     *uint64 `json:"size,omitempty"`
	Deltas   []Delta `json:"deltas,omitempty"`
}

type Delta struct {
	Format   string `json:"format"`
	Sha3_384 string `json:"sha3-384"`
	Size     uint64 `json:"size"`
	Source   uint64 `json:"source"`
	Target   uint64 `json:"target"`
	URL      string `json:"url"`
}

// SnapRevision represents a snap revision in the store
type SnapRevision struct {
	Base          string   `json:"base,omitempty"`
	Channel       string   `json:"channel,omitempty"`
	CommonIds     []string `json:"common-ids,omitempty"`
	Confinement   string   `json:"confinement,omitempty"`
	Revision      int      `json:"revision"`
	Version       string   `json:"version"`
	Status        string   `json:"status"`
	Architectures []string `json:"architectures"`
	Channels      []string `json:"channels"`
	Download      Download `json:"download,omitempty"`
	Type          string   `json:"type,omitempty"`
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
	Categories      []Category     `json:"categories,omitempty"`
	Status          string         `json:"status,omitempty"`
	Price           float64        `json:"price,omitempty"`
	Since           time.Time      `json:"since"`
	SnapID          string         `json:"snap-id"`
	Store           string         `json:"store"`
	Private         bool           `json:"private"`
	IconURL         string         `json:"icon_url,omitempty" default:""`
	Publisher       Publisher      `json:"publisher"`
	LatestComments  []SnapComment  `json:"latest_comments"`
	LatestRevisions []SnapRevision `json:"latest_revisions"`
	Contact         any            `json:"contact"`
	Description     string         `json:"description"`
	License         string         `json:"license"`
	Links           Links          `json:"links"`
	Media           []Media        `json:"media"`
	Prices          any            `json:"prices"` // TODO: Construct prices in many currencies based on stored price field.
	Summary         string         `json:"summary"`
	Title           string         `json:"title"`
	Website         string         `json:"website"`
}

type Links struct {
	Contact   []string `json:"contact"`
	Donations []string `json:"donations"`
	Issues    []string `json:"issues"`
	Source    []string `json:"source"`
	Website   []string `json:"website"`
}

type Media struct {
	Height int    `json:"height"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
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

type CreateCohortsRequest struct {
	SnapNames []string `json:"snaps"`
}

type CreateCohortsResult struct {
	CohortKeys map[string]string `json:"cohort-keys"`
}

// CohortKey: A CohortKey consists of the following fields (in order):
// Version SnapID CreatedAt Signature
// A CohortKey is sent base64 encoded to the client. It is not stored server-side; all information
// to understand and validate a CohortKey is contained within the key.
// Example:
//
//	Key Version     Snap ID            CreatedAt(Unix Timestamp)                 Signature
//	   v               v                       v                                     v
//	   1 iCEzvDZMvRrIWd5XLxgff6Tc6Zx20aeO 1777020308 09aef5c41697e06c50b715288cf9efa67b45df3a5494a4a8ca1452ad7fe03878
//
// What is sent to client:
//
//	Base64 encoding of the above string
//	                 v
//
// MSBpQ0V6dkRaTXZScklXZDVYTHhnZmY2VGM2WngyMGFlTyAxNzc3MDIwMzA4IDA5YWVmNWM0MTY5N2UwNmM1MGI3MTUyODhjZjllZmE2N2I0NWRmM2E1NDk0YTRhOGNhMTQ1MmFkN2ZlMDM4Nzg=
type CohortKey struct {
	Version   uint8
	SnapID    string
	CreatedAt time.Time
	Signature string
}

// NOTE: The below formats are used to enable testing of the ApplyUploadDelta Flow
// Actual formats for these requests/responses will most likely be different compared to what is defined here
type SnapDeltaPushRequest struct {
	Name                 string   `json:"name" binding:"required"`
	UnscannedFileName    string   `json:"updown_id" binding:"required"`
	BaseRevisionSequence uint32   `json:"base_revision_sequence" binding:"required"`
	DeltaSha3_385        string   `json:"delta_sha3_384" binding:"required"`
	DeltaFormat          string   `json:"delta_format" binding:"required"`
	TracksAndChannels    []string `json:"tracks_and_channels" binding:"required"`
	TimeoutSeconds       uint32   `json:"timeout_seconds"`
}

type SnapDeltaPushResponse struct {
	SnapName string `json:"snap_name"`
	Status   string `json:"status"`
	Revision uint32 `json:"revision"`
}
