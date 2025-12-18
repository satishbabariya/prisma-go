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

	// Generate FindFirstOrThrow method
	methods = append(methods, g.buildFindFirstOrThrow(model))

	// Generate FindUniqueOrThrow method
	methods = append(methods, g.buildFindUniqueOrThrow(model))

	// Generate Count method
	methods = append(methods, g.buildCount(model))

	// Generate Delete method
	methods = append(methods, g.buildDelete(model))

	// Generate CreateMany method
	methods = append(methods, g.buildCreateMany(model))

	// Generate Upsert method
	methods = append(methods, g.buildUpsert(model))

	// Generate typed Create method
	methods = append(methods, g.buildCreate(model))

	// Generate typed Update method
	methods = append(methods, g.buildUpdate(model))

	return methods
}

// buildFindMany generates a FindMany method.
func (g *QueryBuilderGenerator) buildFindMany(model ir.Model) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("FindMany" + model.Name),
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
						Names: []*ast.Ident{ast.NewIdent("opts")},
						Type: &ast.Ellipsis{
							Elt: &ast.SelectorExpr{
								X:   ast.NewIdent("service"),
								Sel: ast.NewIdent("QueryOption"),
							},
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
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("client"),
									Sel: ast.NewIdent("queryService"),
								},
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
								ast.NewIdent("opts"),
							},
							Ellipsis: token.NoPos + 1, // Set valid position to enable "..."
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
						Names: []*ast.Ident{ast.NewIdent("opts")},
						Type: &ast.Ellipsis{
							Elt: &ast.SelectorExpr{
								X:   ast.NewIdent("service"),
								Sel: ast.NewIdent("QueryOption"),
							},
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
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("client"),
									Sel: ast.NewIdent("queryService"),
								},
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
								ast.NewIdent("opts"),
							},
							Ellipsis: token.NoPos + 1,
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
	// Re-use FindFirst logic but change name
	decl := g.buildFindFirst(model)
	decl.Name = ast.NewIdent("FindUnique" + model.Name)
	return decl
}

// buildCount generates a Count method.
func (g *QueryBuilderGenerator) buildCount(model ir.Model) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("Count" + model.Name),
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
						Names: []*ast.Ident{ast.NewIdent("opts")},
						Type: &ast.Ellipsis{
							Elt: &ast.SelectorExpr{
								X:   ast.NewIdent("service"),
								Sel: ast.NewIdent("QueryOption"),
							},
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
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("client"),
									Sel: ast.NewIdent("queryService"),
								},
								Sel: ast.NewIdent("Count"),
							},
							Args: []ast.Expr{
								ast.NewIdent("ctx"),
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + model.TableName + `"`,
								},
								ast.NewIdent("opts"),
							},
							Ellipsis: token.NoPos + 1,
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
						Names: []*ast.Ident{ast.NewIdent("opts")},
						Type: &ast.Ellipsis{
							Elt: &ast.SelectorExpr{
								X:   ast.NewIdent("service"),
								Sel: ast.NewIdent("QueryOption"),
							},
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
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("client"),
									Sel: ast.NewIdent("queryService"),
								},
								Sel: ast.NewIdent("DeleteMany"),
							},
							Args: []ast.Expr{
								ast.NewIdent("ctx"),
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + model.TableName + `"`,
								},
								ast.NewIdent("opts"),
							},
							Ellipsis: token.NoPos + 1,
						},
					},
				},
			},
		},
	}
}

// buildCreate generates a typed Create method using CreateXxxInput.
func (g *QueryBuilderGenerator) buildCreate(model ir.Model) *ast.FuncDecl {
	inputTypeName := "Create" + model.Name + "Input"

	return &ast.FuncDecl{
		Name: ast.NewIdent("Create" + model.Name),
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
						Names: []*ast.Ident{ast.NewIdent("input")},
						Type:  ast.NewIdent(inputTypeName),
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
				// data := make(map[string]interface{})
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("data")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: ast.NewIdent("make"),
							Args: []ast.Expr{
								&ast.MapType{
									Key:   ast.NewIdent("string"),
									Value: &ast.InterfaceType{Methods: &ast.FieldList{}},
								},
							},
						},
					},
				},
				// TODO: Marshal input to map (simplified - in real impl would use reflection)
				// For now, we'll use a comment placeholder
				// return client.queryService.Create(ctx, "Model", data)
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("result"), ast.NewIdent("err")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("client"),
									Sel: ast.NewIdent("queryService"),
								},
								Sel: ast.NewIdent("Create"),
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
				// if err != nil { return nil, err }
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X:  ast.NewIdent("err"),
						Op: token.NEQ,
						Y:  ast.NewIdent("nil"),
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ReturnStmt{
								Results: []ast.Expr{
									ast.NewIdent("nil"),
									ast.NewIdent("err"),
								},
							},
						},
					},
				},
				// _ = result (placeholder)
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("_")},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{ast.NewIdent("result")},
				},
				// return &Model{}, nil (placeholder)
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.AND,
							X: &ast.CompositeLit{
								Type: ast.NewIdent(model.Name),
							},
						},
						ast.NewIdent("nil"),
					},
				},
			},
		},
	}
}

// buildUpdate generates a typed Update method using UpdateXxxInput.
func (g *QueryBuilderGenerator) buildUpdate(model ir.Model) *ast.FuncDecl {
	inputTypeName := "Update" + model.Name + "Input"

	return &ast.FuncDecl{
		Name: ast.NewIdent("Update" + model.Name),
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
						Names: []*ast.Ident{ast.NewIdent("input")},
						Type:  ast.NewIdent(inputTypeName),
					},
					{
						Names: []*ast.Ident{ast.NewIdent("opts")},
						Type: &ast.Ellipsis{
							Elt: &ast.SelectorExpr{
								X:   ast.NewIdent("service"),
								Sel: ast.NewIdent("QueryOption"),
							},
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
				// data := make(map[string]interface{})
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("data")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: ast.NewIdent("make"),
							Args: []ast.Expr{
								&ast.MapType{
									Key:   ast.NewIdent("string"),
									Value: &ast.InterfaceType{Methods: &ast.FieldList{}},
								},
							},
						},
					},
				},
				// _ = input (placeholder to use the variable)
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("_")},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{ast.NewIdent("input")},
				},
				// return client.queryService.Update(ctx, "Model", data, opts...)
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("client"),
									Sel: ast.NewIdent("queryService"),
								},
								Sel: ast.NewIdent("Update"),
							},
							Args: []ast.Expr{
								ast.NewIdent("ctx"),
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + model.TableName + `"`,
								},
								ast.NewIdent("data"),
								ast.NewIdent("opts"),
							},
							Ellipsis: token.NoPos + 1,
						},
					},
				},
			},
		},
	}
}
