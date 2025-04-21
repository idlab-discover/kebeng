package model

// ############# ACCOUNT SERVICE #############
type TestKey struct {
	ID               string `json:"id" db:"id"`
	Name             string `json:"name" db:"name"`
	SHA3384          string `json:"sha3384" db:"sha3384"` // Should be unique
	EncodedPublicKey string `json:"encoded_public_key" db:"encoded_public_key"`
	AccountID        string `json:"account_id" db:"account_id"`
	AccountEmail     string `json:"account_email" db:"account_email"`
}

type TestSSHKey struct {
	ID              string `json:"id" db:"id"`
	PublicKeyString string `json:"public_key_string" db:"public_key_string"` // should be unique
	AccountID       string `json:"account_id" db:"account_id"`
}

type TestAccount struct {
	ID             string `json:"id" db:"id"`
	DisplayName    string `json:"display_name" db:"display_name"`
	Username       string `json:"username" db:"username"`
	Email          string `json:"email" db:"email"`
	HashedPassword string `json:"hashed_password" db:"hashed_password"`
}

// ############# STORE SERVICE #############

type TestSnapEntry struct {
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Type        string  `json:"type" db:"type"`
	Confinement string  `json:"confinement" db:"confinement"`
	Base        string  `json:"base" db:"base"`
	Private     bool    `json:"private,omitempty" db:"private"`
	Status      string  `json:"status" db:"status"`
	Price       float64 `json:"price" db:"price"`
	Store       string  `json:"store" db:"store"`
	IconURL     string  `json:"icon_url" db:"icon_url"`
	AccountID   string  `json:"account_id" db:"account_id"`
}
