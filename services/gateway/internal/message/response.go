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
    Macaroon string `json:"macaroon"`
    Errors  errors.ErrorList `json:"error_list"`
}
