package responses

import "github.com/google/uuid"

type RegisterSnap struct {
	Id   uuid.UUID `json:"snap_id"`
	Name string    `json:"snap_name"`
}

type Status struct {
	Processed bool   `json:"processed"`
	Code      string `json:"code"`
	Revision  int    `json:"revision"`
}

type SnapRelease struct {
	Success bool
}
