package model

type Alias struct {
	Name   string
	Target string
}

// TODO: both Plug and Slot should be able to have more complex types than just bools (see https://ubuntu.com/core/docs/reference/assertions/snap-declaration)
// aswel but keep simple for now

// NOTE: maybe change this to use an enum? with states like Allowed, Denied and NotSet and can be expanded later on to support more complex objects
// make pointers out of the bool values so that we can use nil to indicate that the field is not set
type Plug struct {
	AllowInstallation   *bool
	DenyInstallation    *bool
	AllowConnection     *bool
	DenyConnection      *bool
	AllowAutoConnection *bool
	DenyAutoConnection  *bool
}

type PlugMap map[string]*Plug

type Slot struct {
	AllowInstallation   *bool
	DenyInstallation    *bool
	AllowConnection     *bool
	DenyConnection      *bool
	AllowAutoConnection *bool
	DenyAutoConnection  *bool
}

type SlotMap map[string]*Slot
