package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// NOTE: this file contains the parsing functions to convert the JSONB in the DB to a valid Go struct.
// This can be extended later on to support more complex parsing if needed.

func (p *Plugs) Scan(value interface{}) error {
	if value == nil {
		*p = Plugs{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan Plugs: expected []byte, got %T", value)
	}

	if len(bytes) == 0 {
		*p = Plugs{}
		return nil
	}

	return json.Unmarshal(bytes, p)
}

func (p Plugs) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (s *Slots) Scan(value interface{}) error {
	if value == nil {
		*s = Slots{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan Slots: expected []byte, got %T", value)
	}

	if len(bytes) == 0 {
		*s = Slots{}
		return nil
	}

	return json.Unmarshal(bytes, s)
}

func (s Slots) Value() (driver.Value, error) {
	return json.Marshal(s)
}
