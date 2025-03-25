package account

type CreateAccountRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Username    string `json:"username"`
}

type CreateAccountResponse struct {
	Id string `json:"id"`
}

type AccountPatchRequest struct {
	ShortNameSpace string `json:"short_namespace"`
}