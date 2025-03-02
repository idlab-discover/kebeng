package message

import (

)

type CreateAccountReq struct {
    DisplayName string `json:"display_name"`
    Email string `json:"email"`
    Username string `json:"username"`
}
