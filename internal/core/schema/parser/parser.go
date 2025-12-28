// Package parser implements the Prisma schema parser.
package parser

import (
	"context"
	"fmt"
	"os"
	"strings"

	pslparser "github.com/satishbabariya/prisma-go/psl/parsing/v2"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// Parser implements the SchemaParser interface.
type Parser struct {
	adapter *ASTToDomain
}

// NewParser creates a new schema parser.
func NewParser() *Parser {
	return &Parser{
		adapter: NewASTToDomain(),
	}
}

// Parse parses schema content from a string.
func (p *Parser) Parse(ctx context.Context, content string) (*domain.Schema, error) {
	// Use the existing PSL v2 parser
	pslSchema, err := pslparser.ParseSchema("schema.prisma", strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Convert PSL AST to v3 domain model
	schema := p.adapter.ConvertSchema(pslSchema)

	return schema, nil
}

// ParseFile parses schema from a file.
func (p *Parser) ParseFile(ctx context.Context, path string) (*domain.Schema, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	return p.Parse(ctx, string(content))
}

// Ensure Parser implements SchemaParser interface.
var _ domain.SchemaParser = (*Parser)(nil)
