// Package astgen builds Include builder types for relation loading using Go AST.
package astgen

import (
	"go/ast"
	"go/token"

	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
)

// IncludeBuilderGenerator generates Include types for eager loading relations.
type IncludeBuilderGenerator struct {
	ir *ir.IR
}

// NewIncludeBuilderGenerator creates a new Include builder generator.
func NewIncludeBuilderGenerator(ir *ir.IR) *IncludeBuilderGenerator {
	return &IncludeBuilderGenerator{ir: ir}
}

// BuildIncludeTypes builds Include struct types and methods for all models.
func (g *IncludeBuilderGenerator) BuildIncludeTypes() []ast.Decl {
	decls := []ast.Decl{}

	for _, model := range g.ir.Models {
		// Skip models with no relations
		hasRelations := false
		for _, field := range model.Fields {
			if field.Relation != nil {
				hasRelations = true
				break
			}
		}
		if !hasRelations {
			continue
		}

		// Generate IncludeXxx struct
		decls = append(decls, g.buildIncludeStruct(model))

		// Generate IncludeXxxField variable
		decls = append(decls, g.buildIncludeVar(model))
	}

	return decls
}

// buildIncludeStruct generates an IncludeXxx struct with relation accessor fields.
func (g *IncludeBuilderGenerator) buildIncludeStruct(model ir.Model) *ast.GenDecl {
	fields := []*ast.Field{}

	for _, field := range model.Fields {
		if field.Relation == nil {
			continue
		}

		// Add relation field accessor
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(field.GoName)},
			Type:  ast.NewIdent(model.Name + field.GoName + "Include"),
		})
	}

	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent("Include" + model.Name),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: fields,
					},
				},
			},
		},
	}
}

// buildIncludeVar generates an IncludeXxxField variable instance.
func (g *IncludeBuilderGenerator) buildIncludeVar(model ir.Model) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent("Include" + model.Name + "Field")},
				Type:  ast.NewIdent("Include" + model.Name),
			},
		},
	}
}

// BuildIncludeRelationTypes builds the relation-specific Include types.
func (g *IncludeBuilderGenerator) BuildIncludeRelationTypes() []ast.Decl {
	decls := []ast.Decl{}

	for _, model := range g.ir.Models {
		for _, field := range model.Fields {
			if field.Relation == nil {
				continue
			}

			// Generate XxxYyyInclude struct (e.g., UserPostsInclude)
			decls = append(decls, g.buildRelationIncludeStruct(model, field))
		}
	}

	return decls
}

// buildRelationIncludeStruct generates a relation-specific Include struct.
func (g *IncludeBuilderGenerator) buildRelationIncludeStruct(model ir.Model, field ir.Field) *ast.GenDecl {
	typeName := model.Name + field.GoName + "Include"

	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(typeName),
				Type: &ast.StructType{
					Fields: &ast.FieldList{},
				},
			},
		},
	}
}

// BuildIncludeMethods builds the Include() methods that return QueryOption.
func (g *IncludeBuilderGenerator) BuildIncludeMethods() []*ast.FuncDecl {
	methods := []*ast.FuncDecl{}

	for _, model := range g.ir.Models {
		for _, field := range model.Fields {
			if field.Relation == nil {
				continue
			}

			typeName := model.Name + field.GoName + "Include"
			relationName := field.Name

			// Build Include() method
			methods = append(methods, g.buildIncludeMethod(typeName, relationName))
		}
	}

	return methods
}

// buildIncludeMethod generates an Include() method for a relation.
func (g *IncludeBuilderGenerator) buildIncludeMethod(typeName, relationName string) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("Include"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("i")},
					Type:  ast.NewIdent(typeName),
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.SelectorExpr{
							X:   ast.NewIdent("service"),
							Sel: ast.NewIdent("QueryOption"),
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("service"),
								Sel: ast.NewIdent("WithInclude"),
							},
							Args: []ast.Expr{
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: `"` + relationName + `"`,
								},
							},
						},
					},
				},
			},
		},
	}
}
