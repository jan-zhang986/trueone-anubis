package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type BitBool bool

func (b BitBool) Value() (driver.Value, error) {
	if b {
		return []byte{1}, nil
	}
	return []byte{0}, nil
}

func (b *BitBool) Scan(val interface{}) error {
	if val == nil {
		*b = false
		return nil
	}
	switch v := val.(type) {
	case []byte:
		*b = len(v) > 0 && (v[0] == 1 || v[0] == '1')
	case int64:
		*b = v == 1
	case bool:
		*b = BitBool(v)
	default:
		return fmt.Errorf("cannot scan %T into BitBool", val)
	}
	return nil
}

func (b BitBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(b))
}

func (b *BitBool) UnmarshalJSON(data []byte) error {
	var val bool
	if err := json.Unmarshal(data, &val); err == nil {
		*b = BitBool(val)
		return nil
	}
	var num int
	if err := json.Unmarshal(data, &num); err == nil {
		*b = num == 1
		return nil
	}
	return nil
}
