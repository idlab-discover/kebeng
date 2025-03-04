package message

type CreateAccountRes struct {
	Id string `json:"id"`
}

type RegisterSnapNameRes struct {
	SnapId   string `json:"snap_id"`
	SnapName string `json:"snap_name"`
}
