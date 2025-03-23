package message

type RequestStoreDeviceNonceReq struct {
}

type RequestStoreDeviceNonceRes struct {
	Nonce string `json:"nonce"`
}

type FindSnapsRequest struct {
	Name string `json:"name"`
}

type CreateAccountRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Username    string `json:"username"`
}

type RegisterSnapNameReq struct {
	SnapName  string `json:"snap_name" binding:"required"` //TODO: check wheter this gets handled correctly
	IsPrivate bool   `json:"is_private" default:"false"`
	Store     string `json:"store" default:"default_store"`
}

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

type SnapBuildAssertionReq struct {
	Assertion []byte `json:"assertion"`
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

type AccountPatchRequest struct {
	ShortNameSpace string `json:"short_namespace"`
}
