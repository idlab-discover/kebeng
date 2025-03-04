package message

type CreateAccountReq struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Username    string `json:"username"`
}

type RegisterSnapNameReq struct {
	SnapName  string `json:"snap_name"`
	IsPrivate bool   `json:"is_private"`
	Store     string `json:"store"`
}
