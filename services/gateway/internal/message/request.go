package message

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

// the formats are  {"name" : "package_name", "series" : "package_series"}
//
//	or   {"snap_id" : "package_snap_id"}
type PackageRestriction struct {
	Name   string `json:"name"`
	Series string `json:"series"`
	SnapId string `json:"snap_id"`
}

type GenerateMacaroonReq struct {
	Permissions []string             `json:"permissions"`
	Channels    []string             `json:"channels"`
	Packages    []PackageRestriction `json:"packages"`
	Expires     string               `json:"expires"`
}
