package model

type Alias struct {
	Name   string
	Target string
}

type Plugs map[string]map[string]interface{}

type Slot struct {
	AllowInstallation   *bool
	DenyInstallation    *bool
	AllowConnection     *bool
	DenyConnection      *bool
	AllowAutoConnection *bool
	DenyAutoConnection  *bool
}

type SlotMap map[string]*Slot
