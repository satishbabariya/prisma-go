// Package astgen builds query builder methods using Go AST.
package astgen

import (
	"go/ast"

	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
)

// buildFindFirstOrThrow generates a FindFirstOrThrow method that throws error if not found.
func (g *QueryBuilderGenerator) buildFindFirstOrThrow(model ir.Model) *ast.FuncDecl {
	// Start with FindFirst as base
	decl := g.buildFindFirst(model)

	// Change the method name
	decl.Name = ast.NewIdent("FindFirstOrThrow" + model.Name)

	// Change the service call from FindFirstInto to FindFirstIntoOrThrow
	// The body is a BlockStmt with three statements: var result, call, return
	// We need to modify the call statement (index 1) to switch the method call
	callStmt := decl.Body.List[1].(*ast.AssignStmt)
	callExpr := callStmt.Rhs[0].(*ast.CallExpr)
	selectorExpr := callExpr.Fun.(*ast.SelectorExpr)
	selectorExpr.Sel = ast.NewIdent("FindFirstIntoOrThrow")

	return decl
}

// buildFindUniqueOrThrow generates a FindUniqueOrThrow method that throws error if not found.
func (g *QueryBuilderGenerator) buildFindUniqueOrThrow(model ir.Model) *ast.FuncDecl {
	// Start with FindUnique as base
	decl := g.buildFindUnique(model)

	// Change the method name
	decl.Name = ast.NewIdent("FindUniqueOrThrow" + model.Name)

	// Change the service call from FindFirstInto to FindFirstIntoOrThrow
	// Same logic as FindFirstOrThrow
	callStmt := decl.Body.List[1].(*ast.AssignStmt)
	callExpr := callStmt.Rhs[0].(*ast.CallExpr)
	selectorExpr := callExpr.Fun.(*ast.SelectorExpr)
	selectorExpr.Sel = ast.NewIdent("FindFirstIntoOrThrow")

	return decl
}
