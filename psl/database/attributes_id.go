// Package parserdatabase provides @id and @@id attribute handling.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// HandleFieldId handles @id on a scalar field.
func HandleFieldId(
	astModel *v2ast.Model,
	sfid ScalarFieldId,
	fieldID uint32,
	modelAttrs *ModelAttributes,
	ctx *Context,
) {
	if modelAttrs.PrimaryKey != nil {
		pos := astModel.TopPos()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(astModel.GetName()), diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewModelValidationError(
			"At most one field must be marked as the id field with the `@id` attribute.",
			"model",
			astModel.GetName(),
			span,
		))
		return
	}

	mappedName := primaryKeyMappedName(ctx)

	// Handle length argument for @id fields
	var length *int
	if expr := ctx.VisitOptionalArg("length"); expr != nil {
		if intVal, ok := CoerceInteger(expr, ctx.diagnostics); ok {
			if intVal > 0 {
				lengthInt := int(intVal)
				length = &lengthInt
			} else {
				ctx.PushAttributeValidationError("The `length` argument must be a positive integer.")
			}
		}
	}

	var sortOrder *SortOrder
	if expr := ctx.VisitOptionalArg("sort"); expr != nil {
		if sortVal, ok := CoerceConstant(expr, ctx.diagnostics); ok {
			switch sortVal {
			case "Desc":
				desc := SortOrderDesc
				sortOrder = &desc
			case "Asc":
				asc := SortOrderAsc
				sortOrder = &asc
			default:
				ctx.PushAttributeValidationError(
					"The `sort` argument can only be `Asc` or `Desc` you provided: " + sortVal + ".",
				)
			}
		}
	}

	clustered := validateClusteringSetting(ctx)

	sourceAttribute := ctx.CurrentAttributeID()

	// Create a FieldWithArgs for the single field
	fieldWithArgs := FieldWithArgs{
		Field:         sfid,
		Path:          nil, // No composite type path for simple @id
		SortOrder:     sortOrder,
		Length:        length,
		OperatorClass: nil, // Operator class not supported for @id
	}

	// Convert AttributeId to uint32 for storage
	// TODO: Store full AttributeId instead of uint32
	sourceAttrID := uint32(sourceAttribute.Index) // Simplified - should encode container too

	modelAttrs.PrimaryKey = &IdAttribute{
		Name:            nil,
		MappedName:      mappedName,
		SourceAttribute: sourceAttrID,
		Fields:          []FieldWithArgs{fieldWithArgs},
		SourceField:     &fieldID,
		Clustered:       clustered,
	}
}

// HandleModelId handles @@id on a model.
func HandleModelId(
	modelID ModelId,
	modelAttrs *ModelAttributes,
	ctx *Context,
) {
	attr := ctx.CurrentAttribute()

	// Get the fields argument
	fieldsExpr, _, err := ctx.VisitDefaultArg("fields")
	if err != nil {
		if diagErr, ok := err.(diagnostics.DatamodelError); ok {
			ctx.PushError(diagErr)
		} else {
			ctx.PushError(diagnostics.NewDatamodelError(err.Error(), diagnostics.EmptySpan()))
		}
		return
	}

	// Use the common field resolution function (without composite type support for @@id)
	pos := attr.Pos
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attr.GetName()), diagnostics.FileIDZero)
	resolvedFields, resolveErr := resolveFieldArrayWithArgs(fieldsExpr, span, modelID, false, false, ctx)
	if resolveErr != nil {
		// Errors already pushed
		return
	}

	astModel := getModelFromID(modelID, ctx)
	if astModel == nil {
		return
	}

	// Validate that all fields are required
	var fieldsThatAreNotRequired []string
	for _, fieldWithArgs := range resolvedFields {
		sf := &ctx.types.ScalarFields[fieldWithArgs.Field]
		if astModel != nil && int(sf.FieldID) < len(astModel.Fields) {
			astField := astModel.Fields[sf.FieldID]
			if astField == nil {
				continue
			}
			// Check if field is required (not optional and not array)
			if astField.Arity.IsOptional() || astField.Arity.IsList() {
				fieldsThatAreNotRequired = append(fieldsThatAreNotRequired, astField.GetName())
			}
		}
	}

	if len(fieldsThatAreNotRequired) > 0 && !modelAttrs.IsIgnored {
		pos := attr.Pos
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attr.GetName()), diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewModelValidationError(
			"The id definition refers to the optional fields: "+formatFieldNames(fieldsThatAreNotRequired)+". ID definitions must reference only required fields.",
			"model",
			astModel.GetName(),
			span,
		))
	}

	if modelAttrs.PrimaryKey != nil {
		pos := astModel.TopPos()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(astModel.GetName()), diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewModelValidationError(
			"Each model must have at most one id criteria. You can't have `@id` and `@@id` at the same time.",
			"model",
			astModel.GetName(),
			span,
		))
		return
	}

	mappedName := primaryKeyMappedName(ctx)
	name := getNameArgument(ctx)
	if name != nil {
		pos := attr.Pos
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attr.GetName()), diagnostics.FileIDZero)
		validateClientName(span, astModel.GetName(), *name, "@@id", ctx)
	}

	clustered := validateClusteringSetting(ctx)

	sourceAttrID := uint32(ctx.CurrentAttributeID().Index) // Simplified

	modelAttrs.PrimaryKey = &IdAttribute{
		Name:            name,
		MappedName:      mappedName,
		SourceAttribute: sourceAttrID,
		Fields:          resolvedFields,
		SourceField:     nil, // @@id doesn't have a source field
		Clustered:       clustered,
	}
}

// ValidateIdFieldArities validates that @id fields are required.
// This must be called after all attributes are resolved.
func ValidateIdFieldArities(
	modelID ModelId,
	modelAttrs *ModelAttributes,
	ctx *Context,
) {
	if modelAttrs.IsIgnored {
		return
	}

	if modelAttrs.PrimaryKey == nil {
		return
	}

	pk := modelAttrs.PrimaryKey
	if pk.SourceField == nil {
		return // @@id doesn't have a source field to validate
	}

	astModel := getModelFromID(modelID, ctx)
	if astModel == nil {
		return
	}

	if int(*pk.SourceField) >= len(astModel.Fields) {
		return
	}

	astField := astModel.Fields[*pk.SourceField]
	if astField == nil {
		return
	}
	if astField.Arity.IsOptional() || astField.Arity.IsList() {
		// TODO: Get proper attribute span from AttributeId
		// For now, use the field span
		pos := astField.Pos
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(astField.GetName()), diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewAttributeValidationError(
			"Fields that are marked as id must be required.",
			"@id",
			span,
		))
	}
}

// primaryKeyMappedName extracts the mapped name from a @map argument on @id.
func primaryKeyMappedName(ctx *Context) *StringId {
	expr := ctx.VisitOptionalArg("map")
	if expr == nil {
		return nil
	}

	name, ok := CoerceString(expr, ctx.diagnostics)
	if !ok {
		return nil
	}

	if name == "" {
		ctx.PushAttributeValidationError("The `map` argument cannot be an empty string.")
		return nil
	}

	nameID := ctx.interner.Intern(name)
	return &nameID
}

// getNameArgument extracts the name argument from an attribute.
func getNameArgument(ctx *Context) *StringId {
	expr := ctx.VisitOptionalArg("name")
	if expr == nil {
		return nil
	}

	name, ok := CoerceString(expr, ctx.diagnostics)
	if !ok {
		return nil
	}

	if name == "" {
		ctx.PushAttributeValidationError("The `name` argument cannot be an empty string.")
		return nil
	}

	nameID := ctx.interner.Intern(name)
	return &nameID
}

// validateClientName validates that a name contains only valid characters for client API.
func validateClientName(span diagnostics.Span, objectName string, name StringId, attribute string, ctx *Context) {
	nameStr := ctx.interner.Get(name)

	// Only alphanumeric characters and underscore are allowed
	isValid := true
	for _, c := range nameStr {
		if c != '_' && !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			isValid = false
			break
		}
	}

	if isValid {
		return
	}

	ctx.PushError(diagnostics.NewModelValidationError(
		"The `name` property within the `"+attribute+"` attribute only allows for the following characters: `_a-zA-Z0-9`.",
		"model",
		objectName,
		span,
	))
}

// validateClusteringSetting extracts and validates the clustered argument.
func validateClusteringSetting(ctx *Context) *bool {
	expr := ctx.VisitOptionalArg("clustered")
	if expr == nil {
		return nil
	}

	val, ok := CoerceBoolean(expr, ctx.diagnostics)
	if !ok {
		return nil
	}

	return &val
}

// Helper functions

var (
	errFieldResolutionFailed           = &fieldResolutionError{msg: "field resolution failed"}
	errFieldResolutionAlreadyDealtWith = &fieldResolutionError{msg: "already dealt with"}
)

type fieldResolutionError struct {
	msg string
}

func (e *fieldResolutionError) Error() string {
	return e.msg
}

func formatFieldNames(fields []string) string {
	if len(fields) == 0 {
		return ""
	}
	if len(fields) == 1 {
		return fields[0]
	}
	result := fields[0]
	for i := 1; i < len(fields)-1; i++ {
		result += ", " + fields[i]
	}
	result += " and " + fields[len(fields)-1]
	return result
}
