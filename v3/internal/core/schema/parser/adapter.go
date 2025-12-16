// Package parser provides conversion between PSL AST and v3 domain models.
package parser

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// ASTToDomain converts PSL AST to v3 domain model.
type ASTToDomain struct{}

// NewASTToDomain creates a new PSL AST to domain converter.
func NewASTToDomain() *ASTToDomain {
	return &ASTToDomain{}
}

// ConvertSchema converts a PSL SchemaAst to domain.Schema.
func (a *ASTToDomain) ConvertSchema(pslSchema *ast.SchemaAst) *domain.Schema {
	schema := &domain.Schema{
		Datasources: a.convertDatasources(pslSchema.Sources()),
		Generators:  a.convertGenerators(pslSchema.Generators()),
		Models:      a.convertModels(pslSchema.Models()),
		Enums:       a.convertEnums(pslSchema.Enums()),
	}
	return schema
}

func (a *ASTToDomain) convertDatasources(sources []*ast.SourceConfig) []domain.Datasource {
	result := make([]domain.Datasource, 0, len(sources))
	for _, src := range sources {
		result = append(result, domain.Datasource{
			Name:     src.GetName(),
			Provider: a.getConfigProperty(src, "provider"),
			URL:      a.getConfigProperty(src, "url"),
		})
	}
	return result
}

func (a *ASTToDomain) convertGenerators(generators []*ast.GeneratorConfig) []domain.Generator {
	result := make([]domain.Generator, 0, len(generators))
	for _, gen := range generators {
		result = append(result, domain.Generator{
			Name:     gen.GetName(),
			Provider: a.getConfigProperty(gen, "provider"),
			Output:   a.getConfigProperty(gen, "output"),
		})
	}
	return result
}

func (a *ASTToDomain) convertModels(models []*ast.Model) []domain.Model {
	result := make([]domain.Model, 0, len(models))
	for _, model := range models {
		result = append(result, domain.Model{
			Name:       model.GetName(),
			Fields:     a.convertFields(model.Fields),
			Indexes:    a.convertIndexes(model.BlockAttributes),
			Attributes: a.convertModelAttributes(model.BlockAttributes),
			Comments:   []string{model.GetDocumentation()},
		})
	}
	return result
}

func (a *ASTToDomain) convertFields(fields []*ast.Field) []domain.Field {
	result := make([]domain.Field, 0, len(fields))
	for _, field := range fields {
		result = append(result, domain.Field{
			Name:         field.GetName(),
			Type:         a.convertFieldType(field.Type),
			IsRequired:   !field.Arity.IsOptional(),
			IsList:       field.Arity.IsList(),
			IsUnique:     a.hasAttribute(field.Attributes, "unique"),
			DefaultValue: a.getDefaultValue(field),
			Attributes:   a.convertFieldAttributes(field.Attributes),
		})
	}
	return result
}

func (a *ASTToDomain) convertFieldType(ft *ast.FieldType) domain.FieldType {
	if ft == nil {
		return domain.FieldType{Name: "Unknown"}
	}

	return domain.FieldType{
		Name:      ft.Name,
		IsBuiltin: a.isBuiltinType(ft.Name),
		IsModel:   !a.isBuiltinType(ft.Name) && !a.isEnumType(ft.Name),
		IsEnum:    a.isEnumType(ft.Name),
	}
}

func (a *ASTToDomain) convertEnums(enums []*ast.Enum) []domain.Enum {
	result := make([]domain.Enum, 0, len(enums))
	for _, enum := range enums {
		values := make([]string, 0, len(enum.Values))
		for _, val := range enum.Values {
			values = append(values, val.GetName())
		}
		result = append(result, domain.Enum{
			Name:   enum.GetName(),
			Values: values,
		})
	}
	return result
}

func (a *ASTToDomain) convertIndexes(attrs []*ast.BlockAttribute) []domain.Index {
	var result []domain.Index

	for _, attr := range attrs {
		if attr.Name == nil {
			continue
		}

		switch attr.GetName() {
		case "index":
			if fields := a.getIndexFields(attr); len(fields) > 0 {
				result = append(result, domain.Index{
					Fields: fields,
					Unique: false,
					Type:   domain.BTreeIndex,
				})
			}
		case "unique":
			if fields := a.getIndexFields(attr); len(fields) > 0 {
				result = append(result, domain.Index{
					Fields: fields,
					Unique: true,
					Type:   domain.BTreeIndex,
				})
			}
		}
	}

	return result
}

func (a *ASTToDomain) convertFieldAttributes(attrs []*ast.Attribute) []domain.Attribute {
	result := make([]domain.Attribute, 0, len(attrs))
	for _, attr := range attrs {
		if attr.Name == nil {
			continue
		}
		result = append(result, domain.Attribute{
			Name:      attr.GetName(),
			Arguments: a.convertArguments(attr.Arguments),
		})
	}
	return result
}

func (a *ASTToDomain) convertModelAttributes(attrs []*ast.BlockAttribute) []domain.Attribute {
	result := make([]domain.Attribute, 0, len(attrs))
	for _, attr := range attrs {
		if attr.Name == nil {
			continue
		}
		result = append(result, domain.Attribute{
			Name:      attr.GetName(),
			Arguments: a.convertArguments(attr.Arguments),
		})
	}
	return result
}

func (a *ASTToDomain) convertArguments(args *ast.ArgumentsList) []interface{} {
	if args == nil {
		return nil
	}

	result := make([]interface{}, 0, len(args.Arguments))
	for _, arg := range args.Arguments {
		if arg.Value != nil {
			result = append(result, a.convertExpression(arg.Value))
		}
	}
	return result
}

func (a *ASTToDomain) convertExpression(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.StringValue:
		return e.Value
	case *ast.NumericValue:
		return e.Value
	case *ast.ConstantValue:
		return e.Value
	case *ast.ArrayExpression:
		elements := make([]interface{}, 0, len(e.Elements))
		for _, elem := range e.Elements {
			elements = append(elements, a.convertExpression(elem))
		}
		return elements
	case *ast.FunctionCall:
		return e.Name + "()"
	default:
		return nil
	}
}

// Helper methods

func (a *ASTToDomain) getConfigProperty(config interface{}, propName string) string {
	switch cfg := config.(type) {
	case *ast.SourceConfig:
		for _, prop := range cfg.Properties {
			if prop.Name != nil && prop.Name.Name == propName {
				if str, ok := prop.Value.(*ast.StringValue); ok {
					return str.Value
				}
				if fn, ok := prop.Value.(*ast.FunctionCall); ok && fn.Name != "" {
					// Handle env("DATABASE_URL") pattern
					if fn.Name == "env" && fn.Arguments != nil && len(fn.Arguments.Arguments) > 0 {
						if arg, ok := fn.Arguments.Arguments[0].Value.(*ast.StringValue); ok {
							return arg.Value
						}
					}
				}
			}
		}
	case *ast.GeneratorConfig:
		for _, prop := range cfg.Properties {
			if prop.Name != nil && prop.Name.Name == propName {
				if str, ok := prop.Value.(*ast.StringValue); ok {
					return str.Value
				}
			}
		}
	}
	return ""
}

func (a *ASTToDomain) hasAttribute(attrs []*ast.Attribute, name string) bool {
	for _, attr := range attrs {
		if attr.GetName() == name {
			return true
		}
	}
	return false
}

func (a *ASTToDomain) getDefaultValue(field *ast.Field) interface{} {
	for _, attr := range field.Attributes {
		if attr.GetName() == "default" {
			if attr.Arguments != nil && len(attr.Arguments.Arguments) > 0 {
				return a.convertExpression(attr.Arguments.Arguments[0].Value)
			}
		}
	}
	return nil
}

func (a *ASTToDomain) getIndexFields(attr *ast.BlockAttribute) []string {
	if attr.Arguments == nil || len(attr.Arguments.Arguments) == 0 {
		return nil
	}

	// The first argument is typically the field list
	firstArg := attr.Arguments.Arguments[0].Value
	if arr, ok := firstArg.(*ast.ArrayExpression); ok {
		fields := make([]string, 0, len(arr.Elements))
		for _, elem := range arr.Elements {
			if id, ok := elem.(*ast.ConstantValue); ok {
				fields = append(fields, id.Value)
			}
		}
		return fields
	}

	return nil
}

func (a *ASTToDomain) isBuiltinType(typeName string) bool {
	builtinTypes := map[string]bool{
		"String":   true,
		"Int":      true,
		"BigInt":   true,
		"Float":    true,
		"Decimal":  true,
		"Boolean":  true,
		"DateTime": true,
		"Json":     true,
		"Bytes":    true,
	}
	return builtinTypes[typeName]
}

func (a *ASTToDomain) isEnumType(typeName string) bool {
	// This is a simplification - in a real implementation,
	// you'd need to check against the schema's enum definitions
	return false
}
