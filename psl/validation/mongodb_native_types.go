// Package pslcore provides MongoDB native type definitions.
package validation

import (
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// MongoDbType represents a MongoDB native type (BSON type).
type MongoDbType int

const (
	MongoDbTypeString MongoDbType = iota
	MongoDbTypeDouble
	MongoDbTypeBinData
	MongoDbTypeObjectId
	MongoDbTypeBool
	MongoDbTypeDate
	MongoDbTypeInt
	MongoDbTypeTimestamp
	MongoDbTypeLong
	MongoDbTypeJson
)

// String returns the string representation of the MongoDB type.
func (t MongoDbType) String() string {
	switch t {
	case MongoDbTypeString:
		return "String"
	case MongoDbTypeDouble:
		return "Double"
	case MongoDbTypeBinData:
		return "BinData"
	case MongoDbTypeObjectId:
		return "ObjectId"
	case MongoDbTypeBool:
		return "Bool"
	case MongoDbTypeDate:
		return "Date"
	case MongoDbTypeInt:
		return "Int"
	case MongoDbTypeTimestamp:
		return "Timestamp"
	case MongoDbTypeLong:
		return "Long"
	case MongoDbTypeJson:
		return "Json"
	default:
		return "Unknown"
	}
}

// MongoDbNativeTypeInstance represents a MongoDB native type instance.
// MongoDB types don't have parameters like SQL types.
type MongoDbNativeTypeInstance struct {
	Type MongoDbType
}

// ParseMongoDbNativeType parses a MongoDB native type from name and arguments.
func ParseMongoDbNativeType(name string, args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MongoDbNativeTypeInstance {
	name = strings.ToLower(name)

	// MongoDB native types don't take arguments
	if len(args) > 0 {
		diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError(name, 0, len(args), span))
		return nil
	}

	switch name {
	case "string":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeString}
	case "double":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeDouble}
	case "bindata", "binary":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeBinData}
	case "objectid":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeObjectId}
	case "bool", "boolean":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeBool}
	case "date", "datetime":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeDate}
	case "int", "int32":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeInt}
	case "timestamp":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeTimestamp}
	case "long", "int64":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeLong}
	case "json":
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeJson}
	default:
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("MongoDB", name, span))
		return nil
	}
}

// MongoDbNativeTypeToScalarType returns the scalar type for a MongoDB native type.
func MongoDbNativeTypeToScalarType(nativeType *MongoDbNativeTypeInstance) *database.ScalarType {
	if nativeType == nil {
		return nil
	}

	var scalarType database.ScalarType
	switch nativeType.Type {
	case MongoDbTypeString, MongoDbTypeObjectId:
		scalarType = database.ScalarTypeString
	case MongoDbTypeDouble:
		scalarType = database.ScalarTypeFloat
	case MongoDbTypeBinData:
		scalarType = database.ScalarTypeBytes
	case MongoDbTypeBool:
		scalarType = database.ScalarTypeBoolean
	case MongoDbTypeDate, MongoDbTypeTimestamp:
		scalarType = database.ScalarTypeDateTime
	case MongoDbTypeInt:
		scalarType = database.ScalarTypeInt
	case MongoDbTypeLong:
		// Long can map to Int or BigInt depending on value
		scalarType = database.ScalarTypeBigInt // Default to BigInt
	case MongoDbTypeJson:
		scalarType = database.ScalarTypeJson
	default:
		return nil
	}

	return &scalarType
}

// MongoDbScalarTypeToDefaultNativeType returns the default native type for a scalar type.
func MongoDbScalarTypeToDefaultNativeType(scalarType database.ScalarType) *MongoDbNativeTypeInstance {
	switch scalarType {
	case database.ScalarTypeInt:
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeLong}
	case database.ScalarTypeBigInt:
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeLong}
	case database.ScalarTypeFloat:
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeDouble}
	case database.ScalarTypeBoolean:
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeBool}
	case database.ScalarTypeString:
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeString}
	case database.ScalarTypeDateTime:
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeDate}
	case database.ScalarTypeBytes:
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeBinData}
	case database.ScalarTypeJson:
		return &MongoDbNativeTypeInstance{Type: MongoDbTypeJson}
	default:
		return nil
	}
}

// MongoDbNativeTypeToString returns the string representation of a native type instance.
func MongoDbNativeTypeToString(nativeType *MongoDbNativeTypeInstance) string {
	if nativeType == nil {
		return ""
	}
	return nativeType.Type.String()
}

// GetMongoDbNativeTypeConstructors returns all available MongoDB native type constructors.
func GetMongoDbNativeTypeConstructors() []*NativeTypeConstructor {
	intType := database.ScalarTypeInt
	bigIntType := database.ScalarTypeBigInt
	floatType := database.ScalarTypeFloat
	stringType := database.ScalarTypeString
	bytesType := database.ScalarTypeBytes
	dateTimeType := database.ScalarTypeDateTime
	booleanType := database.ScalarTypeBoolean
	jsonType := database.ScalarTypeJson

	intFieldType := &database.ScalarFieldType{BuiltInScalar: &intType}
	bigIntFieldType := &database.ScalarFieldType{BuiltInScalar: &bigIntType}
	floatFieldType := &database.ScalarFieldType{BuiltInScalar: &floatType}
	stringFieldType := &database.ScalarFieldType{BuiltInScalar: &stringType}
	bytesFieldType := &database.ScalarFieldType{BuiltInScalar: &bytesType}
	dateTimeFieldType := &database.ScalarFieldType{BuiltInScalar: &dateTimeType}
	booleanFieldType := &database.ScalarFieldType{BuiltInScalar: &booleanType}
	jsonFieldType := &database.ScalarFieldType{BuiltInScalar: &jsonType}

	return []*NativeTypeConstructor{
		// String types
		{Name: "String", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "ObjectId", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}, {FieldType: bytesFieldType}}},

		// Numeric types
		{Name: "Double", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},
		{Name: "Int", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "Long", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}, {FieldType: bigIntFieldType}}},

		// Boolean type
		{Name: "Bool", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: booleanFieldType}}},

		// DateTime types
		{Name: "Date", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType}}},
		{Name: "Timestamp", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType}}},

		// Bytes types
		{Name: "BinData", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bytesFieldType}}},

		// JSON type
		{Name: "Json", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: jsonFieldType}}},
	}
}
