// Package pslcore provides PostgreSQL native type definitions.
package validation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// PostgresType represents a PostgreSQL native type.
type PostgresType int

const (
	PostgresTypeSmallInt PostgresType = iota
	PostgresTypeInteger
	PostgresTypeBigInt
	PostgresTypeDecimal
	PostgresTypeMoney
	PostgresTypeInet
	PostgresTypeOid
	PostgresTypeCitext
	PostgresTypeReal
	PostgresTypeDoublePrecision
	PostgresTypeVarChar
	PostgresTypeChar
	PostgresTypeText
	PostgresTypeBytea
	PostgresTypeTimestamp
	PostgresTypeTimestamptz
	PostgresTypeDate
	PostgresTypeTime
	PostgresTypeTimetz
	PostgresTypeBoolean
	PostgresTypeBit
	PostgresTypeVarBit
	PostgresTypeUUID
	PostgresTypeXML
	PostgresTypeJson
	PostgresTypeJsonB
)

// String returns the string representation of the PostgreSQL type.
func (t PostgresType) String() string {
	switch t {
	case PostgresTypeSmallInt:
		return "SmallInt"
	case PostgresTypeInteger:
		return "Integer"
	case PostgresTypeBigInt:
		return "BigInt"
	case PostgresTypeDecimal:
		return "Decimal"
	case PostgresTypeMoney:
		return "Money"
	case PostgresTypeInet:
		return "Inet"
	case PostgresTypeOid:
		return "Oid"
	case PostgresTypeCitext:
		return "Citext"
	case PostgresTypeReal:
		return "Real"
	case PostgresTypeDoublePrecision:
		return "DoublePrecision"
	case PostgresTypeVarChar:
		return "VarChar"
	case PostgresTypeChar:
		return "Char"
	case PostgresTypeText:
		return "Text"
	case PostgresTypeBytea:
		return "Bytea"
	case PostgresTypeTimestamp:
		return "Timestamp"
	case PostgresTypeTimestamptz:
		return "Timestamptz"
	case PostgresTypeDate:
		return "Date"
	case PostgresTypeTime:
		return "Time"
	case PostgresTypeTimetz:
		return "Timetz"
	case PostgresTypeBoolean:
		return "Boolean"
	case PostgresTypeBit:
		return "Bit"
	case PostgresTypeVarBit:
		return "VarBit"
	case PostgresTypeUUID:
		return "UUID"
	case PostgresTypeXML:
		return "XML"
	case PostgresTypeJson:
		return "Json"
	case PostgresTypeJsonB:
		return "JsonB"
	default:
		return "Unknown"
	}
}

// PostgresNativeTypeInstance represents a PostgreSQL native type instance with parameters.
type PostgresNativeTypeInstance struct {
	Type      PostgresType
	Precision *int
	Scale     *int
	Length    *int
}

// ParsePostgresNativeType parses a PostgreSQL native type from name and arguments.
func ParsePostgresNativeType(name string, args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	// Normalize name to lowercase for case-insensitive matching
	name = strings.ToLower(name)

	switch name {
	case "smallint", "int2":
		return &PostgresNativeTypeInstance{Type: PostgresTypeSmallInt}
	case "integer", "int", "int4":
		return &PostgresNativeTypeInstance{Type: PostgresTypeInteger}
	case "bigint", "int8":
		return &PostgresNativeTypeInstance{Type: PostgresTypeBigInt}
	case "decimal", "numeric":
		return parseDecimal(args, span, diags)
	case "money":
		return &PostgresNativeTypeInstance{Type: PostgresTypeMoney}
	case "inet":
		return &PostgresNativeTypeInstance{Type: PostgresTypeInet}
	case "oid":
		return &PostgresNativeTypeInstance{Type: PostgresTypeOid}
	case "citext":
		return &PostgresNativeTypeInstance{Type: PostgresTypeCitext}
	case "real", "float4":
		return &PostgresNativeTypeInstance{Type: PostgresTypeReal}
	case "double precision", "doubleprecision", "float8":
		return &PostgresNativeTypeInstance{Type: PostgresTypeDoublePrecision}
	case "varchar", "character varying":
		return parseVarChar(args, span, diags)
	case "char", "character":
		return parseChar(args, span, diags)
	case "text":
		return &PostgresNativeTypeInstance{Type: PostgresTypeText}
	case "bytea":
		return &PostgresNativeTypeInstance{Type: PostgresTypeBytea}
	case "timestamp", "timestamp without time zone":
		return parseTimestamp(args, span, diags)
	case "timestamptz", "timestamp with time zone":
		return parseTimestamptz(args, span, diags)
	case "date":
		return &PostgresNativeTypeInstance{Type: PostgresTypeDate}
	case "time", "time without time zone":
		return parseTime(args, span, diags)
	case "timetz", "time with time zone":
		return parseTimetz(args, span, diags)
	case "boolean", "bool":
		return &PostgresNativeTypeInstance{Type: PostgresTypeBoolean}
	case "bit":
		return parseBit(args, span, diags)
	case "varbit", "bit varying":
		return parseVarBit(args, span, diags)
	case "uuid":
		return &PostgresNativeTypeInstance{Type: PostgresTypeUUID}
	case "xml":
		return &PostgresNativeTypeInstance{Type: PostgresTypeXML}
	case "json":
		return &PostgresNativeTypeInstance{Type: PostgresTypeJson}
	case "jsonb":
		return &PostgresNativeTypeInstance{Type: PostgresTypeJsonB}
	default:
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("PostgreSQL", name, span))
		return nil
	}
}

// parseDecimal parses Decimal(precision, scale) type.
func parseDecimal(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	if len(args) == 0 {
		return &PostgresNativeTypeInstance{Type: PostgresTypeDecimal}
	}
	if len(args) == 2 {
		precision, err1 := strconv.Atoi(args[0])
		scale, err2 := strconv.Atoi(args[1])
		if err1 == nil && err2 == nil {
			return &PostgresNativeTypeInstance{
				Type:      PostgresTypeDecimal,
				Precision: &precision,
				Scale:     &scale,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Decimal", 2, len(args), span))
	return nil
}

// parseVarChar parses VarChar(length) type.
func parseVarChar(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	if len(args) == 0 {
		return &PostgresNativeTypeInstance{Type: PostgresTypeVarChar}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &PostgresNativeTypeInstance{
				Type:   PostgresTypeVarChar,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("VarChar", 1, len(args), span))
	return nil
}

// parseChar parses Char(length) type.
func parseChar(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	if len(args) == 0 {
		return &PostgresNativeTypeInstance{Type: PostgresTypeChar}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &PostgresNativeTypeInstance{
				Type:   PostgresTypeChar,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Char", 1, len(args), span))
	return nil
}

// parseTimestamp parses Timestamp(precision) type.
func parseTimestamp(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	if len(args) == 0 {
		precision := 3 // Default precision
		return &PostgresNativeTypeInstance{
			Type:      PostgresTypeTimestamp,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &PostgresNativeTypeInstance{
				Type:      PostgresTypeTimestamp,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Timestamp", 1, len(args), span))
	return nil
}

// parseTimestamptz parses Timestamptz(precision) type.
func parseTimestamptz(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	if len(args) == 0 {
		precision := 3 // Default precision
		return &PostgresNativeTypeInstance{
			Type:      PostgresTypeTimestamptz,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &PostgresNativeTypeInstance{
				Type:      PostgresTypeTimestamptz,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Timestamptz", 1, len(args), span))
	return nil
}

// parseTime parses Time(precision) type.
func parseTime(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	if len(args) == 0 {
		precision := 3 // Default precision
		return &PostgresNativeTypeInstance{
			Type:      PostgresTypeTime,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &PostgresNativeTypeInstance{
				Type:      PostgresTypeTime,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Time", 1, len(args), span))
	return nil
}

// parseTimetz parses Timetz(precision) type.
func parseTimetz(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	if len(args) == 0 {
		precision := 3 // Default precision
		return &PostgresNativeTypeInstance{
			Type:      PostgresTypeTimetz,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &PostgresNativeTypeInstance{
				Type:      PostgresTypeTimetz,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Timetz", 1, len(args), span))
	return nil
}

// parseBit parses Bit(length) type.
func parseBit(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	if len(args) == 0 {
		return &PostgresNativeTypeInstance{Type: PostgresTypeBit}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &PostgresNativeTypeInstance{
				Type:   PostgresTypeBit,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Bit", 1, len(args), span))
	return nil
}

// parseVarBit parses VarBit(length) type.
func parseVarBit(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *PostgresNativeTypeInstance {
	if len(args) == 0 {
		return &PostgresNativeTypeInstance{Type: PostgresTypeVarBit}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &PostgresNativeTypeInstance{
				Type:   PostgresTypeVarBit,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("VarBit", 1, len(args), span))
	return nil
}

// PostgresNativeTypeToScalarType returns the scalar type for a PostgreSQL native type.
func PostgresNativeTypeToScalarType(nativeType *PostgresNativeTypeInstance) *database.ScalarType {
	if nativeType == nil {
		return nil
	}

	var scalarType database.ScalarType
	switch nativeType.Type {
	case PostgresTypeSmallInt, PostgresTypeInteger:
		scalarType = database.ScalarTypeInt
	case PostgresTypeBigInt:
		scalarType = database.ScalarTypeBigInt
	case PostgresTypeDecimal, PostgresTypeMoney:
		scalarType = database.ScalarTypeDecimal
	case PostgresTypeReal, PostgresTypeDoublePrecision:
		scalarType = database.ScalarTypeFloat
	case PostgresTypeVarChar, PostgresTypeChar, PostgresTypeText, PostgresTypeCitext, PostgresTypeXML, PostgresTypeInet, PostgresTypeUUID:
		scalarType = database.ScalarTypeString
	case PostgresTypeBytea:
		scalarType = database.ScalarTypeBytes
	case PostgresTypeTimestamp, PostgresTypeTimestamptz, PostgresTypeDate, PostgresTypeTime, PostgresTypeTimetz:
		scalarType = database.ScalarTypeDateTime
	case PostgresTypeBoolean:
		scalarType = database.ScalarTypeBoolean
	case PostgresTypeBit, PostgresTypeVarBit:
		scalarType = database.ScalarTypeString
	case PostgresTypeJson, PostgresTypeJsonB:
		scalarType = database.ScalarTypeJson
	case PostgresTypeOid:
		scalarType = database.ScalarTypeInt
	default:
		return nil
	}

	return &scalarType
}

// PostgresScalarTypeToDefaultNativeType returns the default native type for a scalar type.
func PostgresScalarTypeToDefaultNativeType(scalarType database.ScalarType) *PostgresNativeTypeInstance {
	switch scalarType {
	case database.ScalarTypeInt:
		return &PostgresNativeTypeInstance{Type: PostgresTypeInteger}
	case database.ScalarTypeBigInt:
		return &PostgresNativeTypeInstance{Type: PostgresTypeBigInt}
	case database.ScalarTypeFloat:
		return &PostgresNativeTypeInstance{Type: PostgresTypeDoublePrecision}
	case database.ScalarTypeDecimal:
		precision := 65
		scale := 30
		return &PostgresNativeTypeInstance{
			Type:      PostgresTypeDecimal,
			Precision: &precision,
			Scale:     &scale,
		}
	case database.ScalarTypeBoolean:
		return &PostgresNativeTypeInstance{Type: PostgresTypeBoolean}
	case database.ScalarTypeString:
		return &PostgresNativeTypeInstance{Type: PostgresTypeText}
	case database.ScalarTypeDateTime:
		precision := 3
		return &PostgresNativeTypeInstance{
			Type:      PostgresTypeTimestamp,
			Precision: &precision,
		}
	case database.ScalarTypeBytes:
		return &PostgresNativeTypeInstance{Type: PostgresTypeBytea}
	case database.ScalarTypeJson:
		return &PostgresNativeTypeInstance{Type: PostgresTypeJsonB}
	default:
		return nil
	}
}

// PostgresNativeTypeToString returns the string representation of a native type instance.
func PostgresNativeTypeToString(nativeType *PostgresNativeTypeInstance) string {
	if nativeType == nil {
		return ""
	}

	typeName := nativeType.Type.String()

	// Add parameters if present
	if nativeType.Precision != nil && nativeType.Scale != nil {
		return fmt.Sprintf("%s(%d, %d)", typeName, *nativeType.Precision, *nativeType.Scale)
	}
	if nativeType.Precision != nil {
		return fmt.Sprintf("%s(%d)", typeName, *nativeType.Precision)
	}
	if nativeType.Length != nil {
		return fmt.Sprintf("%s(%d)", typeName, *nativeType.Length)
	}

	return typeName
}

// GetPostgresNativeTypeConstructors returns all available PostgreSQL native type constructors.
func GetPostgresNativeTypeConstructors() []*NativeTypeConstructor {
	intType := database.ScalarTypeInt
	bigIntType := database.ScalarTypeBigInt
	decimalType := database.ScalarTypeDecimal
	floatType := database.ScalarTypeFloat
	stringType := database.ScalarTypeString
	bytesType := database.ScalarTypeBytes
	dateTimeType := database.ScalarTypeDateTime
	booleanType := database.ScalarTypeBoolean
	jsonType := database.ScalarTypeJson

	intFieldType := &database.ScalarFieldType{BuiltInScalar: &intType}
	bigIntFieldType := &database.ScalarFieldType{BuiltInScalar: &bigIntType}
	decimalFieldType := &database.ScalarFieldType{BuiltInScalar: &decimalType}
	floatFieldType := &database.ScalarFieldType{BuiltInScalar: &floatType}
	stringFieldType := &database.ScalarFieldType{BuiltInScalar: &stringType}
	bytesFieldType := &database.ScalarFieldType{BuiltInScalar: &bytesType}
	dateTimeFieldType := &database.ScalarFieldType{BuiltInScalar: &dateTimeType}
	booleanFieldType := &database.ScalarFieldType{BuiltInScalar: &booleanType}
	jsonFieldType := &database.ScalarFieldType{BuiltInScalar: &jsonType}

	return []*NativeTypeConstructor{
		// Integer types
		{Name: "SmallInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "Integer", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "BigInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bigIntFieldType}}},
		{Name: "Oid", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},

		// Decimal types
		{Name: "Decimal", NumberOfArgs: 2, NumberOfOptionalArgs: 2, AllowedTypes: []AllowedType{{FieldType: decimalFieldType, ExpectedArguments: []string{"precision", "scale"}}}},
		{Name: "Money", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: decimalFieldType}}},

		// Float types
		{Name: "Real", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},
		{Name: "DoublePrecision", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},

		// String types
		{Name: "VarChar", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "Char", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "Text", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "Citext", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "Inet", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "UUID", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "XML", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},

		// Bytes types
		{Name: "Bytea", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bytesFieldType}}},

		// DateTime types
		{Name: "Timestamp", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "Timestamptz", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "Date", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType}}},
		{Name: "Time", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "Timetz", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},

		// Boolean types
		{Name: "Boolean", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: booleanFieldType}}},

		// Bit types
		{Name: "Bit", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "VarBit", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},

		// JSON types
		{Name: "Json", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: jsonFieldType}}},
		{Name: "JsonB", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: jsonFieldType}}},
	}
}
