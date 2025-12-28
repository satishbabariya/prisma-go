// Package astgen builds input type structs using Go AST.
package astgen

import (
	"go/ast"
	"go/token"

	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
)

// InputTypesGenerator generates Create/Update input structs.
type InputTypesGenerator struct {
	ir *ir.IR
}

// NewInputTypesGenerator creates a new input types generator.
func NewInputTypesGenerator(ir *ir.IR) *InputTypesGenerator {
	return &InputTypesGenerator{ir: ir}
}

// BuildInputTypes builds input type structs for all models.
func (g *InputTypesGenerator) BuildInputTypes() []ast.Decl {
	decls := []ast.Decl{}

	for _, model := range g.ir.Models {
		// Generate CreateInput struct
		decls = append(decls, g.buildCreateInput(model))

		// Generate UpdateInput struct
		decls = append(decls, g.buildUpdateInput(model))
	}

	return decls
}

// buildCreateInput generates a CreateXxxInput struct for a model.
func (g *InputTypesGenerator) buildCreateInput(model ir.Model) *ast.GenDecl {
	fields := []*ast.Field{}

	for _, field := range model.Fields {
		// Skip relation fields
		if field.Relation != nil {
			continue
		}

		// Skip auto-generated fields (like @id with @default)
		if field.IsID && field.DefaultValue != nil {
			continue
		}

		// Build field
		fieldType := g.getFieldType(field)

		// Make optional fields pointer types in Create input
		if field.IsOptional {
			fieldType = &ast.StarExpr{X: fieldType}
		}

		// Get db name from tags or use field name
		dbName := field.Name
		if db, ok := field.Tags["db"]; ok {
			dbName = db
		}

		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(field.GoName)},
			Type:  fieldType,
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: "`json:\"" + dbName + "\"`",
			},
		})
	}

	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent("Create" + model.Name + "Input"),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: fields,
					},
				},
			},
		},
	}
}

// buildUpdateInput generates an UpdateXxxInput struct for a model.
func (g *InputTypesGenerator) buildUpdateInput(model ir.Model) *ast.GenDecl {
	fields := []*ast.Field{}

	for _, field := range model.Fields {
		// Skip relation fields
		if field.Relation != nil {
			continue
		}

		// Skip primary key fields (you typically don't update the PK)
		if field.IsID {
			continue
		}

		// Build field - all fields are optional (pointer) in update
		fieldType := &ast.StarExpr{X: g.getFieldType(field)}

		// Get db name from tags or use field name
		dbName := field.Name
		if db, ok := field.Tags["db"]; ok {
			dbName = db
		}

		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(field.GoName)},
			Type:  fieldType,
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: "`json:\"" + dbName + ",omitempty\"`",
			},
		})
	}

	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent("Update" + model.Name + "Input"),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: fields,
					},
				},
			},
		},
	}
}

// getFieldType returns the Go AST type for a field.
func (g *InputTypesGenerator) getFieldType(field ir.Field) ast.Expr {
	goType := field.Type.GoType

	// Handle common type mappings
	switch field.Type.PrismaType {
	case "String":
		return ast.NewIdent("string")
	case "Int":
		return ast.NewIdent("int")
	case "BigInt":
		return ast.NewIdent("int64")
	case "Float":
		return ast.NewIdent("float64")
	case "Decimal":
		return ast.NewIdent("float64")
	case "Boolean":
		return ast.NewIdent("bool")
	case "DateTime":
		return &ast.SelectorExpr{
			X:   ast.NewIdent("time"),
			Sel: ast.NewIdent("Time"),
		}
	case "Json":
		return &ast.MapType{
			Key:   ast.NewIdent("string"),
			Value: &ast.InterfaceType{Methods: &ast.FieldList{}},
		}
	case "Bytes":
		return &ast.ArrayType{Elt: ast.NewIdent("byte")}
	}

	// Default: use the Go type from the IR
	if goType != "" {
		return ast.NewIdent(goType)
	}

	return ast.NewIdent(field.Type.PrismaType)
}
