package domain

import (
	"database/sql/driver"
	"encoding/json"
)

type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return nil
	}

	if len(data) == 0 || string(data) == "null" {
		*s = nil
		return nil
	}

	return json.Unmarshal(data, s)
}
