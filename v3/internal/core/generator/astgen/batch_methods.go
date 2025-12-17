// Package astgen builds batch operation methods using Go AST.
package astgen

import (
	"go/ast"
	"go/token"

	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
)

// buildCreateMany generates a CreateMany method for batch inserts.
func (g *QueryBuilderGenerator) buildCreateMany(model ir.Model) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("CreateMany" + model.Name),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("client")},
					Type: &ast.StarExpr{
						X: ast.NewIdent("Client"),
					},
				},
			},
		},
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
						Names: []*ast.Ident{ast.NewIdent("data")},
						Type: &ast.ArrayType{
							Elt: &ast.MapType{
								Key:   ast.NewIdent("string"),
								Value: &ast.InterfaceType{Methods: &ast.FieldList{}},
							},
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.ArrayType{
							Elt: &ast.MapType{
								Key:   ast.NewIdent("string"),
								Value: &ast.InterfaceType{Methods: &ast.FieldList{}},
							},
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
				// return queryService.CreateMany(ctx, "tableName", data)
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("client"),
									Sel: ast.NewIdent("queryService"),
								},
								Sel: ast.NewIdent("CreateMany"),
							},
							Args: []ast.Expr{
								ast.NewIdent("ctx"),
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + model.TableName + `"`,
								},
								ast.NewIdent("data"),
							},
						},
					},
				},
			},
		},
	}
}

// buildUpsert generates an Upsert method.
func (g *QueryBuilderGenerator) buildUpsert(model ir.Model) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("Upsert" + model.Name),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("client")},
					Type: &ast.StarExpr{
						X: ast.NewIdent("Client"),
					},
				},
			},
		},
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
						Names: []*ast.Ident{ast.NewIdent("data")},
						Type: &ast.MapType{
							Key:   ast.NewIdent("string"),
							Value: &ast.InterfaceType{Methods: &ast.FieldList{}},
						},
					},
					{
						Names: []*ast.Ident{ast.NewIdent("updateData")},
						Type: &ast.MapType{
							Key:   ast.NewIdent("string"),
							Value: &ast.InterfaceType{Methods: &ast.FieldList{}},
						},
					},
					{
						Names: []*ast.Ident{ast.NewIdent("conflictKeys")},
						Type: &ast.ArrayType{
							Elt: ast.NewIdent("string"),
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.MapType{
							Key:   ast.NewIdent("string"),
							Value: &ast.InterfaceType{Methods: &ast.FieldList{}},
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
				// return queryService.Upsert(ctx, "tableName", data, updateData, conflictKeys)
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("client"),
									Sel: ast.NewIdent("queryService"),
								},
								Sel: ast.NewIdent("Upsert"),
							},
							Args: []ast.Expr{
								ast.NewIdent("ctx"),
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + model.TableName + `"`,
								},
								ast.NewIdent("data"),
								ast.NewIdent("updateData"),
								ast.NewIdent("conflictKeys"),
							},
						},
					},
				},
			},
		},
	}
}
