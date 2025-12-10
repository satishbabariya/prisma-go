package diagnostics

import (
	"fmt"
	"io"
)

// DatamodelError represents a validation or parser error in a Prisma schema.
type DatamodelError struct {
	span    Span
	message string
}

// NewDatamodelError creates a new DatamodelError with the given message and span.
func NewDatamodelError(message string, span Span) DatamodelError {
	return DatamodelError{
		message: message,
		span:    span,
	}
}

// NewStaticDatamodelError creates a new DatamodelError with a static message.
func NewStaticDatamodelError(message string, span Span) DatamodelError {
	return NewDatamodelError(message, span)
}

// NewLiteralParserError creates an error for invalid literal values.
func NewLiteralParserError(literalType, rawValue string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("\"%s\" is not a valid value for %s.", rawValue, literalType), span)
}

// NewNamedEnvValError creates an error for named env function arguments.
func NewNamedEnvValError(span Span) DatamodelError {
	return NewDatamodelError("The env function expects a singular, unnamed, string argument.", span)
}

// NewArgumentNotFoundError creates an error for missing arguments.
func NewArgumentNotFoundError(argumentName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Argument \"%s\" is missing.", argumentName), span)
}

// NewArgumentCountMismatchError creates an error for wrong argument counts.
func NewArgumentCountMismatchError(functionName string, requiredCount, givenCount int, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Function \"%s\" takes %d arguments, but received %d.", functionName, requiredCount, givenCount), span)
}

// NewAttributeArgumentNotFoundError creates an error for missing attribute arguments.
func NewAttributeArgumentNotFoundError(argumentName, attributeName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Argument \"%s\" is missing in attribute \"@%s\".", argumentName, attributeName), span)
}

// NewSourceArgumentNotFoundError creates an error for missing datasource arguments.
func NewSourceArgumentNotFoundError(argumentName, sourceName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Argument \"%s\" is missing in data source block \"%s\".", argumentName, sourceName), span)
}

// NewGeneratorArgumentNotFoundError creates an error for missing generator arguments.
func NewGeneratorArgumentNotFoundError(argumentName, generatorName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Argument \"%s\" is missing in generator block \"%s\".", argumentName, generatorName), span)
}

// NewAttributeValidationError creates an error for invalid attribute parsing.
func NewAttributeValidationError(message, attributeName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Error parsing attribute \"%s\": %s", attributeName, message), span)
}

// NewDuplicateAttributeError creates an error for duplicate attributes.
func NewDuplicateAttributeError(attributeName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Attribute \"@%s\" can only be defined once.", attributeName), span)
}

// NewIncompatibleNativeTypeError creates an error for incompatible native types.
func NewIncompatibleNativeTypeError(nativeType, fieldType, expectedTypes string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Native type %s is not compatible with declared field type %s, expected field type %s.", nativeType, fieldType, expectedTypes), span)
}

// NewInvalidNativeTypeArgumentError creates an error for invalid native type arguments.
func NewInvalidNativeTypeArgumentError(nativeType, got, expected string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Invalid argument for type %s: %s. Allowed values: %s.", nativeType, got, expected), span)
}

// NewInvalidPrefixForNativeTypesError creates an error for invalid native type prefixes.
func NewInvalidPrefixForNativeTypesError(givenPrefix, expectedPrefix, suggestion string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("The prefix %s is invalid. It must be equal to the name of an existing datasource e.g. %s. Did you mean to use %s?", givenPrefix, expectedPrefix, suggestion), span)
}

// NewNativeTypesNotSupportedError creates an error when native types are not supported.
func NewNativeTypesNotSupportedError(connectorName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Native types are not supported with %s connector", connectorName), span)
}

// NewReservedScalarTypeError creates an error for reserved type names.
func NewReservedScalarTypeError(typeName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("\"%s\" is a reserved scalar type name and cannot be used.", typeName), span)
}

// NewDuplicateEnumDatabaseNameError creates an error for duplicate enum database names.
func NewDuplicateEnumDatabaseNameError(span Span) DatamodelError {
	return NewDatamodelError("An enum with the same database name is already defined.", span)
}

// NewDuplicateModelDatabaseNameError creates an error for duplicate model database names.
func NewDuplicateModelDatabaseNameError(modelDatabaseName, existingModelName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("The model with database name \"%s\" could not be defined because another model or view with this name exists: \"%s\"", modelDatabaseName, existingModelName), span)
}

// NewDuplicateViewDatabaseNameError creates an error for duplicate view database names.
func NewDuplicateViewDatabaseNameError(modelDatabaseName, existingModelName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("The view with database name \"%s\" could not be defined because another model or view with this name exists: \"%s\"", modelDatabaseName, existingModelName), span)
}

// NewDuplicateTopError creates an error for duplicate top-level definitions.
func NewDuplicateTopError(name, topType, existingTopType string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("The %s \"%s\" cannot be defined because a %s with that name already exists.", topType, name, existingTopType), span)
}

// NewDuplicateConfigKeyError creates an error for duplicate config keys.
func NewDuplicateConfigKeyError(confBlockName, keyName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Key \"%s\" is already defined in %s.", keyName, confBlockName), span)
}

// NewDuplicateArgumentError creates an error for duplicate arguments.
func NewDuplicateArgumentError(argName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Argument \"%s\" is already specified.", argName), span)
}

// NewUnusedArgumentError creates an error for unused arguments.
func NewUnusedArgumentError(span Span) DatamodelError {
	return NewDatamodelError("No such argument.", span)
}

// NewDuplicateDefaultArgumentError creates an error for duplicate default arguments.
func NewDuplicateDefaultArgumentError(argName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Argument \"%s\" is already specified as unnamed argument.", argName), span)
}

// NewDuplicateEnumValueError creates an error for duplicate enum values.
func NewDuplicateEnumValueError(enumName, valueName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Value \"%s\" is already defined on enum \"%s\".", valueName, enumName), span)
}

// NewCompositeTypeDuplicateFieldError creates an error for duplicate composite type fields.
func NewCompositeTypeDuplicateFieldError(typeName, fieldName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Field \"%s\" is already defined on composite type \"%s\".", fieldName, typeName), span)
}

// NewDuplicateFieldError creates an error for duplicate fields.
func NewDuplicateFieldError(modelName, fieldName, container string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Field \"%s\" is already defined on %s \"%s\".", fieldName, container, modelName), span)
}

// NewScalarListFieldsAreNotSupportedError creates an error for unsupported list fields.
func NewScalarListFieldsAreNotSupportedError(container, containerName, fieldName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Field \"%s\" in %s \"%s\" can't be a list. The current connector does not support lists of primitive types.", fieldName, container, containerName), span)
}

// NewModelValidationError creates an error for model validation issues.
func NewModelValidationError(message, blockType, modelName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Error validating %s \"%s\": %s", blockType, modelName, message), span)
}

// NewCompositeTypeValidationError creates an error for composite type validation issues.
func NewCompositeTypeValidationError(message, compositeTypeName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Error validating composite type \"%s\": %s", compositeTypeName, message), span)
}

// NewEnumValidationError creates an error for enum validation issues.
func NewEnumValidationError(message, enumName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Error validating enum `%s`: %s", enumName, message), span)
}

// NewCompositeTypeFieldValidationError creates an error for composite type field validation issues.
func NewCompositeTypeFieldValidationError(message, compositeTypeName, field string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Error validating field `%s` in composite type `%s`: %s", field, compositeTypeName, message), span)
}

// NewFieldValidationError creates an error for field validation issues.
func NewFieldValidationError(message, containerType, containerName, field string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Error validating field `%s` in %s `%s`: %s", field, containerType, containerName, message), span)
}

// NewSourceValidationError creates an error for datasource validation issues.
func NewSourceValidationError(message, source string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Error validating datasource `%s`: %s", source, message), span)
}

// NewValidationError creates a general validation error.
func NewValidationError(message string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Error validating: %s", message), span)
}

// NewLegacyParserError creates a legacy parser error.
func NewLegacyParserError(message string, span Span) DatamodelError {
	return NewDatamodelError(message, span)
}

// NewOptionalArgumentCountMismatchError creates an error for optional argument count mismatches.
func NewOptionalArgumentCountMismatchError(nativeType string, optionalCount, givenCount int, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Native type %s takes %d optional arguments, but received %d.", nativeType, optionalCount, givenCount), span)
}

// NewParserError creates a parser error with expected tokens.
func NewParserError(expectedStr string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Unexpected token. Expected one of: %s", expectedStr), span)
}

// NewFunctionalEvaluationError creates an error for functional evaluation issues.
func NewFunctionalEvaluationError(message string, span Span) DatamodelError {
	return NewDatamodelError(message, span)
}

// NewEnvironmentFunctionalEvaluationError creates an error for environment variable issues.
func NewEnvironmentFunctionalEvaluationError(varName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Environment variable not found: %s.", varName), span)
}

// NewTypeNotFoundError creates an error for unknown types.
func NewTypeNotFoundError(typeName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Type \"%s\" is neither a built-in type, nor refers to another model, composite type, or enum.", typeName), span)
}

// NewTypeForCaseNotFoundError creates an error for unknown types with suggestions.
func NewTypeForCaseNotFoundError(typeName, suggestion string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Type \"%s\" is neither a built-in type, nor refers to another model, composite type, or enum. Did you mean \"%s\"?", typeName, suggestion), span)
}

// NewScalarTypeNotFoundError creates an error for unknown scalar types.
func NewScalarTypeNotFoundError(typeName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Type \"%s\" is not a built-in type.", typeName), span)
}

// NewAttributeNotKnownError creates an error for unknown attributes.
func NewAttributeNotKnownError(attributeName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Attribute not known: \"@%s\".", attributeName), span)
}

// NewPropertyNotKnownError creates an error for unknown properties.
func NewPropertyNotKnownError(propertyName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Property not known: \"%s\".", propertyName), span)
}

// NewArgumentNotKnownError creates an error for unknown arguments.
func NewArgumentNotKnownError(propertyName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Argument not known: \"%s\".", propertyName), span)
}

// NewDefaultUnknownFunctionError creates an error for unknown default functions.
func NewDefaultUnknownFunctionError(functionName string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Unknown function in @default(): `%s` is not known. You can read about the available functions here: https://pris.ly/d/attribute-functions", functionName), span)
}

// NewInvalidModelError creates an error for invalid models.
func NewInvalidModelError(msg string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Invalid model: %s", msg), span)
}

// NewDatasourceProviderNotKnownError creates an error for unknown datasource providers.
func NewDatasourceProviderNotKnownError(provider string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Datasource provider not known: \"%s\".", provider), span)
}

// NewDatasourceURLRemovedError creates an error for removed URL property.
func NewDatasourceURLRemovedError(span Span) DatamodelError {
	return NewDatamodelError("The datasource property `url` is no longer supported in schema files. Move connection URLs for Migrate to `prisma.config.ts` and pass either `adapter` for a direct database connection or `accelerateUrl` for Accelerate to the `PrismaClient` constructor. See https://pris.ly/d/config-datasource and https://pris.ly/d/prisma7-client-config", span)
}

// NewDatasourceDirectURLRemovedError creates an error for removed directUrl property.
func NewDatasourceDirectURLRemovedError(span Span) DatamodelError {
	return NewDatamodelError("The datasource property `directUrl` is no longer supported in schema files. Move connection URLs to `prisma.config.ts`. See https://pris.ly/d/config-datasource", span)
}

// NewDatasourceShadowDatabaseURLRemovedError creates an error for removed shadowDatabaseUrl property.
func NewDatasourceShadowDatabaseURLRemovedError(span Span) DatamodelError {
	return NewDatamodelError("The datasource property `shadowDatabaseUrl` is no longer supported in schema files. Move connection URLs to `prisma.config.ts`. See https://pris.ly/d/config-datasource", span)
}

// NewPreviewFeatureNotKnownError creates an error for unknown preview features.
func NewPreviewFeatureNotKnownError(previewFeature, expectedPreviewFeatures string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("The preview feature \"%s\" is not known. Expected one of: %s", previewFeature, expectedPreviewFeatures), span)
}

// NewValueParserError creates an error for value parsing issues.
func NewValueParserError(expectedType, raw string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Expected %s, but found %s.", expectedType, raw), span)
}

// NewNativeTypeArgumentCountMismatchError creates an error for native type argument count mismatches.
func NewNativeTypeArgumentCountMismatchError(nativeType string, requiredCount, givenCount int, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Native type %s takes %d arguments, but received %d.", nativeType, requiredCount, givenCount), span)
}

// NewNativeTypeNameUnknownError creates an error for unknown native types.
func NewNativeTypeNameUnknownError(connectorName, nativeType string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Native type %s is not supported for %s connector.", nativeType, connectorName), span)
}

// NewNativeTypeParserError creates an error for invalid native type parsing.
func NewNativeTypeParserError(nativeType string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Invalid Native type %s.", nativeType), span)
}

// NewTypeMismatchError creates an error for type mismatches.
func NewTypeMismatchError(expectedType, receivedType, raw string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Expected a %s value, but received %s value `%s`.", expectedType, receivedType, raw), span)
}

// NewSchemasArrayEmptyError creates an error for empty schemas arrays.
func NewSchemasArrayEmptyError(span Span) DatamodelError {
	return NewDatamodelError("If provided, the schemas array can not be empty.", span)
}

// NewReferentialIntegrityAndRelationModeCooccurError creates an error for conflicting attributes.
func NewReferentialIntegrityAndRelationModeCooccurError(span Span) DatamodelError {
	return NewDatamodelError("The `referentialIntegrity` and `relationMode` attributes cannot be used together. Please use only `relationMode` instead.", span)
}

// NewConfigPropertyMissingValueError creates an error for missing config property values.
func NewConfigPropertyMissingValueError(propertyName, configName, configKind string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Property %s in %s %s needs to be assigned a value", propertyName, configKind, configName), span)
}

// Span returns the span of the error.
func (e DatamodelError) Span() Span {
	return e.span
}

// Message returns the error message.
func (e DatamodelError) Message() string {
	return e.message
}

// Error implements the error interface.
func (e DatamodelError) Error() string {
	return e.message
}

// PrettyPrint writes a pretty-printed representation of the error to the writer.
func (e DatamodelError) PrettyPrint(w io.Writer, fileName, text string) error {
	return PrettyPrint(w, fileName, text, e.span, e.message, ErrorColorer{})
}
