// Package compiler compiles query AST into SQL.
package compiler

import (
	"github.com/satishbabariya/prisma-go/query/ast"
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// Compiler compiles query AST into SQL
type Compiler struct {
	generator sqlgen.Generator
}

// NewCompiler creates a new query compiler
func NewCompiler(provider string) *Compiler {
	return &Compiler{
		generator: sqlgen.NewGenerator(provider),
	}
}

// Compile compiles a query node into SQL
func (c *Compiler) Compile(node ast.QueryNode) (string, []interface{}, error) {
	switch node.Type() {
	case ast.NodeTypeFindMany:
		return c.compileFindMany(node.(*ast.FindManyQuery))
	default:
		return "", nil, ErrUnsupportedQuery
	}
}

func (c *Compiler) compileFindMany(query *ast.FindManyQuery) (string, []interface{}, error) {
	// TODO: Implement findMany compilation
	return "", []interface{}{}, nil
}

