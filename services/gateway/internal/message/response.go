package message

import (
	"time"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
)



type RegisterSnapNameRes struct {
	SnapId   string `json:"snap_id"`
	SnapName string `json:"snap_name"`
}

type VerifyAccount struct {
	Email       string           `json:"email"`
	DisplayName string           `json:"displayname"`
	OpenId      string           `json:"openid"`
	Verified    bool             `json:"verified"`
	Errors      cerror.ErrorList `json:"error_list"`
}

type VerifyResponse struct {
	Allowed               bool             `json:"allowed"`
	DeviceRefreshRequired bool             `json:"device_refresh_required"`
	RefreshRequired       bool             `json:"refresh_required"`
	Account               *VerifyAccount   `json:"account,omitempty"`
	Device                *string          `json:"device"`
	LastAuth              string           `json:"last_auth"`
	Permissions           *[]string        `json:"permissions"`
	SnapIds               *string          `json:"snap_ids"`
	Channels              *string          `json:"channels"`
	Errors                cerror.ErrorList `json:"error_list"`
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
	Result      *string      `json:"result,omitempty"`
	InstanceKey *string      `json:"instance-key,omitempty"`
	SnapId      *string      `json:"snap-id,omitempty"`
	Name        *string      `json:"name,omitempty"`
	Snap        *RefreshSnap `json:"snap,omitempty"`
}

type RefreshSnap struct {
	Architectures *[]string  `json:"architectures,omitempty"`
	SnapId        *string    `json:"snap-id,omitempty"`
	Name          *string    `json:"name,omitempty"`
	Publisher     *Publisher `json:"publisher,omitempty"`
	Download      *Download  `json:"download,omitempty"`
	Version       *string    `json:"version,omitempty"`
	Confinement   *string    `json:"confinement,omitempty"`
	Revision      *float64   `json:"revision,omitempty"`
	Type          *string    `json:"type,omitempty"`
	Base          *string    `json:"base,omitempty"`
}

type Download struct {
	URL      *string  `json:"url,omitempty"`
	Sha3_384 *string  `json:"sha3-384,omitempty"`
	Size     *float64 `json:"size,omitempty"`
}

// ********************* AccountResponse **************************
// AccountKey represents an account key object
type AccountKey struct {
	Name            string    `json:"name"`
	PublicKeySHA384 string    `json:"public-key-sha3-384"`
	Since           time.Time `json:"since"`
	Until           time.Time `json:"until,omitempty"`
}

// Publisher represents a snap publisher
type Publisher struct {
	ID          string `json:"id"`
	DisplayName string `json:"display-name,omitempty"`
	Username    string `json:"username"`
	Validation  string `json:"validation,omitempty"`
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
	IconURL         *string        `json:"icon_url,omitempty"`
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

// AccountResponse represents the response returned by the account API
type AccountResponse struct {
	AccountKeys []AccountKey               `json:"account-keys"`
	DisplayName string                     `json:"display-name"` // user's full name
	Email       string                     `json:"email"`
	ID          uuid.UUID                  `json:"id"`
	Validation  string                     `json:"validation"` // validation status
	Snaps       map[string]map[string]Snap `json:"snaps"`      // Properly nested structure (Series -> Snap Name -> Snap)
	Stores      []Store                    `json:"stores"`     // list of stores the user has access to
	Username    string                     `json:"username"`   // store username

	// Deprecated Fields, here for backwards compatibility but may not always hold correct values
	AccountID      string       `json:"account_id,omitempty"`
	AccountKeysOld []AccountKey `json:"account_keys,omitempty"`
	DisplayNameOld string       `json:"displayname,omitempty"`
	Namespace      string       `json:"namespace,omitempty"`
	OpenID         string       `json:"openid_identifier,omitempty"`
	ShortNamespace string       `json:"short_namespace,omitempty"`
}

// *************************** End AccountResponse ****************************

// ********************* AccountPatchResponse **************************
// AccountPatchResponse represents the response returned by the account patch API
type AccountPatchResponse struct {
	ShortNamespace string `json:"short_namespace"`
}

// *************************** End AccountPatchResponse ****************************
