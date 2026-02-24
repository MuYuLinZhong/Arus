package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON is a custom type for PostgreSQL JSONB columns.
type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JSON")
	}
	return json.Unmarshal(bytes, j)
}
