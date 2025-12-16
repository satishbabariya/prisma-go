// Package astgen builds query builder methods using Go AST.
package astgen

import (
	"go/ast"
	"go/token"

	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
)

// QueryBuilderGenerator generates query builder methods.
type QueryBuilderGenerator struct {
	ir *ir.IR
}

// NewQueryBuilderGenerator creates a new query builder generator.
func NewQueryBuilderGenerator(ir *ir.IR) *QueryBuilderGenerator {
	return &QueryBuilderGenerator{ir: ir}
}

// BuildQueryMethods builds query builder methods for a model.
func (g *QueryBuilderGenerator) BuildQueryMethods(model ir.Model) []*ast.FuncDecl {
	methods := []*ast.FuncDecl{}

	// Generate FindMany method
	methods = append(methods, g.buildFindMany(model))

	// Generate FindFirst method
	methods = append(methods, g.buildFindFirst(model))

	// Generate FindUnique method
	methods = append(methods, g.buildFindUnique(model))

	// Generate Count method
	methods = append(methods, g.buildCount(model))

	// Generate Delete method
	methods = append(methods, g.buildDelete(model))

	// Generate CreateMany method
	methods = append(methods, g.buildCreateMany(model))

	// Generate Upsert method
	methods = append(methods, g.buildUpsert(model))

	return methods
}

// buildFindMany generates a FindMany method.
func (g *QueryBuilderGenerator) buildFindMany(model ir.Model) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("FindMany" + model.Name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("ctx")},
						Type: &ast.SelectorExpr{
							X:   ast.NewIdent("context"),
							Sel: ast.NewIdent("Context"),
						},
					},
					{
						Names: []*ast.Ident{ast.NewIdent("opts")},
						Type: &ast.Ellipsis{
							Elt: ast.NewIdent("QueryOption"),
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.ArrayType{
							Elt: ast.NewIdent(model.Name),
						},
					},
					{
						Type: ast.NewIdent("error"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// var result []Model
				&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{ast.NewIdent("result")},
								Type: &ast.ArrayType{
									Elt: ast.NewIdent(model.Name),
								},
							},
						},
					},
				},
				// err := queryService.FindManyInto(ctx, "Model", &result, opts...)
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("err")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("queryService"),
								Sel: ast.NewIdent("FindManyInto"),
							},
							Args: []ast.Expr{
								ast.NewIdent("ctx"),
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + model.TableName + `"`,
								},
								&ast.UnaryExpr{
									Op: token.AND,
									X:  ast.NewIdent("result"),
								},
								&ast.Ellipsis{
									Elt: ast.NewIdent("opts"),
								},
							},
						},
					},
				},
				// return result, err
				&ast.ReturnStmt{
					Results: []ast.Expr{
						ast.NewIdent("result"),
						ast.NewIdent("err"),
					},
				},
			},
		},
	}
}

// buildFindFirst generates a FindFirst method.
func (g *QueryBuilderGenerator) buildFindFirst(model ir.Model) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("FindFirst" + model.Name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("ctx")},
						Type: &ast.SelectorExpr{
							X:   ast.NewIdent("context"),
							Sel: ast.NewIdent("Context"),
						},
					},
					{
						Names: []*ast.Ident{ast.NewIdent("opts")},
						Type: &ast.Ellipsis{
							Elt: ast.NewIdent("QueryOption"),
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.StarExpr{
							X: ast.NewIdent(model.Name),
						},
					},
					{
						Type: ast.NewIdent("error"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// var result Model
				&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{ast.NewIdent("result")},
								Type:  ast.NewIdent(model.Name),
							},
						},
					},
				},
				// err := queryService.FindFirstInto(ctx, "Model", &result, opts...)
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("err")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("queryService"),
								Sel: ast.NewIdent("FindFirstInto"),
							},
							Args: []ast.Expr{
								ast.NewIdent("ctx"),
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + model.TableName + `"`,
								},
								&ast.UnaryExpr{
									Op: token.AND,
									X:  ast.NewIdent("result"),
								},
								&ast.Ellipsis{
									Elt: ast.NewIdent("opts"),
								},
							},
						},
					},
				},
				// return &result, err
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.AND,
							X:  ast.NewIdent("result"),
						},
						ast.NewIdent("err"),
					},
				},
			},
		},
	}
}

// buildFindUnique generates a FindUnique method.
func (g *QueryBuilderGenerator) buildFindUnique(model ir.Model) *ast.FuncDecl {
	// Same as FindFirst for now
	return g.buildFindFirst(model)
}

// buildCount generates a Count method.
func (g *QueryBuilderGenerator) buildCount(model ir.Model) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("Count" + model.Name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("ctx")},
						Type: &ast.SelectorExpr{
							X:   ast.NewIdent("context"),
							Sel: ast.NewIdent("Context"),
						},
					},
					{
						Names: []*ast.Ident{ast.NewIdent("opts")},
						Type: &ast.Ellipsis{
							Elt: ast.NewIdent("QueryOption"),
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: ast.NewIdent("int64"),
					},
					{
						Type: ast.NewIdent("error"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// return queryService.Count(ctx, "Model", opts...)
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("queryService"),
								Sel: ast.NewIdent("Count"),
							},
							Args: []ast.Expr{
								ast.NewIdent("ctx"),
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + model.TableName + `"`,
								},
								&ast.Ellipsis{
									Elt: ast.NewIdent("opts"),
								},
							},
						},
					},
				},
			},
		},
	}
}

// buildDelete generates a Delete method.
func (g *QueryBuilderGenerator) buildDelete(model ir.Model) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("Delete" + model.Name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("ctx")},
						Type: &ast.SelectorExpr{
							X:   ast.NewIdent("context"),
							Sel: ast.NewIdent("Context"),
						},
					},
					{
						Names: []*ast.Ident{ast.NewIdent("opts")},
						Type: &ast.Ellipsis{
							Elt: ast.NewIdent("QueryOption"),
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: ast.NewIdent("int64"),
					},
					{
						Type: ast.NewIdent("error"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// return queryService.DeleteMany(ctx, "Model", opts...)
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("queryService"),
								Sel: ast.NewIdent("DeleteMany"),
							},
							Args: []ast.Expr{
								ast.NewIdent("ctx"),
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + model.TableName + `"`,
								},
								&ast.Ellipsis{
									Elt: ast.NewIdent("opts"),
								},
							},
						},
					},
				},
			},
		},
	}
}
