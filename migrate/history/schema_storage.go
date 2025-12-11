// Package history provides schema storage functionality for migration history.
package history

import (
	"encoding/json"
	"fmt"

	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// SerializeSchema serializes a DatabaseSchema to JSON
func SerializeSchema(schema *introspect.DatabaseSchema) (string, error) {
	if schema == nil {
		return "", nil
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("failed to serialize schema: %w", err)
	}
	return string(data), nil
}

// DeserializeSchema deserializes a JSON string to DatabaseSchema
func DeserializeSchema(jsonStr string) (*introspect.DatabaseSchema, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var schema introspect.DatabaseSchema
	if err := json.Unmarshal([]byte(jsonStr), &schema); err != nil {
		return nil, fmt.Errorf("failed to deserialize schema: %w", err)
	}
	return &schema, nil
}
