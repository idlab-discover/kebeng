package snap

type RequestStoreDeviceNonceRequest struct {
}

type RequestStoreDeviceNonceResponse struct {
	Nonce string `json:"nonce"`
}

type FindSnapsRequest struct {
	Name string `json:"name"`
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
	Context []struct {
		SnapID          string `json:"snap-id"`
		InstanceKey     string `json:"instance-key"`
		Revision        int    `json:"revision"`
		TrackingChannel string `json:"tracking-channel"`
		Epoch           struct {
			Read  []int `json:"read"`
			Write []int `json:"write"`
		} `json:"epoch"`
		RefreshedDate string `json:"refreshed-date"`
	} `json:"context"`
	Actions []*Action `json:"actions"`
	Fields  []string  `json:"fields"`
}

type Action struct {
	Action      string  `json:"action"`
	InstanceKey string  `json:"instance-key"`
	Name        *string `json:"name"`
	SnapID      *string `json:"snap-id"`
	Channel     string  `json:"channel"`
	Epoch       *struct {
		Read  []int `json:"read"`
		Write []int `json:"write"`
	} `json:"epoch"`
}
