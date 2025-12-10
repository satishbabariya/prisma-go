// Package pls provides the main API for working with Prisma Schema Language.
package pls

import (
	"github.com/satishbabariya/prisma-go/psl/core"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/formatting"
	"github.com/satishbabariya/prisma-go/psl/parsing"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// Re-export key types for convenience
type (
	SourceFile    = core.SourceFile
	Configuration = core.Configuration
	Datasource    = core.Datasource
	Generator     = core.Generator
	Diagnostics   = diagnostics.Diagnostics
	SchemaAst     = ast.SchemaAst
)

// ParseSchema parses a Prisma schema string and returns the AST and diagnostics.
func ParseSchema(input string) (*ast.SchemaAst, diagnostics.Diagnostics) {
	return parsing.ParseSchema(input)
}

// ParseSchemaFromFile parses a Prisma schema from a source file.
func ParseSchemaFromFile(file core.SourceFile) (*ast.SchemaAst, diagnostics.Diagnostics) {
	return parsing.ParseSchemaFromSourceFile(file)
}

// Reformat reformats a Prisma schema string.
// Returns the reformatted schema, or an error if the schema cannot be parsed.
func Reformat(source string, indentWidth int) (string, error) {
	return formatting.Reformat(source, indentWidth)
}

// NewSourceFile creates a new source file.
func NewSourceFile(path, data string) core.SourceFile {
	return core.NewSourceFile(path, data)
}
