package message

import (

)

type CreateAccountReq struct {
    DisplayName string `json:"display_name"`
    Email string `json:"email"`
    Username string `json:"username"`
}


// the formats are  {"name" : "package_name", "series" : "package_series"}
//             or   {"snap_id" : "package_snap_id"}
type PackageRestriction struct {
    Name string `json:"name"`
    Series string `json:"series"`
    SnapId string `json:"snap_id"`
}

type GenerateMacaroonReq struct {
    Permissions []string `json:"permissions"`
    Channels []string `json:"channels"`
    Packages []PackageRestriction `json:"packages"`
    Expires string `json:"expires"`
}

