// Package parserdatabase provides type resolution functionality.
package database

import (
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// ResolveTypes resolves types for all models, enums, and composite types.
// This is the second pass of validation after name resolution.
func ResolveTypes(ctx *Context) {
	for _, entry := range ctx.IterTops() {
		switch t := entry.Top.(type) {
		case *v2ast.Model:
			modelID := ModelId{
				FileID: entry.TopID.FileID,
				ID:     entry.TopID.ID,
			}
			visitModel(modelID, t, ctx)
		case *v2ast.Enum:
			enumID := EnumId{
				FileID: entry.TopID.FileID,
				ID:     entry.TopID.ID,
			}
			visitEnum(enumID, t, ctx)
		case *v2ast.CompositeType:
			ctID := CompositeTypeId{
				FileID: entry.TopID.FileID,
				ID:     entry.TopID.ID,
			}
			visitCompositeType(ctID, t, ctx)
		}
	}
}

// FieldType represents the type of a field (either a relation to a model or a scalar type).
type FieldType struct {
	IsModel    bool
	ModelID    *ModelId
	ScalarType *ScalarFieldType
}

// visitModel processes a model and resolves types for all its fields.
func visitModel(modelID ModelId, astModel *v2ast.Model, ctx *Context) {
	for i, astField := range astModel.Fields {
		if astField == nil {
			continue
		}
		fieldID := uint32(i)
		fieldType, err := resolveFieldType(astField, ctx)

		if err != nil {
			// Type not found - try to find similar types for better error messages
			typeName := astField.GetTypeName()
			foundSimilar := false

			// Check for case-insensitive matches in top-level names
			for _, entry := range ctx.IterTops() {
				var name string
				switch t := entry.Top.(type) {
				case *v2ast.Model:
					name = t.GetName()
				case *v2ast.Enum:
					name = t.GetName()
				case *v2ast.CompositeType:
					name = t.GetName()
				default:
					continue
				}

				if strings.EqualFold(name, typeName) {
					pos := astField.Pos
					span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(typeName), diagnostics.FileIDZero)
					ctx.PushError(diagnostics.NewTypeForCaseNotFoundError(
						typeName,
						name,
						span,
					))
					foundSimilar = true
					break
				}
			}

			// Check for case-insensitive matches in scalar types
			if !foundSimilar {
				if scalarType := parseScalarTypeCaseInsensitive(typeName); scalarType != nil {
					pos := astField.Pos
					span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(typeName), diagnostics.FileIDZero)
					ctx.PushError(diagnostics.NewTypeForCaseNotFoundError(
						typeName,
						string(*scalarType),
						span,
					))
					foundSimilar = true
				}
			}

			if !foundSimilar {
				pos := astField.Pos
				span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(typeName), diagnostics.FileIDZero)
				ctx.PushError(diagnostics.NewTypeNotFoundError(
					typeName,
					span,
				))
			}
			continue
		}

		if fieldType.IsModel && fieldType.ModelID != nil {
			// This is a relation field
			rf := RelationField{
				ModelID:           modelID,
				FieldID:           fieldID,
				ReferencedModel:   *fieldType.ModelID,
				OnDelete:          nil,
				OnUpdate:          nil,
				Fields:            nil,
				References:        nil,
				Name:              nil,
				IsIgnored:         false,
				MappedName:        nil,
				RelationAttribute: nil,
			}
			ctx.types.PushRelationField(rf)
		} else if fieldType.ScalarType != nil {
			// This is a scalar field
			sf := ScalarField{
				ModelID:     modelID,
				FieldID:     fieldID,
				Type:        *fieldType.ScalarType,
				IsIgnored:   false,
				IsUpdatedAt: false,
				Default:     nil,
				MappedName:  nil,
				NativeType:  nil,
			}
			ctx.types.PushScalarField(sf)
		}
	}
}

// visitEnum processes an enum and stores its attributes.
func visitEnum(enumID EnumId, astEnum *v2ast.Enum, ctx *Context) {
	// Validate that enum has at least one value
	if len(astEnum.Values) == 0 {
		pos := astEnum.TopPos()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(astEnum.GetName()), diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewValidationError(
			"An enum must have at least one value.",
			span,
		))
	}

	// Initialize enum attributes (will be populated during attribute resolution)
	ctx.types.EnumAttributes[enumID] = EnumAttributes{
		MappedName: nil,
	}
}

// visitCompositeType processes a composite type and resolves types for all its fields.
func visitCompositeType(ctID CompositeTypeId, astCT *v2ast.CompositeType, ctx *Context) {
	for i, astField := range astCT.Fields {
		if astField == nil {
			continue
		}
		fieldID := uint32(i)
		fieldType, err := resolveFieldType(astField, ctx)

		if err != nil {
			// Type not found
			typeName := astField.GetTypeName()
			pos := astField.Pos
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(typeName), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewTypeNotFoundError(
				typeName,
				span,
			))
			continue
		}

		if fieldType.IsModel && fieldType.ModelID != nil {
			// Composite types cannot have relation fields
			// Find the model name for the error message
			var modelName string
			for _, entry := range ctx.IterTops() {
				if model, ok := entry.Top.(*v2ast.Model); ok {
					entryModelID := ModelId{
						FileID: entry.TopID.FileID,
						ID:     entry.TopID.ID,
					}
					if entryModelID == *fieldType.ModelID {
						modelName = model.GetName()
						break
					}
				}
			}
			typeName := astField.GetTypeName()
			pos := astField.Pos
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(typeName), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewCompositeTypeValidationError(
				modelName+" refers to a model, making this a relation field. Relation fields inside composite types are not supported.",
				astCT.GetName(),
				span,
			))
			continue
		}

		if fieldType.ScalarType != nil {
			ctf := CompositeTypeField{
				Type:       *fieldType.ScalarType,
				MappedName: nil,
				Default:    nil,
				NativeType: nil,
			}
			key := CompositeTypeFieldKeyByID{
				CompositeTypeID: ctID,
				FieldID:         fieldID,
			}
			ctx.types.CompositeTypeFields[key] = ctf
		}
	}
}

// resolveFieldType determines the type of a field (relation or scalar).
// Returns (FieldType, error) where error is non-nil if the type is not found.
func resolveFieldType(field *v2ast.Field, ctx *Context) (FieldType, error) {
	typeName := field.GetTypeName()

	// Check if it's a built-in scalar type
	if scalarType := parseScalarType(typeName); scalarType != nil {
		return FieldType{
			IsModel:    false,
			ScalarType: &ScalarFieldType{BuiltInScalar: scalarType},
		}, nil
	}

	// Check if it's a model, enum, or composite type
	typeNameID := ctx.interner.Intern(typeName)
	topID, found := ctx.names.Tops[typeNameID]
	if found {
		// Get the actual top-level item
		for _, entry := range ctx.IterTops() {
			if entry.TopID == topID {
				switch entry.Top.(type) {
				case *v2ast.Model:
					modelID := ModelId{
						FileID: topID.FileID,
						ID:     topID.ID,
					}
					return FieldType{
						IsModel:    true,
						ModelID:    &modelID,
						ScalarType: nil,
					}, nil
				case *v2ast.Enum:
					enumID := EnumId{
						FileID: topID.FileID,
						ID:     topID.ID,
					}
					return FieldType{
						IsModel:    false,
						ScalarType: &ScalarFieldType{EnumID: &enumID},
					}, nil
				case *v2ast.CompositeType:
					ctID := CompositeTypeId{
						FileID: topID.FileID,
						ID:     topID.ID,
					}
					return FieldType{
						IsModel:    false,
						ScalarType: &ScalarFieldType{CompositeTypeID: &ctID},
					}, nil
				}
			}
		}
	}

	// Check if it's an extension type
	if ctx.extensionTypes != nil {
		// Try to find extension type by Prisma name
		for _, entry := range ctx.extensionTypes.Enumerate() {
			if entry.PrismaName == typeName {
				extID := entry.ID
				return FieldType{
					IsModel:    false,
					ScalarType: &ScalarFieldType{ExtensionID: &extID},
				}, nil
			}
		}
	}

	// Unknown type
	return FieldType{
		IsModel:    false,
		ScalarType: nil,
	}, &TypeNotFoundError{TypeName: typeName}
}

// TypeNotFoundError represents an error when a type is not found.
type TypeNotFoundError struct {
	TypeName string
}

func (e *TypeNotFoundError) Error() string {
	return "type not found: " + e.TypeName
}

// parseScalarType parses a string as a scalar type (case-sensitive).
func parseScalarType(s string) *ScalarType {
	switch s {
	case "Int":
		t := ScalarTypeInt
		return &t
	case "BigInt":
		t := ScalarTypeBigInt
		return &t
	case "Float":
		t := ScalarTypeFloat
		return &t
	case "Boolean":
		t := ScalarTypeBoolean
		return &t
	case "String":
		t := ScalarTypeString
		return &t
	case "DateTime":
		t := ScalarTypeDateTime
		return &t
	case "Json":
		t := ScalarTypeJson
		return &t
	case "Bytes":
		t := ScalarTypeBytes
		return &t
	case "Decimal":
		t := ScalarTypeDecimal
		return &t
	default:
		return nil
	}
}

// parseScalarTypeCaseInsensitive parses a string as a scalar type (case-insensitive).
func parseScalarTypeCaseInsensitive(s string) *ScalarType {
	switch strings.ToLower(s) {
	case "int":
		t := ScalarTypeInt
		return &t
	case "bigint":
		t := ScalarTypeBigInt
		return &t
	case "float":
		t := ScalarTypeFloat
		return &t
	case "boolean":
		t := ScalarTypeBoolean
		return &t
	case "string":
		t := ScalarTypeString
		return &t
	case "datetime":
		t := ScalarTypeDateTime
		return &t
	case "json":
		t := ScalarTypeJson
		return &t
	case "bytes":
		t := ScalarTypeBytes
		return &t
	case "decimal":
		t := ScalarTypeDecimal
		return &t
	default:
		return nil
	}
}
