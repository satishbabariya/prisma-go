// Package parsing provides the main API for parsing Prisma schemas.
package parsing

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/core"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
	"github.com/satishbabariya/prisma-go/psl/parsing/lexer"
)

// StringLiteralValue transforms the input string into a valid PSL string literal.
func StringLiteralValue(s string) string {
	// TODO: Implement proper JSON-style string escaping
	// For now, just wrap in quotes and escape basic characters
	result := "\""
	for _, r := range s {
		switch r {
		case '\t':
			result += "\\t"
		case '\n':
			result += "\\n"
		case '"':
			result += "\\\""
		case '\r':
			result += "\\r"
		case '\\':
			result += "\\\\"
		default:
			if r < 32 {
				// Control character - escape as \uXXXX
				result += fmt.Sprintf("\\u%04x", r)
			} else {
				result += string(r)
			}
		}
	}
	result += "\""
	return result
}

// ParseSchema parses a Prisma schema string into an AST.
func ParseSchema(input string) (*ast.SchemaAst, diagnostics.Diagnostics) {
	lex := lexer.NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		diags := diagnostics.NewDiagnostics()
		// TODO: Add proper error handling
		return &ast.SchemaAst{Tops: []ast.Top{}}, diags
	}

	diags := diagnostics.NewDiagnostics()
	parser := ast.NewParser(tokens, &diags)
	astResult := parser.Parse()

	return astResult, diags
}

// ParseSchemaFromSourceFile parses a Prisma schema from a source file.
func ParseSchemaFromSourceFile(file core.SourceFile) (*ast.SchemaAst, diagnostics.Diagnostics) {
	return ParseSchema(file.Data)
}
