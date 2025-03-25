package auth

import "github.com/idlab-discover/kebeng/common/cerror"

// the formats  are  {"name" : "package_name", "series" : "package_series"}
//
//	or   {"snap_id" : "package_snap_id"}
type PackageRestriction struct {
	Name   string `json:"name"`
	Series string `json:"series"`
	SnapId string `json:"snap_id"`
}

type GenerateMacaroonRequest struct {
	Permissions []string             `json:"permissions"`
	Channels    []string             `json:"channels"`
	Packages    []PackageRestriction `json:"packages"`
	Expires     string               `json:"expires"`
}

type MacaroonResponse struct {
	Macaroon string           `json:"macaroon"`
	Errors   cerror.ErrorList `json:"error_list"`
}

type AuthData struct {
	Authorization string `json:"authorization"`
}

type VerifyRequest struct {
	AuthData AuthData `json:"auth_data"`
}

type AccountRequest struct {
	Authorization AuthData `header:"Authorization" binding:"required"`
}
