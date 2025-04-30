package model

type SnapcraftUploadRequest struct {
	SnapNames []string `json:"snap_names"`
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
