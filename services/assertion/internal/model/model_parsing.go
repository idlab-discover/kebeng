package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// NOTE: this file contains the parsing functions to convert the JSONB in the DB to a valid Go struct.
// This can be extended later on to support more complex parsing if needed.

func (p *Plugs) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan Plugs: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, p)
}

func (p Plugs) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implements sql.Scanner for SlotMap, unmarshalling JSONB into the map.
func (sm *SlotMap) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	default:
		return fmt.Errorf("unexpected type %T for SlotMap Scan", src)
	}
	var inter map[string]map[string]bool
	if err := json.Unmarshal(raw, &inter); err != nil {
		return err
	}
	result := make(SlotMap, len(inter))
	for iface, rules := range inter {
		s := &Slot{}
		if v, ok := rules["allow-installation"]; ok {
			s.AllowInstallation = &v
		}
		if v, ok := rules["deny-installation"]; ok {
			s.DenyInstallation = &v
		}
		if v, ok := rules["allow-connection"]; ok {
			s.AllowConnection = &v
		}
		if v, ok := rules["deny-connection"]; ok {
			s.DenyConnection = &v
		}
		if v, ok := rules["allow-auto-connection"]; ok {
			s.AllowAutoConnection = &v
		}
		if v, ok := rules["deny-auto-connection"]; ok {
			s.DenyAutoConnection = &v
		}
		result[iface] = s
	}
	*sm = result
	return nil
}

// Value implements driver.Valuer for SlotMap, marshalling the map to JSONB.
func (sm SlotMap) Value() (driver.Value, error) {
	inter := make(map[string]map[string]bool, len(sm))
	for iface, s := range sm {
		m := make(map[string]bool)
		if s.AllowInstallation != nil {
			m["allow-installation"] = *s.AllowInstallation
		}
		if s.DenyInstallation != nil {
			m["deny-installation"] = *s.DenyInstallation
		}
		if s.AllowConnection != nil {
			m["allow-connection"] = *s.AllowConnection
		}
		if s.DenyConnection != nil {
			m["deny-connection"] = *s.DenyConnection
		}
		if s.AllowAutoConnection != nil {
			m["allow-auto-connection"] = *s.AllowAutoConnection
		}
		if s.DenyAutoConnection != nil {
			m["deny-auto-connection"] = *s.DenyAutoConnection
		}
		inter[iface] = m
	}
	return json.Marshal(inter)
}
