// Package pslcore provides MCF (Model Configuration Format) support.
package validation

import (
	"encoding/json"

	"github.com/satishbabariya/prisma-go/psl/database"
)

// SerializableMCF represents the serializable MCF structure.
type SerializableMCF struct {
	Generators  json.RawMessage `json:"generators"`
	Datasources json.RawMessage `json:"datasources"`
	Warnings    []string        `json:"warnings"`
}

// GetConfig converts a Configuration to MCF JSON format.
func GetConfig(config Configuration, files database.Files) (json.RawMessage, error) {
	generatorsJSON, err := GeneratorsToJSONValue(config.Generators, files)
	if err != nil {
		return nil, err
	}

	datasourcesJSON, err := RenderSourcesToJSONValue(config.Datasources, files)
	if err != nil {
		return nil, err
	}

	// Convert warnings to string messages
	warnings := make([]string, len(config.Warnings))
	for i, warn := range config.Warnings {
		warnings[i] = warn.Message()
	}

	mcf := SerializableMCF{
		Generators:  generatorsJSON,
		Datasources: datasourcesJSON,
		Warnings:    warnings,
	}

	return json.Marshal(mcf)
}
