package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
)

type CreateAccountRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Username    string `json:"username"`
}

type CreateAccountResponse struct {
	Id string `json:"id"`
}

type AccountPatchRequest struct {
	ShortNameSpace string `json:"short_namespace"`
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
