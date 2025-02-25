package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Key struct {
	gorm.Model
	Name             string
	SHA3384          string `gorm:"unique"`
	EncodedPublicKey string
	AccountID        uuid.UUID
	Account          Account
}

type SSHKey struct {
	gorm.Model
	PublicKeyString string `gorm:"unique"`
	AccountID       uint
	Account         Account
}

type Account struct {
	gorm.Model
	// AccountId is the same as publisher-id and developer-id

	ID          uuid.UUID `gorm:"unique"`
	DisplayName string    `gorm:"unique"`
	Username    string    `gorm:"unique"`
	Keys        []Key
	SnapEntries []SnapEntry
	SSHKeys     []SSHKey
	Email       string
}
