package schema

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// PrismaLexer defines the token types for Prisma Schema Language.
var PrismaLexer = lexer.MustSimple([]lexer.SimpleRule{
	// Keywords
	{Name: "Keyword", Pattern: `\b(model|enum|type|view|datasource|generator|extend|Unsupported)\b`},

	// Block attribute prefix (must come before single @)
	{Name: "BlockAttr", Pattern: `@@`},
	// Field attribute prefix
	{Name: "FieldAttr", Pattern: `@`},

	// Punctuation
	{Name: "LBrace", Pattern: `\{`},
	{Name: "RBrace", Pattern: `\}`},
	{Name: "LParen", Pattern: `\(`},
	{Name: "RParen", Pattern: `\)`},
	{Name: "LBracket", Pattern: `\[`},
	{Name: "RBracket", Pattern: `\]`},
	{Name: "Colon", Pattern: `:`},
	{Name: "Comma", Pattern: `,`},
	{Name: "Dot", Pattern: `\.`},
	{Name: "Equal", Pattern: `=`},
	{Name: "Question", Pattern: `\?`},
	{Name: "Exclaim", Pattern: `!`},

	// Literals
	{Name: "String", Pattern: `"(?:\\.|[^"\\])*"`},
	{Name: "Number", Pattern: `-?\d+(?:\.\d+)?`},

	// Identifiers (Unicode alphanumeric with _ and -)
	{Name: "Ident", Pattern: `[\p{L}\p{N}][\p{L}\p{N}_-]*`},

	// Comments (doc comments first, then regular)
	{Name: "DocComment", Pattern: `///[^\n]*`},
	{Name: "Comment", Pattern: `//[^\n]*`},
	{Name: "MultiLineComment", Pattern: `/\*(?:[^*]|\*[^/])*\*/`},

	// Whitespace and newlines
	{Name: "Newline", Pattern: `[\r\n]+`},
	{Name: "Whitespace", Pattern: `[ \t]+`},
})
