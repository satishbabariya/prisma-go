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

	// Analyze models (Pass 1)
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

	// Resolve relations (Pass 2)
	// We need to look up models easily
	modelMap := make(map[string]*ir.Model)
	for i := range result.Models {
		modelMap[result.Models[i].Name] = &result.Models[i]
	}

	for i := range result.Models {
		model := &result.Models[i]
		if err := a.resolveRelations(model, modelMap); err != nil {
			return nil, fmt.Errorf("failed to resolve relations for model %s: %w", model.Name, err)
		}
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
	isOptional := field.Arity.IsOptional() || field.OptionalMark != nil
	isArray := field.Arity.IsList() || field.ListSuffix != nil

	// Map the type)
	goType := a.typeMapper.MapType(prismaType, isOptional, isArray)
	isScalar := a.typeMapper.IsScalarType(prismaType)

	result.Type = ir.FieldType{
		PrismaType: prismaType,
		GoType:     goType,
		IsScalar:   isScalar,
		IsModel:    !isScalar && !isArray, // This initial guess needs refinement in analyzer
		IsEnum:     false,                 // TODO: Check against enum list
	}

	// If it's not a scalar, checks if it's a known model later.
	// For now we assume anything not scalar is a potential model or enum.
	// Since we don't have the full list of enums here easily without pre-scanning,
	// we rely on Pass 2 or simple heuristic.
	// Actually, IsModel in logic above is strictly "!IsScalar".
	// We'll refine "IsModel" vs "IsEnum" if needed, but standard logic usually checks if type exists in Models list.

	result.IsOptional = isOptional
	result.IsArray = isArray

	// Check for field-level attributes (id, unique, map, etc.)
	// Note: @relation is handled in Pass 2
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
		case "default":
			// TODO: Handle default value
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

// resolveRelations resolves relations between models.
func (a *SchemaAnalyzer) resolveRelations(model *ir.Model, modelMap map[string]*ir.Model) error {
	// Find the PSL model corresponding to this IR model to get attributes
	// Efficient lookup would likely need a map from Pass 1, but linear scan is fine for now
	var pslModel *pslast.Model
	for _, m := range a.schema.Models() {
		if m.GetName() == model.Name {
			pslModel = m
			break
		}
	}
	if pslModel == nil {
		return fmt.Errorf("model %s not found in schema", model.Name)
	}

	for i := range model.Fields {
		field := &model.Fields[i]

		// Check if field type matches a known model
		relatedModel, isModel := modelMap[field.Type.PrismaType]
		if !isModel {
			// Not a model relation, skip
			// Also update IsModel/IsEnum flags correctly
			// If it was marked as IsModel but not found in map, it might be an enum
			// We haven't implemented Enum lookup map yet, but let's assume if it is scalar it is false.
			// The current logic in analyzeField sets IsModel = !IsScalar.
			// If it's an enum, IsScalar is false (based on current mapper), so IsModel is true.
			// We should fix this:
			if field.Type.IsModel { // currently true for enums too
				// Check if it's actually an enum
				// For now, if it's not in modelMap, set IsModel=false.
				field.Type.IsModel = false
				// Ideally we'd map enums too to set IsEnum=true
				field.Type.IsEnum = true // simplified assumption for now
			}
			continue
		}

		// It IS a model relation
		field.Type.IsModel = true
		field.Type.IsEnum = false

		// Parse @relation attribute
		relationInfo := &ir.RelationInfo{
			Name:         "", // default
			RelatedModel: relatedModel.Name,
			RelationType: ir.OneToMany, // default, will refine
		}

		// Find @relation attribute on the field
		var relationAttr *pslast.Attribute
		// Locate the PSL field again
		for _, pf := range pslModel.Fields {
			if pf.GetName() == field.Name {
				for _, attr := range pf.Attributes {
					if attr.GetName() == "relation" {
						relationAttr = attr
						break
					}
				}
				break
			}
		}

		if relationAttr != nil {
			// Extract arguments: fields, references, name, onDelete, onUpdate
			if relationAttr.Arguments != nil {
				for _, arg := range relationAttr.Arguments.Arguments {
					argName := ""
					if arg.Name != nil {
						argName = arg.Name.Name
					}

					// If argName is empty and it's a string, it might be the relation name
					if argName == "" {
						if str, ok := arg.Value.(*pslast.StringValue); ok {
							relationInfo.Name = str.Value
						}
					} else if argName == "fields" {
						if array, ok := arg.Value.(*pslast.ArrayExpression); ok {
							// For simplicity assuming single field key for now
							if len(array.Elements) > 0 {
								if ref, ok := array.Elements[0].(*pslast.ConstantValue); ok {
									relationInfo.ForeignKey = ref.Value
								}
							}
						}
					} else if argName == "references" {
						if array, ok := arg.Value.(*pslast.ArrayExpression); ok {
							if len(array.Elements) > 0 {
								if ref, ok := array.Elements[0].(*pslast.ConstantValue); ok {
									relationInfo.References = ref.Value
								}
							}
						}
					}
				}
			}
		}

		// Determine RelationType
		// 1. Array = Many
		// 2. Optional/Required Model = One
		// If current field is Array, it's x-To-Many.
		// If current field is not Array, it's x-To-One.

		// We need to look at the OTHER side of the relation to know the full type.
		// Find the back-relation field on relatedModel.
		var relatedField *ir.Field
		for _, rf := range relatedModel.Fields {
			if rf.Type.PrismaType == model.Name {
				// Potential back relation.
				// If relation names are specified, they must match.
				// If not, it's usually matched by type.
				// Simplify: match first field of correct type for now.
				// (Real logic handles named relations)
				relatedField = &rf // Note: this is a copy in the loop variable, but we just need info.
				// Be careful, 'rf' value changes. We need relatedModel.Fields[k]
				break
			}
		}

		if field.IsArray {
			// I am Many.
			// Check other side.
			if relatedField != nil && relatedField.IsArray {
				relationInfo.RelationType = ir.ManyToMany
			} else {
				relationInfo.RelationType = ir.OneToMany
				// Since I'm the "Many" side, I likely don't store the FK.
				// The "One" side (relatedField) usually stores the FK.
			}
		} else {
			// I am One.
			// Check other side.
			if relatedField != nil && relatedField.IsArray {
				// Other is Many. So Many-to-One (which is One-to-Many from the other perspective).
				// In Prisma IR, we usually call it OneToMany on the "One" side too if we view it as a relation *object*,
				// but here RelationType describes *this specific relation field*?
				// Actually, Prisma uses:
				// OneToOne: both sides are One.
				// OneToMany: One side is One, other is Many.
				// ManyToMany: check logic.

				// Let's stick to standard definitions:
				relationInfo.RelationType = ir.OneToMany
				// Wait, if I am "One" and they are "Many", then relative to ME it is a "Many" relation?
				// No, the relation itself is 1-n.
				// If I am the "Child" (holding FK), I am the "One" side... wait.
				// Post has author (One User). User has posts (Many Post).
				// Post.author is "One". User.posts is "Many".
				// The relation is "OneToMany".
			} else {
				// Both are One.
				relationInfo.RelationType = ir.OneToOne
			}
		}

		// Refine Foreign Keys if implementation side
		// If I have "fields" argument, I am the owner (the one holding the FK).
		// relationInfo already populated from attributes.

		field.Relation = relationInfo
	}

	return nil
}
