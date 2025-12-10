// Package pslcore provides MCF generator JSON conversion.
package validation

import (
	"encoding/json"

	"github.com/satishbabariya/prisma-go/psl/database"
)

// ExtendedGenerator represents a generator with source file path.
type ExtendedGenerator struct {
	Generator      Generator `json:",inline"`
	SourceFilePath string    `json:"sourceFilePath"`
}

// GeneratorsToJSONValue converts generators to JSON value.
func GeneratorsToJSONValue(generators []Generator, files database.Files) (json.RawMessage, error) {
	extended := make([]ExtendedGenerator, len(generators))

	for i, gen := range generators {
		// Find source file path for generator using span
		sourceFilePath := ""
		fileEntry := files.Get(gen.Span.FileID)
		if fileEntry != nil {
			sourceFilePath = fileEntry.Name
		}

		extended[i] = ExtendedGenerator{
			Generator:      gen,
			SourceFilePath: sourceFilePath,
		}
	}

	return json.Marshal(extended)
}

// GeneratorsToJSON converts generators to pretty JSON string.
func GeneratorsToJSON(generators []Generator) (string, error) {
	data, err := json.MarshalIndent(generators, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
