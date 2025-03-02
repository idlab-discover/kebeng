package requests

import "github.com/google/uuid"

type AddAccount struct {
	Username    string
	AcccountId  uuid.UUID
	Email       string
	DisplayName string
}

type AddTrack struct {
	SnapName  string
	TrackName string
}
