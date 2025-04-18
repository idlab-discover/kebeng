package model

type Alias struct {
	Name   string
	Target string
}

// TODO: both Plug and Slot should be able to have more complex types than just bools (see https://ubuntu.com/core/docs/reference/assertions/snap-declaration)
// aswel but keep simple for now

// make pointers out of the bool values so that we can use nil to indicate that the field is not set
type Plug struct {
	AllowInstallation   *bool
	DenyInstallation    *bool
	AllowConnection     *bool
	DenyConnection      *bool
	AllowAutoConnection *bool
	DenyAutoConnection  *bool
}

// docs at  say that Slot has similar fields to Plug so for now the same ones
// make pointers out of the bool values so that we can use nil to indicate that the field is not set
type Slot struct {
	AllowInstallation   *bool
	DenyInstallation    *bool
	AllowConnection     *bool
	DenyConnection      *bool
	AllowAutoConnection *bool
	DenyAutoConnection  *bool
}
