package model

type CreateAccountRequest struct {
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	HashedPassword string `json:"hashed_password"`
}

type AccountPatchRequest struct {
	ShortNameSpace string `json:"short_namespace"`
}

type RequestStoreDeviceNonceRequest struct {
}

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

type FindSnapsRequest struct {
	Query         string   `json:"query"`
	Fields        []string `json:"fields"`
	Architectures []string `json:"architectures,omitempty"`
	Channels      []string `json:"channels,omitempty"`
	Confinements  []string `json:"confinements,omitempty"`
	Private       bool     `json:"private,omitempty"`
}

type RegisterSnapNameRequest struct {
	SnapName  string `json:"snap_name" binding:"required"` //TODO: check wheter this gets handled correctly
	IsPrivate bool   `json:"is_private" default:"false"`
	Store     string `json:"store" default:"default_store"`
}

type SnapBuildAssertionRequest struct {
	Assertion []byte `json:"assertion"`
}

type RefreshSnapRequest struct {
	Context []Context `json:"context"`
	Actions []*Action `json:"actions"`
	Fields  []string  `json:"fields"`
}

type Context struct {
	SnapID          string `json:"snap-id"`
	InstanceKey     string `json:"instance-key"`
	Revision        int    `json:"revision"`
	TrackingChannel string `json:"tracking-channel"`
	Epoch           struct {
		Read  []int `json:"read"`
		Write []int `json:"write"`
	} `json:"epoch"`
	RefreshedDate string `json:"refreshed-date"`
}

type Action struct {
	Action      string `json:"action"`
	InstanceKey string `json:"instance-key"`
	Name        string `json:"name"`
	Channel     string `json:"channel"`
	Epoch       *struct {
		Read  []int `json:"read"`
		Write []int `json:"write"`
	} `json:"epoch"`
}

type SnapDownloadRequest struct {
	SnapName string `json:"snap_name"`
	Channel  string `json:"channel"`
}

type CohortKeysRequest struct {
	SnapNames []string `json:"snap_names" binding:"required,min=1"`
}

type ProvisionSnapNamesRequest struct {
	Count int `json:"count" binding:"required,min=1"`
}

type ProvisionDeltaBaseRequest struct {
	Count            int    `json:"count" binding:"required,min=1"`
	BaseSnapFileName string `json:"base_snap_file_name" binding:"required"`
}

type DeltaUploadRequest struct {
	SnapNames            []string   `json:"snap_names" binding:"required,min=1"`
	BaseRevisionSequence uint32   `json:"base_revision_sequence" binding:"required"`
	DeltaFileName        string   `json:"delta_file_name" binding:"required"`
	DeltaFormat          string   `json:"delta_format" binding:"required"`
	TracksAndChannels    []string `json:"tracks_and_channels" binding:"required"`
	TimeoutSeconds       uint32   `json:"timeout_seconds"`
}

type DeltaDownloadRequest struct {
	SnapName    string `json:"snap_name" bidning:"required"`
	DeltaFormat string `json:"delta_format" binding:"required"`
	DeltaName   string `json:"delta_name" binding:"required"`
}
