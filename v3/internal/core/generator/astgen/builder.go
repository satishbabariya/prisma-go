// Package astgen builds Go AST nodes from IR.
package astgen

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/types"
)

// Builder builds Go AST from IR.
type Builder struct {
	ir      *ir.IR
	imports map[string]bool
}

// NewBuilder creates a new AST builder.
func NewBuilder(ir *ir.IR) *Builder {
	return &Builder{
		ir:      ir,
		imports: make(map[string]bool),
	}
}

// BuildFile builds a complete Go file AST.
func (b *Builder) BuildFile() *ast.File {
	file := &ast.File{
		Name:  ast.NewIdent(b.ir.Config.PackageName),
		Decls: []ast.Decl{},
	}

	// Build Client struct
	clientGen := NewClientGenerator(b.ir)
	clientDecls := clientGen.BuildClient()
	file.Decls = append(file.Decls, clientDecls...)

	// Build type declarations for models
	for _, model := range b.ir.Models {
		typeDecl := b.BuildModelStruct(model)
		file.Decls = append(file.Decls, typeDecl)
	}

	// Build enum declarations
	for _, enum := range b.ir.Enums {
		enumDecls := b.BuildEnum(enum)
		file.Decls = append(file.Decls, enumDecls...)
	}

	// Build query builder methods
	qbGen := NewQueryBuilderGenerator(b.ir)
	for _, model := range b.ir.Models {
		methods := qbGen.BuildQueryMethods(model)
		for _, method := range methods {
			file.Decls = append(file.Decls, method)
		}
	}

	// Add required imports
	b.imports["context"] = true

	// Add imports at the beginning
	if len(b.imports) > 0 {
		importDecl := b.buildImportDecl()
		file.Decls = append([]ast.Decl{importDecl}, file.Decls...)
	}

	return file
}

// BuildModelStruct builds a struct type declaration for a model.
func (b *Builder) BuildModelStruct(model ir.Model) *ast.GenDecl {
	fields := &ast.FieldList{
		List: []*ast.Field{},
	}

	// Build fields
	for _, field := range model.Fields {
		// Skip relation fields for now
		if field.Relation != nil {
			continue
		}

		astField := b.buildStructField(field)
		fields.List = append(fields.List, astField)

		// Track imports
		for _, imp := range types.GetImportsForType(field.Type.GoType) {
			b.imports[imp] = true
		}
	}

	// Create struct type spec
	typeSpec := &ast.TypeSpec{
		Name: ast.NewIdent(model.Name),
		Type: &ast.StructType{
			Fields: fields,
		},
	}

	// Create declaration
	return &ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{typeSpec},
	}
}

// buildStructField builds an ast.Field for a struct.
func (b *Builder) buildStructField(field ir.Field) *ast.Field {
	// Parse the Go type and build the type expression
	typeExpr := b.parseTypeExpr(field.Type.GoType)

	// Build struct tag
	tag := types.BuildStructTag(field.Tags)

	return &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(field.GoName)},
		Type:  typeExpr,
		Tag:   &ast.BasicLit{Kind: token.STRING, Value: "`" + tag + "`"},
	}
}

// parseTypeExpr parses a Go type string into an ast.Expr.
func (b *Builder) parseTypeExpr(goType string) ast.Expr {
	// Handle pointer types
	if strings.HasPrefix(goType, "*") {
		return &ast.StarExpr{
			X: b.parseTypeExpr(goType[1:]),
		}
	}

	// Handle slice types
	if strings.HasPrefix(goType, "[]") {
		return &ast.ArrayType{
			Elt: b.parseTypeExpr(goType[2:]),
		}
	}

	// Handle qualified types (e.g., time.Time, json.RawMessage)
	if strings.Contains(goType, ".") {
		parts := strings.SplitN(goType, ".", 2)
		return &ast.SelectorExpr{
			X:   ast.NewIdent(parts[0]),
			Sel: ast.NewIdent(parts[1]),
		}
	}

	// Simple identifier
	return ast.NewIdent(goType)
}

// BuildEnum builds enum type and constants.
func (b *Builder) BuildEnum(enum ir.Enum) []ast.Decl {
	decls := []ast.Decl{}

	// Type declaration
	typeDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(enum.Name),
				Type: ast.NewIdent("string"),
			},
		},
	}
	decls = append(decls, typeDecl)

	// Const declarations
	constSpecs := []ast.Spec{}
	for _, value := range enum.Values {
		constSpecs = append(constSpecs, &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent(enum.Name + value.GoName)},
			Type:  ast.NewIdent(enum.Name),
			Values: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: `"` + value.Name + `"`,
				},
			},
		})
	}

	if len(constSpecs) > 0 {
		constDecl := &ast.GenDecl{
			Tok:   token.CONST,
			Specs: constSpecs,
		}
		decls = append(decls, constDecl)
	}

	return decls
}

// buildImportDecl builds the import declaration.
func (b *Builder) buildImportDecl() *ast.GenDecl {
	specs := []ast.Spec{}

	for importPath := range b.imports {
		specs = append(specs, &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: `"` + importPath + `"`,
			},
		})
	}

	return &ast.GenDecl{
		Tok:   token.IMPORT,
		Specs: specs,
	}
}
