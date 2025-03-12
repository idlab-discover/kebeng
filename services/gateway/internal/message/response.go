package message

import "github.com/idlab-discover/kebeng/services/gateway/internal/errors"

type CreateAccountResponse struct {
	Id string `json:"id"`
}

type RegisterSnapNameRes struct {
	SnapId   string `json:"snap_id"`
	SnapName string `json:"snap_name"`
}

type MacaroonResponse struct {
	Macaroon string           `json:"macaroon"`
	Errors   errors.ErrorList `json:"error_list"`
}

type VerifyAccount struct {
	Email       string           `json:"email"`
	DisplayName string           `json:"displayname"`
	OpenId      string           `json:"openid"`
	Verified    bool             `json:"verified"`
	Errors      errors.ErrorList `json:"error_list"`
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
	Errors                errors.ErrorList `json:"error_list"`
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
	Errors          errors.ErrorList `json:"error_list"`
}
