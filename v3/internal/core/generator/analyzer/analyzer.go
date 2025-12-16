// Package analyzer analyzes Prisma schemas and produces IR.
package analyzer

import (
	"fmt"

	pslast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/types"
)

// SchemaAnalyzer analyzes PSL schemas.
type SchemaAnalyzer struct {
	schema     *pslast.SchemaAst
	typeMapper *types.TypeMapper
}

// NewSchemaAnalyzer creates a new schema analyzer.
func NewSchemaAnalyzer(schema *pslast.SchemaAst) *SchemaAnalyzer {
	return &SchemaAnalyzer{
		schema:     schema,
		typeMapper: types.NewTypeMapper(),
	}
}

// Analyze analyzes the schema and produces IR.
func (a *SchemaAnalyzer) Analyze() (*ir.IR, error) {
	result := &ir.IR{
		Models: []ir.Model{},
		Enums:  []ir.Enum{},
		Config: ir.Config{
			PackageName: "prisma",
			OutputPath:  "prisma",
		},
	}

	// Analyze models
	for _, model := range a.schema.Models() {
		irModel, err := a.analyzeModel(model)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze model %s: %w", model.GetName(), err)
		}
		result.Models = append(result.Models, irModel)
	}

	// Analyze enums
	for _, enum := range a.schema.Enums() {
		irEnum := a.analyzeEnum(enum)
		result.Enums = append(result.Enums, irEnum)
	}

	return result, nil
}

// analyzeModel analyzes a PSL model.
func (a *SchemaAnalyzer) analyzeModel(model *pslast.Model) (ir.Model, error) {
	result := ir.Model{
		Name:      model.GetName(),
		TableName: model.GetName(), // Default to model name
		Fields:    []ir.Field{},
	}

	// Check for @@map attribute
	for _, attr := range model.BlockAttributes {
		if attr == nil {
			continue
		}
		if attr.GetName() == "map" {
			if attr.Arguments != nil && len(attr.Arguments.Arguments) > 0 {
				if strVal, ok := attr.Arguments.Arguments[0].Value.(*pslast.StringValue); ok {
					result.TableName = strVal.Value
					break
				}
			}
		}
	}

	// Analyze fields
	for _, field := range model.Fields {
		irField, err := a.analyzeField(field)
		if err != nil {
			return result, fmt.Errorf("failed to analyze field %s: %w", field.GetName(), err)
		}
		result.Fields = append(result.Fields, irField)
	}

	return result, nil
}

// analyzeField analyzes a model field.
func (a *SchemaAnalyzer) analyzeField(field *pslast.Field) (ir.Field, error) {
	result := ir.Field{
		Name:   field.GetName(),
		GoName: types.ToPascalCase(field.GetName()),
		Tags:   make(map[string]string),
	}

	// Get field type
	fieldType := field.Type
	prismaType := fieldType.Name
	isOptional := field.Arity.IsOptional()
	isArray := field.Arity.IsList()

	// Map the type
	goType := a.typeMapper.MapType(prismaType, isOptional, isArray)
	isScalar := a.typeMapper.IsScalarType(prismaType)

	result.Type = ir.FieldType{
		PrismaType: prismaType,
		GoType:     goType,
		IsScalar:   isScalar,
		IsModel:    !isScalar && !isArray,
		IsEnum:     false,
	}

	result.IsOptional = isOptional
	result.IsArray = isArray

	// Check attributes
	for _, attr := range field.Attributes {
		if attr == nil {
			continue
		}
		switch attr.GetName() {
		case "id":
			result.IsID = true
		case "unique":
			result.IsUnique = true
		case "map":
			// Get mapped column name
			if attr.Arguments != nil && len(attr.Arguments.Arguments) > 0 {
				if strVal, ok := attr.Arguments.Arguments[0].Value.(*pslast.StringValue); ok {
					result.Tags["db"] = strVal.Value
				}
			}
		}
	}

	// Build struct tags
	if result.Tags["db"] == "" {
		result.Tags["db"] = result.Name
	}
	result.Tags["json"] = result.Name
	if result.IsOptional {
		result.Tags["json"] += ",omitempty"
	}

	return result, nil
}

// analyzeEnum analyzes a PSL enum.
func (a *SchemaAnalyzer) analyzeEnum(enum *pslast.Enum) ir.Enum {
	result := ir.Enum{
		Name:   enum.GetName(),
		Values: []ir.EnumValue{},
	}

	for _, value := range enum.Values {
		result.Values = append(result.Values, ir.EnumValue{
			Name:   value.GetName(),
			GoName: types.ToPascalCase(value.GetName()),
		})
	}

	return result
}
