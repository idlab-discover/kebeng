package model

import (
	"github.com/idlab-discover/kebeng/common/cerror"
)

type Metadata struct {
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	Type           string            `json:"type"`
	Summary        string            `json:"summary"`
	Description    string            `json:"description"`
	Confinement    string            `json:"confinement"`
	Base           string            `json:"base"`
	Grade          string            `json:"grade"`
	Architectures  []string          `json:"architectures"`
	Plugs          Plugs             `json:"plugs"`
	Slots          Slots             `json:"slots"`
	RefreshControl []string          `json:"refresh-control"`
	Errors         *cerror.ErrorList `json:"errors"`
}

type Plugs map[string]map[string]interface{}

type Slots map[string]map[string]interface{}
