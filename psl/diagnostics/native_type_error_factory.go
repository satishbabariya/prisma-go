package diagnostics

import "fmt"

// NativeTypeErrorFactory creates errors related to native types for a specific connector.
type NativeTypeErrorFactory struct {
	nativeType string
	connector  string
}

// NewNativeTypeErrorFactory creates a new NativeTypeErrorFactory.
func NewNativeTypeErrorFactory(nativeType, connector string) NativeTypeErrorFactory {
	return NativeTypeErrorFactory{
		nativeType: nativeType,
		connector:  connector,
	}
}

// NewScaleLargerThanPrecisionError creates an error when scale is larger than precision.
func (f NativeTypeErrorFactory) NewScaleLargerThanPrecisionError(span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("The scale must not be larger than the precision for the %s native type in %s.", f.nativeType, f.connector), span)
}

// NewIncompatibleNativeTypeWithIndexError creates an error for incompatible native types with indexes.
func (f NativeTypeErrorFactory) NewIncompatibleNativeTypeWithIndexError(message string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("You cannot define an index on fields with native type `%s` of %s.%s", f.nativeType, f.connector, message), span)
}

// NewIncompatibleNativeTypeWithUniqueError creates an error for incompatible native types with unique constraints.
func (f NativeTypeErrorFactory) NewIncompatibleNativeTypeWithUniqueError(message string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Native type `%s` cannot be unique in %s.%s", f.nativeType, f.connector, message), span)
}

// NewIncompatibleNativeTypeWithIDError creates an error for incompatible native types with ID fields.
func (f NativeTypeErrorFactory) NewIncompatibleNativeTypeWithIDError(message string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Native type `%s` of %s cannot be used on a field that is `@id` or `@@id`.%s", f.nativeType, f.connector, message), span)
}

// NewArgumentMOutOfRangeError creates an error for out-of-range M arguments.
func (f NativeTypeErrorFactory) NewArgumentMOutOfRangeError(message string, span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Argument M is out of range for native type `%s` of %s: %s", f.nativeType, f.connector, message), span)
}

// NativeTypeNameUnknown creates an error for unknown native type names.
func (f NativeTypeErrorFactory) NativeTypeNameUnknown(span Span) DatamodelError {
	return NewDatamodelError(fmt.Sprintf("Native type %s is not supported for %s connector.", f.nativeType, f.connector), span)
}

