// Package schemaast provides a parser for Prisma Schema Language using Participle.
package schema

import (
	"io"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"

	"github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// RawSchema is the raw parse tree structure that matches the grammar.
// This is converted to SchemaAst after parsing.
type RawSchema struct {
	Pos   lexer.Position
	Items []*TopLevelItem `@@*`
}

// TopLevelItem is a union of all possible top-level declarations.
type TopLevelItem struct {
	Pos           lexer.Position
	Model         *ast.Model           `@@`
	Enum          *ast.Enum            `| @@`
	CompositeType *ast.CompositeType   `| @@`
	Datasource    *ast.SourceConfig    `| @@`
	Generator     *ast.GeneratorConfig `| @@`
}

// ToTop converts the item to the Top interface.
func (t *TopLevelItem) ToTop() ast.Top {
	switch {
	case t.Model != nil:
		return t.Model
	case t.Enum != nil:
		return t.Enum
	case t.CompositeType != nil:
		return t.CompositeType
	case t.Datasource != nil:
		return t.Datasource
	case t.Generator != nil:
		return t.Generator
	default:
		return nil
	}
}

// parser is the Participle parser instance.
var parser = participle.MustBuild[RawSchema](
	participle.Lexer(PrismaLexer),
	participle.Elide("Whitespace", "Newline", "Comment", "MultiLineComment"),
	participle.Unquote("String"),
	participle.UseLookahead(10),
	participle.Union[ast.Expression](
		&ast.FunctionCall{},
		&ast.ArrayExpression{},
		&ast.StringValue{},
		&ast.NumericValue{},
		&ast.ConstantValue{},
	),
)

// ParseSchema parses a Prisma schema from an io.Reader.
func ParseSchema(filename string, r io.Reader) (*ast.SchemaAst, error) {
	raw, err := parser.Parse(filename, r)
	if err != nil {
		return nil, err
	}
	return convertRawSchema(raw), nil
}

// ParseSchemaString parses a Prisma schema from a string.
func ParseSchemaString(filename, input string) (*ast.SchemaAst, error) {
	return ParseSchema(filename, strings.NewReader(input))
}

// MustParseSchemaString parses a Prisma schema from a string, panicking on error.
func MustParseSchemaString(filename, input string) *ast.SchemaAst {
	schema, err := ParseSchemaString(filename, input)
	if err != nil {
		panic(err)
	}
	return schema
}

// convertRawSchema converts the raw parse tree to the AST.
func convertRawSchema(raw *RawSchema) *ast.SchemaAst {
	schema := &ast.SchemaAst{
		Tops: make([]ast.Top, 0, len(raw.Items)),
	}
	for _, item := range raw.Items {
		if top := item.ToTop(); top != nil {
			schema.Tops = append(schema.Tops, top)
		}
	}
	return schema
}
