// Package astgen builds Client struct using Go AST.
package astgen

import (
	"go/ast"
	"go/token"

	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
)

// ClientGenerator generates the main Client struct.
type ClientGenerator struct {
	ir *ir.IR
}

// NewClientGenerator creates a new client generator.
func NewClientGenerator(ir *ir.IR) *ClientGenerator {
	return &ClientGenerator{ir: ir}
}

// BuildClient builds the main Client struct and constructor.
func (g *ClientGenerator) BuildClient() []ast.Decl {
	decls := []ast.Decl{}

	// Build Client struct
	clientStruct := g.buildClientStruct()
	decls = append(decls, clientStruct)

	// Build NewClient constructor
	constructor := g.buildConstructor()
	decls = append(decls, constructor)

	return decls
}

// buildClientStruct builds the Client struct type.
func (g *ClientGenerator) buildClientStruct() *ast.GenDecl {
	fields := []*ast.Field{}

	// Add queryService field
	fields = append(fields, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("queryService")},
		Type: &ast.StarExpr{
			X: &ast.SelectorExpr{
				X:   ast.NewIdent("service"),
				Sel: ast.NewIdent("QueryService"),
			},
		},
	})

	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent("Client"),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: fields,
					},
				},
			},
		},
	}
}

// buildConstructor builds the NewClient constructor function.
func (g *ClientGenerator) buildConstructor() *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("NewClient"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("qs")},
						Type: &ast.StarExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("service"),
								Sel: ast.NewIdent("QueryService"),
							},
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.StarExpr{
							X: ast.NewIdent("Client"),
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.AND,
							X: &ast.CompositeLit{
								Type: ast.NewIdent("Client"),
								Elts: []ast.Expr{
									&ast.KeyValueExpr{
										Key:   ast.NewIdent("queryService"),
										Value: ast.NewIdent("qs"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
