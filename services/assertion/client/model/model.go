package model

type Alias struct {
	Name   string
	Target string
}

type Plugs map[string]map[string]interface{}

type Slots map[string]map[string]interface{}
