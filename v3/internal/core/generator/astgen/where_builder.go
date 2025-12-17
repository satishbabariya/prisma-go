// Package astgen builds fluent Where builders using Go AST.
package astgen

import (
	"go/ast"
	"go/token"

	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
)

// WhereBuilderGenerator generates Where builder types.
type WhereBuilderGenerator struct {
	ir *ir.IR
}

// NewWhereBuilderGenerator creates a new Where builder generator.
func NewWhereBuilderGenerator(ir *ir.IR) *WhereBuilderGenerator {
	return &WhereBuilderGenerator{ir: ir}
}

// BuildWhereTypes builds Where struct types for all models.
func (g *WhereBuilderGenerator) BuildWhereTypes() []ast.Decl {
	decls := []ast.Decl{}

	for _, model := range g.ir.Models {
		// Generate WhereXxx struct
		decls = append(decls, g.buildWhereStruct(model))

		// Generate field condition types
		for _, field := range model.Fields {
			if field.Relation != nil {
				continue // Skip relation fields
			}
			decls = append(decls, g.buildFieldConditionStruct(model, field))
		}

		// Generate WhereXxx variable with field accessors
		decls = append(decls, g.buildWhereVar(model))
	}

	return decls
}

// buildWhereStruct generates a WhereXxx struct type.
func (g *WhereBuilderGenerator) buildWhereStruct(model ir.Model) *ast.GenDecl {
	fields := []*ast.Field{}

	for _, field := range model.Fields {
		if field.Relation != nil {
			continue
		}

		// Add field accessor
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(field.GoName)},
			Type:  ast.NewIdent(model.Name + field.GoName + "Condition"),
		})
	}

	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent("Where" + model.Name),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: fields,
					},
				},
			},
		},
	}
}

// buildFieldConditionStruct generates a condition struct for a field.
func (g *WhereBuilderGenerator) buildFieldConditionStruct(model ir.Model, field ir.Field) *ast.GenDecl {
	typeName := model.Name + field.GoName + "Condition"

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

// buildWhereVar generates a WhereXxx variable instance.
func (g *WhereBuilderGenerator) buildWhereVar(model ir.Model) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent("Where" + model.Name + "Field")},
				Type:  ast.NewIdent("Where" + model.Name),
			},
		},
	}
}

// BuildConditionMethods builds methods for condition types.
func (g *WhereBuilderGenerator) BuildConditionMethods() []*ast.FuncDecl {
	methods := []*ast.FuncDecl{}

	for _, model := range g.ir.Models {
		for _, field := range model.Fields {
			if field.Relation != nil {
				continue
			}

			typeName := model.Name + field.GoName + "Condition"
			fieldGoType := g.getGoType(field)

			// Equals method
			methods = append(methods, g.buildEqualsMethod(typeName, field, fieldGoType))

			// NotEquals method
			methods = append(methods, g.buildNotEqualsMethod(typeName, field, fieldGoType))

			// For string fields, add Contains
			if field.Type.PrismaType == "String" {
				methods = append(methods, g.buildContainsMethod(typeName, field))
				methods = append(methods, g.buildStartsWithMethod(typeName, field))
				methods = append(methods, g.buildEndsWithMethod(typeName, field))
			}

			// For numeric fields, add comparison methods
			if isNumericType(field.Type.PrismaType) {
				methods = append(methods, g.buildGtMethod(typeName, field, fieldGoType))
				methods = append(methods, g.buildGteMethod(typeName, field, fieldGoType))
				methods = append(methods, g.buildLtMethod(typeName, field, fieldGoType))
				methods = append(methods, g.buildLteMethod(typeName, field, fieldGoType))
			}
		}
	}

	return methods
}

func isNumericType(prismaType string) bool {
	switch prismaType {
	case "Int", "BigInt", "Float", "Decimal":
		return true
	}
	return false
}

// buildEqualsMethod generates an Equals method.
func (g *WhereBuilderGenerator) buildEqualsMethod(typeName string, field ir.Field, goType string) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("Equals"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("c")},
					Type:  ast.NewIdent(typeName),
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("value")},
						Type:  ast.NewIdent(goType),
					},
				},
			},
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
								Sel: ast.NewIdent("WithWhere"),
							},
							Args: []ast.Expr{
								&ast.CompositeLit{
									Type: &ast.SelectorExpr{
										X:   ast.NewIdent("domain"),
										Sel: ast.NewIdent("Condition"),
									},
									Elts: []ast.Expr{
										&ast.KeyValueExpr{
											Key:   ast.NewIdent("Field"),
											Value: &ast.BasicLit{Kind: token.STRING, Value: `"` + field.Name + `"`},
										},
										&ast.KeyValueExpr{
											Key:   ast.NewIdent("Operator"),
											Value: &ast.SelectorExpr{X: ast.NewIdent("domain"), Sel: ast.NewIdent("Equals")},
										},
										&ast.KeyValueExpr{
											Key:   ast.NewIdent("Value"),
											Value: ast.NewIdent("value"),
										},
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

// buildNotEqualsMethod generates a NotEquals method.
func (g *WhereBuilderGenerator) buildNotEqualsMethod(typeName string, field ir.Field, goType string) *ast.FuncDecl {
	return g.buildComparisonMethod(typeName, field, goType, "NotEquals", "NotEquals")
}

func (g *WhereBuilderGenerator) buildGtMethod(typeName string, field ir.Field, goType string) *ast.FuncDecl {
	return g.buildComparisonMethod(typeName, field, goType, "Gt", "GreaterThan")
}

func (g *WhereBuilderGenerator) buildGteMethod(typeName string, field ir.Field, goType string) *ast.FuncDecl {
	return g.buildComparisonMethod(typeName, field, goType, "Gte", "GreaterThanOrEquals")
}

func (g *WhereBuilderGenerator) buildLtMethod(typeName string, field ir.Field, goType string) *ast.FuncDecl {
	return g.buildComparisonMethod(typeName, field, goType, "Lt", "LessThan")
}

func (g *WhereBuilderGenerator) buildLteMethod(typeName string, field ir.Field, goType string) *ast.FuncDecl {
	return g.buildComparisonMethod(typeName, field, goType, "Lte", "LessThanOrEquals")
}

func (g *WhereBuilderGenerator) buildComparisonMethod(typeName string, field ir.Field, goType, methodName, opName string) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent(methodName),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("c")},
					Type:  ast.NewIdent(typeName),
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("value")},
						Type:  ast.NewIdent(goType),
					},
				},
			},
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
								Sel: ast.NewIdent("WithWhere"),
							},
							Args: []ast.Expr{
								&ast.CompositeLit{
									Type: &ast.SelectorExpr{
										X:   ast.NewIdent("domain"),
										Sel: ast.NewIdent("Condition"),
									},
									Elts: []ast.Expr{
										&ast.KeyValueExpr{
											Key:   ast.NewIdent("Field"),
											Value: &ast.BasicLit{Kind: token.STRING, Value: `"` + field.Name + `"`},
										},
										&ast.KeyValueExpr{
											Key:   ast.NewIdent("Operator"),
											Value: &ast.SelectorExpr{X: ast.NewIdent("domain"), Sel: ast.NewIdent(opName)},
										},
										&ast.KeyValueExpr{
											Key:   ast.NewIdent("Value"),
											Value: ast.NewIdent("value"),
										},
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

func (g *WhereBuilderGenerator) buildContainsMethod(typeName string, field ir.Field) *ast.FuncDecl {
	return g.buildStringMethod(typeName, field, "Contains", "Contains")
}

func (g *WhereBuilderGenerator) buildStartsWithMethod(typeName string, field ir.Field) *ast.FuncDecl {
	return g.buildStringMethod(typeName, field, "StartsWith", "StartsWith")
}

func (g *WhereBuilderGenerator) buildEndsWithMethod(typeName string, field ir.Field) *ast.FuncDecl {
	return g.buildStringMethod(typeName, field, "EndsWith", "EndsWith")
}

func (g *WhereBuilderGenerator) buildStringMethod(typeName string, field ir.Field, methodName, opName string) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent(methodName),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("c")},
					Type:  ast.NewIdent(typeName),
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("value")},
						Type:  ast.NewIdent("string"),
					},
				},
			},
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
								Sel: ast.NewIdent("WithWhere"),
							},
							Args: []ast.Expr{
								&ast.CompositeLit{
									Type: &ast.SelectorExpr{
										X:   ast.NewIdent("domain"),
										Sel: ast.NewIdent("Condition"),
									},
									Elts: []ast.Expr{
										&ast.KeyValueExpr{
											Key:   ast.NewIdent("Field"),
											Value: &ast.BasicLit{Kind: token.STRING, Value: `"` + field.Name + `"`},
										},
										&ast.KeyValueExpr{
											Key:   ast.NewIdent("Operator"),
											Value: &ast.SelectorExpr{X: ast.NewIdent("domain"), Sel: ast.NewIdent(opName)},
										},
										&ast.KeyValueExpr{
											Key:   ast.NewIdent("Value"),
											Value: ast.NewIdent("value"),
										},
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

// getGoType returns the Go type string for a field.
func (g *WhereBuilderGenerator) getGoType(field ir.Field) string {
	switch field.Type.PrismaType {
	case "String":
		return "string"
	case "Int":
		return "int"
	case "BigInt":
		return "int64"
	case "Float", "Decimal":
		return "float64"
	case "Boolean":
		return "bool"
	case "DateTime":
		return "time.Time"
	default:
		if field.Type.GoType != "" {
			return field.Type.GoType
		}
		return field.Type.PrismaType
	}
}
