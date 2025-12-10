// Package parserdatabase provides type resolution functionality.
package database

import (
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// ResolveTypes resolves types for all models, enums, and composite types.
// This is the second pass of validation after name resolution.
func ResolveTypes(ctx *Context) {
	for _, entry := range ctx.IterTops() {
		switch {
		case entry.Top.AsModel() != nil:
			model := entry.Top.AsModel()
			modelID := ModelId{
				FileID: entry.TopID.FileID,
				ID:     entry.TopID.ID,
			}
			visitModel(modelID, model, ctx)
		case entry.Top.AsEnum() != nil:
			enum := entry.Top.AsEnum()
			enumID := EnumId{
				FileID: entry.TopID.FileID,
				ID:     entry.TopID.ID,
			}
			visitEnum(enumID, enum, ctx)
		case entry.Top.AsCompositeType() != nil:
			ct := entry.Top.AsCompositeType()
			ctID := CompositeTypeId{
				FileID: entry.TopID.FileID,
				ID:     entry.TopID.ID,
			}
			visitCompositeType(ctID, ct, ctx)
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
func visitModel(modelID ModelId, astModel *ast.Model, ctx *Context) {
	for i, astField := range astModel.Fields {
		fieldID := uint32(i)
		fieldType, err := resolveFieldType(astField, ctx)

		if err != nil {
			// Type not found - try to find similar types for better error messages
			typeName := astField.FieldType.TypeName()
			foundSimilar := false

			// Check for case-insensitive matches in top-level names
			for _, entry := range ctx.IterTops() {
				var name string
				switch {
				case entry.Top.AsModel() != nil:
					name = entry.Top.AsModel().Name.Name
				case entry.Top.AsEnum() != nil:
					name = entry.Top.AsEnum().Name.Name
				case entry.Top.AsCompositeType() != nil:
					name = entry.Top.AsCompositeType().Name.Name
				default:
					continue
				}

				if strings.EqualFold(name, typeName) {
					ctx.PushError(diagnostics.NewTypeForCaseNotFoundError(
						typeName,
						name,
						astField.FieldType.Span(),
					))
					foundSimilar = true
					break
				}
			}

			// Check for case-insensitive matches in scalar types
			if !foundSimilar {
				if scalarType := parseScalarTypeCaseInsensitive(typeName); scalarType != nil {
					ctx.PushError(diagnostics.NewTypeForCaseNotFoundError(
						typeName,
						string(*scalarType),
						astField.FieldType.Span(),
					))
					foundSimilar = true
				}
			}

			if !foundSimilar {
				ctx.PushError(diagnostics.NewTypeNotFoundError(
					typeName,
					astField.FieldType.Span(),
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
func visitEnum(enumID EnumId, astEnum *ast.Enum, ctx *Context) {
	// Validate that enum has at least one value
	if len(astEnum.Values) == 0 {
		ctx.PushError(diagnostics.NewValidationError(
			"An enum must have at least one value.",
			astEnum.Span(),
		))
	}

	// Initialize enum attributes (will be populated during attribute resolution)
	ctx.types.EnumAttributes[enumID] = EnumAttributes{
		MappedName: nil,
	}
}

// visitCompositeType processes a composite type and resolves types for all its fields.
func visitCompositeType(ctID CompositeTypeId, astCT *ast.CompositeType, ctx *Context) {
	for i, astField := range astCT.Fields {
		fieldID := uint32(i)
		fieldType, err := resolveFieldType(astField, ctx)

		if err != nil {
			// Type not found
			ctx.PushError(diagnostics.NewTypeNotFoundError(
				astField.FieldType.TypeName(),
				astField.FieldType.Span(),
			))
			continue
		}

		if fieldType.IsModel && fieldType.ModelID != nil {
			// Composite types cannot have relation fields
			// Find the model name for the error message
			var modelName string
			for _, entry := range ctx.IterTops() {
				if entry.Top.AsModel() != nil {
					entryModelID := ModelId{
						FileID: entry.TopID.FileID,
						ID:     entry.TopID.ID,
					}
					if entryModelID == *fieldType.ModelID {
						modelName = entry.Top.AsModel().Name.Name
						break
					}
				}
			}
			ctx.PushError(diagnostics.NewCompositeTypeValidationError(
				modelName+" refers to a model, making this a relation field. Relation fields inside composite types are not supported.",
				astCT.Name.Name,
				astField.FieldType.Span(),
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
func resolveFieldType(field ast.Field, ctx *Context) (FieldType, error) {
	typeName := field.FieldType.TypeName()

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
				switch {
				case entry.Top.AsModel() != nil:
					modelID := ModelId{
						FileID: topID.FileID,
						ID:     topID.ID,
					}
					return FieldType{
						IsModel:    true,
						ModelID:    &modelID,
						ScalarType: nil,
					}, nil
				case entry.Top.AsEnum() != nil:
					enumID := EnumId{
						FileID: topID.FileID,
						ID:     topID.ID,
					}
					return FieldType{
						IsModel:    false,
						ScalarType: &ScalarFieldType{EnumID: &enumID},
					}, nil
				case entry.Top.AsCompositeType() != nil:
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
