// Package psl provides the main API for working with Prisma Schema Language.
package psl

import (
	"strings"

	"github.com/satishbabariya/prisma-go/psl/core"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/formatting"
	parser "github.com/satishbabariya/prisma-go/psl/parsing/v2"
	"github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
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
	// Adapt V2 parser (returns error) to legacy signature (returns Diagnostics)
	schema, err := parser.ParseSchema("schema.prisma", strings.NewReader(input))
	var diags diagnostics.Diagnostics
	if err != nil {
		// Create a simple error diagnostic
		diags.PushError(diagnostics.NewDatamodelError(err.Error(), diagnostics.Span{}))
	}
	return schema, diags
}

// ParseSchemaFromFile parses a Prisma schema from a source file.
func ParseSchemaFromFile(file core.SourceFile) (*ast.SchemaAst, diagnostics.Diagnostics) {
	schema, err := parser.ParseSchema(file.Path, strings.NewReader(file.Data))
	var diags diagnostics.Diagnostics
	if err != nil {
		diags.PushError(diagnostics.NewDatamodelError(err.Error(), diagnostics.Span{}))
	}
	return schema, diags
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
