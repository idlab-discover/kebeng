package models

import (
    "github.com/google/uuid"
    "gorm.io/gorm"
)

type Key struct {
    gorm.Model
    ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    Name             string
    SHA3384          string `gorm:"unique"`
    EncodedPublicKey string
    AccountID        uuid.UUID
    Account          Account
}

type SSHKey struct {
    gorm.Model
    ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    PublicKeyString string `gorm:"unique"`
    AccountID       uuid.UUID
    Account         Account
}

type Account struct {
    gorm.Model
    // override the default ID field to use uuid
    ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    DisplayName string `gorm:"unique"`
    Username    string `gorm:"unique"`
    Keys        []Key
    //SnapEntryIDs []uuid.UUID `gorm:"type:integer[];column:snap_entry_ids"`
    SSHKeys     []SSHKey
    Email       string
}
