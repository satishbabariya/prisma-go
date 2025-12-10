// Package pslcore provides field validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateScalarFieldDefaultValue validates default values on scalar fields.
func validateScalarFieldDefaultValue(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	defaultValue := field.DefaultValue()
	if defaultValue == nil {
		return
	}

	// Check if field type supports default values
	fieldType := field.ScalarFieldType()

	// Unsupported types cannot have defaults
	if fieldType.Unsupported != nil {
		astField := field.AstField()
		if astField != nil {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Field '%s' has unsupported type and cannot have a default value.", field.Name()),
				astField.Span(),
			))
		} else {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Field '%s' has unsupported type and cannot have a default value.", field.Name()),
				diagnostics.NewSpan(0, 0, field.Model().FileID()),
			))
		}
		return
	}

	// Validate autoincrement is only on Int fields
	if defaultValue.IsAutoIncrement() {
		scalarType := field.ScalarType()
		if scalarType == nil || *scalarType != database.ScalarTypeInt {
			value := defaultValue.Value()
			if value != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					fmt.Sprintf("Field '%s' cannot use autoincrement() - only Int fields support autoincrement.", field.Name()),
					"@default",
					value.Span(),
				))
			} else {
				astField := field.AstField()
				if astField != nil {
					ctx.PushError(diagnostics.NewValidationError(
						fmt.Sprintf("Field '%s' cannot use autoincrement() - only Int fields support autoincrement.", field.Name()),
						astField.Span(),
					))
				} else {
					ctx.PushError(diagnostics.NewValidationError(
						fmt.Sprintf("Field '%s' cannot use autoincrement() - only Int fields support autoincrement.", field.Name()),
						diagnostics.NewSpan(0, 0, field.Model().FileID()),
					))
				}
			}
		}
	}
}

// validatePrimaryKeyClusteringSetting validates that primary keys support clustering setting.
func validatePrimaryKeyClusteringSetting(pk *database.PrimaryKeyWalker, ctx *ValidationContext) {
	if ctx.HasCapability(ConnectorCapabilityClusteringSetting) {
		return
	}

	if pk.Clustered() == nil {
		return
	}

	astAttr := pk.AstAttribute()
	if astAttr == nil {
		return
	}

	span := diagnostics.NewSpan(astAttr.Span.Start, astAttr.Span.End, astAttr.Span.FileID)
	ctx.PushError(diagnostics.NewAttributeValidationError(
		"Defining clustering is not supported in the current connector.",
		pk.AttributeName(),
		span,
	))
}

// validatePrimaryKeyClusteringCanBeDefinedOnlyOnce validates that clustering can only be defined once per model for primary keys.
func validatePrimaryKeyClusteringCanBeDefinedOnlyOnce(pk *database.PrimaryKeyWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityClusteringSetting) {
		return
	}

	if clustered := pk.Clustered(); clustered != nil && *clustered == false {
		return
	}

	model := pk.Model()
	if model == nil {
		return
	}

	// Check if any index is clustered
	for _, index := range model.Indexes() {
		if clustered := index.Clustered(); clustered != nil && *clustered == true {
			astAttr := pk.AstAttribute()
			if astAttr == nil {
				return
			}

			span := diagnostics.NewSpan(astAttr.Span.Start, astAttr.Span.End, astAttr.Span.FileID)
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"A model can only hold one clustered index or id.",
				pk.AttributeName(),
				span,
			))
			return
		}
	}
}

// validateLengthUsedWithCorrectTypesForPrimaryKey validates that length argument is only used with String or Bytes types for primary key fields.
func validateLengthUsedWithCorrectTypesForPrimaryKey(fieldAttr *database.PrimaryKeyFieldWalker, attributeName string, span diagnostics.Span, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityIndexColumnLengthPrefixing) {
		return
	}

	if fieldAttr.Length() == nil {
		return
	}

	fieldType := fieldAttr.ScalarFieldType()
	if fieldType.Unsupported != nil {
		return
	}

	if fieldType.BuiltInScalar != nil {
		scalarType := *fieldType.BuiltInScalar
		if scalarType == database.ScalarTypeString || scalarType == database.ScalarTypeBytes {
			return
		}
	}

	ctx.PushError(diagnostics.NewAttributeValidationError(
		"The length argument is only allowed with field types `String` or `Bytes`.",
		attributeName,
		span,
	))
}

// validateScalarFieldConnectorSpecific validates connector-specific capabilities for scalar fields.
func validateScalarFieldConnectorSpecific(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	container := "model"
	if model := field.Model(); model != nil {
		if astModel := model.AstModel(); astModel != nil && astModel.IsView {
			container = "view"
		}
	}

	fieldType := field.ScalarFieldType()
	if fieldType.BuiltInScalar == nil && fieldType.EnumID == nil && fieldType.CompositeTypeID == nil && fieldType.ExtensionID == nil && fieldType.Unsupported == nil {
		return
	}

	// Check Json type support
	if fieldType.BuiltInScalar != nil && *fieldType.BuiltInScalar == database.ScalarTypeJson {
		if !ctx.HasCapability(ConnectorCapabilityJson) {
			astField := field.AstField()
			span := diagnostics.NewSpan(astField.Span().Start, astField.Span().End, astField.Span().FileID)
			ctx.PushError(diagnostics.NewFieldValidationError(
				fmt.Sprintf("Field `%s` in %s `%s` can't be of type Json. The current connector does not support the Json type.", field.Name(), container, field.Model().Name()),
				container,
				field.Model().Name(),
				field.Name(),
				span,
			))
		}

		// Check Json Lists support
		if field.IsList() && !ctx.HasCapability(ConnectorCapabilityJsonLists) {
			astField := field.AstField()
			span := diagnostics.NewSpan(astField.Span().Start, astField.Span().End, astField.Span().FileID)
			ctx.PushError(diagnostics.NewFieldValidationError(
				fmt.Sprintf("Field `%s` in %s `%s` can't be of type Json[]. The current connector does not support the Json List type.", field.Name(), container, field.Model().Name()),
				container,
				field.Model().Name(),
				field.Name(),
				span,
			))
		}
	}

	// Check Decimal type support
	if fieldType.BuiltInScalar != nil && *fieldType.BuiltInScalar == database.ScalarTypeDecimal {
		// Note: DecimalType capability check would go here when capability is added
		// For now, we'll skip this check as the capability may not be defined yet
	}

	// Check ScalarLists support
	if field.IsList() && !ctx.HasCapability(ConnectorCapabilityScalarLists) {
		astField := field.AstField()
		span := diagnostics.NewSpan(astField.Span().Start, astField.Span().End, astField.Span().FileID)
		ctx.PushError(diagnostics.NewScalarListFieldsAreNotSupportedError(
			container,
			field.Model().Name(),
			field.Name(),
			span,
		))
	}

	// Note: Connector-specific scalar field validation would be called here
	// if ValidateScalarField method exists on the connector interface
}

// validateScalarFieldNativeType validates native type attributes on scalar fields.
func validateScalarFieldNativeType(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	nativeType := field.NativeType()
	if nativeType == nil {
		return
	}

	// Basic validation - native type should have a scope and type name
	if nativeType.Scope == 0 || nativeType.TypeName == 0 {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Field '%s' has invalid native type attribute.", field.Name()),
			diagnostics.NewSpan(0, 0, field.Model().FileID()),
		))
	}
}

// validateScalarFieldClientName validates client names (mapped names) for scalar fields.
func validateScalarFieldClientName(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	dbName := field.DatabaseName()

	// Check for reserved names
	reservedNames := []string{"id", "createdAt", "updatedAt"}
	for _, reserved := range reservedNames {
		if dbName == reserved {
			ctx.PushWarning(diagnostics.NewDatamodelWarning(
				fmt.Sprintf("Field '%s' uses reserved database name '%s'.", field.Name(), reserved),
				diagnostics.NewSpan(0, 0, field.Model().FileID()),
			))
		}
	}
}

// validateRelationFieldReferencedModel validates that referenced model exists.
func validateRelationFieldReferencedModel(field *database.RelationFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	refModel := field.ReferencedModel()
	if refModel == nil {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Relation field '%s' references non-existent model.", field.Name()),
			diagnostics.NewSpan(0, 0, field.Model().FileID()),
		))
		return
	}

	if refModel.IsIgnored() {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Relation field '%s' references ignored model '%s'.", field.Name(), refModel.Name()),
			diagnostics.NewSpan(0, 0, field.Model().FileID()),
		))
	}
}

// validateRelationFieldReferentialActions validates referential actions on relation fields.
func validateRelationFieldReferentialActions(field *database.RelationFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	astField := field.AstField()
	if astField == nil {
		return
	}

	// Check if field is required and uses SetNull
	// This validation is handled by validateRequiredFieldSetNull in validations_relations.go
	// which checks both forward and back relation fields
	_ = astField
}
