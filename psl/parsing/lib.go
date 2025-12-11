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
// Implements proper JSON-style string escaping as per RFC 7159.
func StringLiteralValue(s string) string {
	result := "\""
	for _, r := range s {
		switch r {
		case '\b':
			result += "\\b"
		case '\f':
			result += "\\f"
		case '\n':
			result += "\\n"
		case '\r':
			result += "\\r"
		case '\t':
			result += "\\t"
		case '"':
			result += "\\\""
		case '\\':
			result += "\\\\"
		default:
			if r < 32 {
				// Control character - escape as \uXXXX (4-digit hex)
				result += fmt.Sprintf("\\u%04x", r)
			} else if r > 0xFFFF {
				// Surrogate pair for characters > U+FFFF
				// Encode as two \uXXXX sequences
				r -= 0x10000
				high := 0xD800 + (r >> 10)
				low := 0xDC00 + (r & 0x3FF)
				result += fmt.Sprintf("\\u%04x\\u%04x", high, low)
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
	diags := diagnostics.NewDiagnostics()
	
	lex := lexer.NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		// Report lexer error with proper span information
		// Create a span at the beginning of the file for lexer errors
		span := diagnostics.NewSpan(0, len(input), diagnostics.FileIDZero)
		diags.PushError(diagnostics.NewDatamodelError(
			fmt.Sprintf("Lexer error: %v", err),
			span,
		))
		return &ast.SchemaAst{Tops: []ast.Top{}}, diags
	}

	parser := ast.NewParser(tokens, &diags)
	astResult := parser.Parse()

	return astResult, diags
}

// ParseSchemaFromSourceFile parses a Prisma schema from a source file.
func ParseSchemaFromSourceFile(file core.SourceFile) (*ast.SchemaAst, diagnostics.Diagnostics) {
	return ParseSchema(file.Data)
}
