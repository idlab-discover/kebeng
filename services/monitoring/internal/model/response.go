package model

import "github.com/idlab-discover/kebeng/common/cerror"

type UploadStatusResponse struct {
	Code      string           `json:"code"`
	Processed bool             `json:"processed"`
	Revision  int              `json:"revision"`
	Errors    cerror.ErrorList `json:"errors"`
}

type UnscannedUploadResponse struct {
	Successful bool   `json:"successful"`
	UploadID   string `json:"upload_id"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
}

type SnapPushResponse struct {
	Success          bool   `json:"success"`
	SnapName         string `json:"snap_name"`
	UploadID         string `json:"upload_id"`
	StatusDetailsURL string `json:"status_details_url"`
}
