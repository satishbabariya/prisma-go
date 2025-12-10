// Package pslcore provides MCF datasource JSON conversion.
package validation

import (
	"encoding/json"

	"github.com/satishbabariya/prisma-go/psl/database"
)

// SourceConfig represents a datasource configuration in MCF format.
type SourceConfig struct {
	Name           string   `json:"name"`
	Provider       string   `json:"provider"`
	ActiveProvider string   `json:"activeProvider"`
	Schemas        []string `json:"schemas"`
	Documentation  *string  `json:"documentation,omitempty"`
	SourceFilePath string   `json:"sourceFilePath"`
}

// RenderSourcesToJSONValue converts datasources to JSON value.
func RenderSourcesToJSONValue(sources []Datasource, files database.Files) (json.RawMessage, error) {
	configs := make([]SourceConfig, len(sources))

	for i, source := range sources {
		// Find source file path for datasource using span
		sourceFilePath := ""
		fileEntry := files.Get(source.Span.FileID)
		if fileEntry != nil {
			sourceFilePath = fileEntry.Name
		}

		configs[i] = SourceConfig{
			Name:           source.Name,
			Provider:       source.Provider,
			ActiveProvider: source.ActiveProvider,
			Schemas:        source.Schemas,
			Documentation:  source.Documentation,
			SourceFilePath: sourceFilePath,
		}
	}

	return json.Marshal(configs)
}

// RenderSourcesToJSON converts datasources to pretty JSON string.
func RenderSourcesToJSON(sources []Datasource, files database.Files) (string, error) {
	configs := make([]SourceConfig, len(sources))

	for i, source := range sources {
		// Find source file path for datasource using span
		sourceFilePath := ""
		fileEntry := files.Get(source.Span.FileID)
		if fileEntry != nil {
			sourceFilePath = fileEntry.Name
		}

		configs[i] = SourceConfig{
			Name:           source.Name,
			Provider:       source.Provider,
			ActiveProvider: source.ActiveProvider,
			Schemas:        source.Schemas,
			Documentation:  source.Documentation,
			SourceFilePath: sourceFilePath,
		}
	}

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
