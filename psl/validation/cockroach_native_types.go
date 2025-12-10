// Package pslcore provides CockroachDB native type definitions.
// CockroachDB is similar to PostgreSQL, so we reuse PostgreSQL types where possible.
package validation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// CockroachType represents a CockroachDB native type.
type CockroachType int

const (
	CockroachTypeBit CockroachType = iota
	CockroachTypeBool
	CockroachTypeBytes
	CockroachTypeChar
	CockroachTypeDate
	CockroachTypeDecimal
	CockroachTypeFloat4
	CockroachTypeFloat8
	CockroachTypeInet
	CockroachTypeInt2
	CockroachTypeInt4
	CockroachTypeInt8
	CockroachTypeJsonB
	CockroachTypeOid
	CockroachTypeCatalogSingleChar
	CockroachTypeString
	CockroachTypeTime
	CockroachTypeTimestamp
	CockroachTypeTimestamptz
	CockroachTypeTimetz
	CockroachTypeUuid
	CockroachTypeVarBit
)

// String returns the string representation of the CockroachDB type.
func (t CockroachType) String() string {
	switch t {
	case CockroachTypeBit:
		return "Bit"
	case CockroachTypeBool:
		return "Bool"
	case CockroachTypeBytes:
		return "Bytes"
	case CockroachTypeChar:
		return "Char"
	case CockroachTypeDate:
		return "Date"
	case CockroachTypeDecimal:
		return "Decimal"
	case CockroachTypeFloat4:
		return "Float4"
	case CockroachTypeFloat8:
		return "Float8"
	case CockroachTypeInet:
		return "Inet"
	case CockroachTypeInt2:
		return "Int2"
	case CockroachTypeInt4:
		return "Int4"
	case CockroachTypeInt8:
		return "Int8"
	case CockroachTypeJsonB:
		return "JsonB"
	case CockroachTypeOid:
		return "Oid"
	case CockroachTypeCatalogSingleChar:
		return "CatalogSingleChar"
	case CockroachTypeString:
		return "String"
	case CockroachTypeTime:
		return "Time"
	case CockroachTypeTimestamp:
		return "Timestamp"
	case CockroachTypeTimestamptz:
		return "Timestamptz"
	case CockroachTypeTimetz:
		return "Timetz"
	case CockroachTypeUuid:
		return "Uuid"
	case CockroachTypeVarBit:
		return "VarBit"
	default:
		return "Unknown"
	}
}

// CockroachNativeTypeInstance represents a CockroachDB native type instance with parameters.
type CockroachNativeTypeInstance struct {
	Type      CockroachType
	Precision *int
	Scale     *int
	Length    *int
}

// ParseCockroachNativeType parses a CockroachDB native type from name and arguments.
func ParseCockroachNativeType(name string, args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	name = strings.ToLower(name)

	switch name {
	case "bit":
		return parseCockroachBit(args, span, diags)
	case "bool", "boolean":
		return &CockroachNativeTypeInstance{Type: CockroachTypeBool}
	case "bytes", "bytea":
		return &CockroachNativeTypeInstance{Type: CockroachTypeBytes}
	case "char", "character":
		return parseCockroachChar(args, span, diags)
	case "date":
		return &CockroachNativeTypeInstance{Type: CockroachTypeDate}
	case "decimal", "numeric":
		return parseCockroachDecimal(args, span, diags)
	case "float4", "real":
		return &CockroachNativeTypeInstance{Type: CockroachTypeFloat4}
	case "float8", "double precision", "doubleprecision":
		return &CockroachNativeTypeInstance{Type: CockroachTypeFloat8}
	case "inet":
		return &CockroachNativeTypeInstance{Type: CockroachTypeInet}
	case "int2", "smallint":
		return &CockroachNativeTypeInstance{Type: CockroachTypeInt2}
	case "int4", "integer", "int":
		return &CockroachNativeTypeInstance{Type: CockroachTypeInt4}
	case "int8", "bigint":
		return &CockroachNativeTypeInstance{Type: CockroachTypeInt8}
	case "jsonb":
		return &CockroachNativeTypeInstance{Type: CockroachTypeJsonB}
	case "oid":
		return &CockroachNativeTypeInstance{Type: CockroachTypeOid}
	case "catalogsinglechar":
		return &CockroachNativeTypeInstance{Type: CockroachTypeCatalogSingleChar}
	case "string", "text", "varchar":
		return parseCockroachString(args, span, diags)
	case "time", "time without time zone":
		return parseCockroachTime(args, span, diags)
	case "timestamp", "timestamp without time zone":
		return parseCockroachTimestamp(args, span, diags)
	case "timestamptz", "timestamp with time zone":
		return parseCockroachTimestamptz(args, span, diags)
	case "timetz", "time with time zone":
		return parseCockroachTimetz(args, span, diags)
	case "uuid":
		return &CockroachNativeTypeInstance{Type: CockroachTypeUuid}
	case "varbit", "bit varying":
		return parseCockroachVarBit(args, span, diags)
	default:
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("CockroachDB", name, span))
		return nil
	}
}

func parseCockroachBit(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	if len(args) == 0 {
		return &CockroachNativeTypeInstance{Type: CockroachTypeBit}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &CockroachNativeTypeInstance{
				Type:   CockroachTypeBit,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Bit", 1, len(args), span))
	return nil
}

func parseCockroachChar(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	if len(args) == 0 {
		return &CockroachNativeTypeInstance{Type: CockroachTypeChar}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &CockroachNativeTypeInstance{
				Type:   CockroachTypeChar,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Char", 1, len(args), span))
	return nil
}

func parseCockroachDecimal(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	if len(args) == 0 {
		return &CockroachNativeTypeInstance{Type: CockroachTypeDecimal}
	}
	if len(args) == 2 {
		precision, err1 := strconv.Atoi(args[0])
		scale, err2 := strconv.Atoi(args[1])
		if err1 == nil && err2 == nil {
			return &CockroachNativeTypeInstance{
				Type:      CockroachTypeDecimal,
				Precision: &precision,
				Scale:     &scale,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Decimal", 2, len(args), span))
	return nil
}

func parseCockroachString(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	if len(args) == 0 {
		return &CockroachNativeTypeInstance{Type: CockroachTypeString}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &CockroachNativeTypeInstance{
				Type:   CockroachTypeString,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("String", 1, len(args), span))
	return nil
}

func parseCockroachTime(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	if len(args) == 0 {
		precision := 3 // Default precision
		return &CockroachNativeTypeInstance{
			Type:      CockroachTypeTime,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &CockroachNativeTypeInstance{
				Type:      CockroachTypeTime,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Time", 1, len(args), span))
	return nil
}

func parseCockroachTimestamp(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	if len(args) == 0 {
		precision := 3 // Default precision
		return &CockroachNativeTypeInstance{
			Type:      CockroachTypeTimestamp,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &CockroachNativeTypeInstance{
				Type:      CockroachTypeTimestamp,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Timestamp", 1, len(args), span))
	return nil
}

func parseCockroachTimestamptz(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	if len(args) == 0 {
		precision := 3 // Default precision
		return &CockroachNativeTypeInstance{
			Type:      CockroachTypeTimestamptz,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &CockroachNativeTypeInstance{
				Type:      CockroachTypeTimestamptz,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Timestamptz", 1, len(args), span))
	return nil
}

func parseCockroachTimetz(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	if len(args) == 0 {
		precision := 3 // Default precision
		return &CockroachNativeTypeInstance{
			Type:      CockroachTypeTimetz,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &CockroachNativeTypeInstance{
				Type:      CockroachTypeTimetz,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Timetz", 1, len(args), span))
	return nil
}

func parseCockroachVarBit(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *CockroachNativeTypeInstance {
	if len(args) == 0 {
		return &CockroachNativeTypeInstance{Type: CockroachTypeVarBit}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &CockroachNativeTypeInstance{
				Type:   CockroachTypeVarBit,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("VarBit", 1, len(args), span))
	return nil
}

// CockroachNativeTypeToScalarType returns the scalar type for a CockroachDB native type.
func CockroachNativeTypeToScalarType(nativeType *CockroachNativeTypeInstance) *database.ScalarType {
	if nativeType == nil {
		return nil
	}

	var scalarType database.ScalarType
	switch nativeType.Type {
	case CockroachTypeInt2:
		scalarType = database.ScalarTypeInt
	case CockroachTypeInt4, CockroachTypeOid:
		scalarType = database.ScalarTypeInt
	case CockroachTypeInt8:
		scalarType = database.ScalarTypeBigInt
	case CockroachTypeDecimal:
		scalarType = database.ScalarTypeDecimal
	case CockroachTypeFloat4, CockroachTypeFloat8:
		scalarType = database.ScalarTypeFloat
	case CockroachTypeChar, CockroachTypeString, CockroachTypeInet, CockroachTypeUuid, CockroachTypeCatalogSingleChar:
		scalarType = database.ScalarTypeString
	case CockroachTypeBytes:
		scalarType = database.ScalarTypeBytes
	case CockroachTypeDate, CockroachTypeTime, CockroachTypeTimestamp, CockroachTypeTimestamptz, CockroachTypeTimetz:
		scalarType = database.ScalarTypeDateTime
	case CockroachTypeBool:
		scalarType = database.ScalarTypeBoolean
	case CockroachTypeBit, CockroachTypeVarBit:
		scalarType = database.ScalarTypeString
	case CockroachTypeJsonB:
		scalarType = database.ScalarTypeJson
	default:
		return nil
	}

	return &scalarType
}

// CockroachScalarTypeToDefaultNativeType returns the default native type for a scalar type.
func CockroachScalarTypeToDefaultNativeType(scalarType database.ScalarType) *CockroachNativeTypeInstance {
	switch scalarType {
	case database.ScalarTypeInt:
		return &CockroachNativeTypeInstance{Type: CockroachTypeInt4}
	case database.ScalarTypeBigInt:
		return &CockroachNativeTypeInstance{Type: CockroachTypeInt8}
	case database.ScalarTypeFloat:
		return &CockroachNativeTypeInstance{Type: CockroachTypeFloat8}
	case database.ScalarTypeDecimal:
		precision := 65
		scale := 30
		return &CockroachNativeTypeInstance{
			Type:      CockroachTypeDecimal,
			Precision: &precision,
			Scale:     &scale,
		}
	case database.ScalarTypeBoolean:
		return &CockroachNativeTypeInstance{Type: CockroachTypeBool}
	case database.ScalarTypeString:
		return &CockroachNativeTypeInstance{Type: CockroachTypeString}
	case database.ScalarTypeDateTime:
		precision := 3
		return &CockroachNativeTypeInstance{
			Type:      CockroachTypeTimestamp,
			Precision: &precision,
		}
	case database.ScalarTypeBytes:
		return &CockroachNativeTypeInstance{Type: CockroachTypeBytes}
	case database.ScalarTypeJson:
		return &CockroachNativeTypeInstance{Type: CockroachTypeJsonB}
	default:
		return nil
	}
}

// CockroachNativeTypeToString returns the string representation of a native type instance.
func CockroachNativeTypeToString(nativeType *CockroachNativeTypeInstance) string {
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

// GetCockroachNativeTypeConstructors returns all available CockroachDB native type constructors.
func GetCockroachNativeTypeConstructors() []*NativeTypeConstructor {
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
		{Name: "Int2", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "Int4", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "Int8", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bigIntFieldType}}},
		{Name: "Oid", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},

		// Decimal types
		{Name: "Decimal", NumberOfArgs: 2, NumberOfOptionalArgs: 2, AllowedTypes: []AllowedType{{FieldType: decimalFieldType, ExpectedArguments: []string{"precision", "scale"}}}},

		// Float types
		{Name: "Float4", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},
		{Name: "Float8", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},

		// String types
		{Name: "Char", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "String", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "Inet", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "Uuid", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "CatalogSingleChar", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},

		// Bytes types
		{Name: "Bytes", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bytesFieldType}}},

		// DateTime types
		{Name: "Date", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType}}},
		{Name: "Time", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "Timestamp", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "Timestamptz", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "Timetz", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},

		// Boolean types
		{Name: "Bool", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: booleanFieldType}}},

		// Bit types
		{Name: "Bit", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "VarBit", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},

		// JSON types
		{Name: "JsonB", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: jsonFieldType}}},
	}
}
