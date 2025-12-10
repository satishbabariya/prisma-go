// Package pslcore provides MySQL native type definitions.
package validation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// MySqlType represents a MySQL native type.
type MySqlType int

const (
	MySqlTypeInt MySqlType = iota
	MySqlTypeUnsignedInt
	MySqlTypeSmallInt
	MySqlTypeUnsignedSmallInt
	MySqlTypeTinyInt
	MySqlTypeUnsignedTinyInt
	MySqlTypeMediumInt
	MySqlTypeUnsignedMediumInt
	MySqlTypeBigInt
	MySqlTypeDecimal
	MySqlTypeUnsignedBigInt
	MySqlTypeFloat
	MySqlTypeDouble
	MySqlTypeBit
	MySqlTypeChar
	MySqlTypeVarChar
	MySqlTypeBinary
	MySqlTypeVarBinary
	MySqlTypeTinyBlob
	MySqlTypeBlob
	MySqlTypeMediumBlob
	MySqlTypeLongBlob
	MySqlTypeTinyText
	MySqlTypeText
	MySqlTypeMediumText
	MySqlTypeLongText
	MySqlTypeDate
	MySqlTypeTime
	MySqlTypeDateTime
	MySqlTypeTimestamp
	MySqlTypeYear
	MySqlTypeJson
)

// String returns the string representation of the MySQL type.
func (t MySqlType) String() string {
	switch t {
	case MySqlTypeInt:
		return "Int"
	case MySqlTypeUnsignedInt:
		return "UnsignedInt"
	case MySqlTypeSmallInt:
		return "SmallInt"
	case MySqlTypeUnsignedSmallInt:
		return "UnsignedSmallInt"
	case MySqlTypeTinyInt:
		return "TinyInt"
	case MySqlTypeUnsignedTinyInt:
		return "UnsignedTinyInt"
	case MySqlTypeMediumInt:
		return "MediumInt"
	case MySqlTypeUnsignedMediumInt:
		return "UnsignedMediumInt"
	case MySqlTypeBigInt:
		return "BigInt"
	case MySqlTypeDecimal:
		return "Decimal"
	case MySqlTypeUnsignedBigInt:
		return "UnsignedBigInt"
	case MySqlTypeFloat:
		return "Float"
	case MySqlTypeDouble:
		return "Double"
	case MySqlTypeBit:
		return "Bit"
	case MySqlTypeChar:
		return "Char"
	case MySqlTypeVarChar:
		return "VarChar"
	case MySqlTypeBinary:
		return "Binary"
	case MySqlTypeVarBinary:
		return "VarBinary"
	case MySqlTypeTinyBlob:
		return "TinyBlob"
	case MySqlTypeBlob:
		return "Blob"
	case MySqlTypeMediumBlob:
		return "MediumBlob"
	case MySqlTypeLongBlob:
		return "LongBlob"
	case MySqlTypeTinyText:
		return "TinyText"
	case MySqlTypeText:
		return "Text"
	case MySqlTypeMediumText:
		return "MediumText"
	case MySqlTypeLongText:
		return "LongText"
	case MySqlTypeDate:
		return "Date"
	case MySqlTypeTime:
		return "Time"
	case MySqlTypeDateTime:
		return "DateTime"
	case MySqlTypeTimestamp:
		return "Timestamp"
	case MySqlTypeYear:
		return "Year"
	case MySqlTypeJson:
		return "Json"
	default:
		return "Unknown"
	}
}

// MySqlNativeTypeInstance represents a MySQL native type instance with parameters.
type MySqlNativeTypeInstance struct {
	Type      MySqlType
	Precision *int
	Scale     *int
	Length    *int
}

// ParseMySqlNativeType parses a MySQL native type from name and arguments.
func ParseMySqlNativeType(name string, args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	name = strings.ToLower(name)

	switch name {
	case "int", "integer":
		return &MySqlNativeTypeInstance{Type: MySqlTypeInt}
	case "unsignedint", "unsigned int":
		return &MySqlNativeTypeInstance{Type: MySqlTypeUnsignedInt}
	case "smallint":
		return &MySqlNativeTypeInstance{Type: MySqlTypeSmallInt}
	case "unsignedsmallint", "unsigned smallint":
		return &MySqlNativeTypeInstance{Type: MySqlTypeUnsignedSmallInt}
	case "tinyint":
		return &MySqlNativeTypeInstance{Type: MySqlTypeTinyInt}
	case "unsignedtinyint", "unsigned tinyint":
		return &MySqlNativeTypeInstance{Type: MySqlTypeUnsignedTinyInt}
	case "mediumint":
		return &MySqlNativeTypeInstance{Type: MySqlTypeMediumInt}
	case "unsignedmediumint", "unsigned mediumint":
		return &MySqlNativeTypeInstance{Type: MySqlTypeUnsignedMediumInt}
	case "bigint":
		return &MySqlNativeTypeInstance{Type: MySqlTypeBigInt}
	case "decimal", "numeric":
		return parseMySqlDecimal(args, span, diags)
	case "unsignedbigint", "unsigned bigint":
		return &MySqlNativeTypeInstance{Type: MySqlTypeUnsignedBigInt}
	case "float":
		return &MySqlNativeTypeInstance{Type: MySqlTypeFloat}
	case "double":
		return &MySqlNativeTypeInstance{Type: MySqlTypeDouble}
	case "bit":
		return parseMySqlBit(args, span, diags)
	case "char", "character":
		return parseMySqlChar(args, span, diags)
	case "varchar", "character varying":
		return parseMySqlVarChar(args, span, diags)
	case "binary":
		return parseMySqlBinary(args, span, diags)
	case "varbinary":
		return parseMySqlVarBinary(args, span, diags)
	case "tinyblob":
		return &MySqlNativeTypeInstance{Type: MySqlTypeTinyBlob}
	case "blob":
		return &MySqlNativeTypeInstance{Type: MySqlTypeBlob}
	case "mediumblob":
		return &MySqlNativeTypeInstance{Type: MySqlTypeMediumBlob}
	case "longblob":
		return &MySqlNativeTypeInstance{Type: MySqlTypeLongBlob}
	case "tinytext":
		return &MySqlNativeTypeInstance{Type: MySqlTypeTinyText}
	case "text":
		return &MySqlNativeTypeInstance{Type: MySqlTypeText}
	case "mediumtext":
		return &MySqlNativeTypeInstance{Type: MySqlTypeMediumText}
	case "longtext":
		return &MySqlNativeTypeInstance{Type: MySqlTypeLongText}
	case "date":
		return &MySqlNativeTypeInstance{Type: MySqlTypeDate}
	case "time":
		return parseMySqlTime(args, span, diags)
	case "datetime":
		return parseMySqlDateTime(args, span, diags)
	case "timestamp":
		return parseMySqlTimestamp(args, span, diags)
	case "year":
		return &MySqlNativeTypeInstance{Type: MySqlTypeYear}
	case "json":
		return &MySqlNativeTypeInstance{Type: MySqlTypeJson}
	default:
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("MySQL", name, span))
		return nil
	}
}

func parseMySqlDecimal(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	if len(args) == 0 {
		return &MySqlNativeTypeInstance{Type: MySqlTypeDecimal}
	}
	if len(args) == 2 {
		precision, err1 := strconv.Atoi(args[0])
		scale, err2 := strconv.Atoi(args[1])
		if err1 == nil && err2 == nil {
			return &MySqlNativeTypeInstance{
				Type:      MySqlTypeDecimal,
				Precision: &precision,
				Scale:     &scale,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Decimal", 2, len(args), span))
	return nil
}

func parseMySqlBit(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	if len(args) == 0 {
		return &MySqlNativeTypeInstance{Type: MySqlTypeBit}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MySqlNativeTypeInstance{
				Type:   MySqlTypeBit,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Bit", 1, len(args), span))
	return nil
}

func parseMySqlChar(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	if len(args) == 0 {
		return &MySqlNativeTypeInstance{Type: MySqlTypeChar}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MySqlNativeTypeInstance{
				Type:   MySqlTypeChar,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Char", 1, len(args), span))
	return nil
}

func parseMySqlVarChar(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	if len(args) == 0 {
		// Default length for VarChar
		length := 191
		return &MySqlNativeTypeInstance{
			Type:   MySqlTypeVarChar,
			Length: &length,
		}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MySqlNativeTypeInstance{
				Type:   MySqlTypeVarChar,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("VarChar", 1, len(args), span))
	return nil
}

func parseMySqlBinary(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	if len(args) == 0 {
		return &MySqlNativeTypeInstance{Type: MySqlTypeBinary}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MySqlNativeTypeInstance{
				Type:   MySqlTypeBinary,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Binary", 1, len(args), span))
	return nil
}

func parseMySqlVarBinary(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	if len(args) == 0 {
		return &MySqlNativeTypeInstance{Type: MySqlTypeVarBinary}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MySqlNativeTypeInstance{
				Type:   MySqlTypeVarBinary,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("VarBinary", 1, len(args), span))
	return nil
}

func parseMySqlTime(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	if len(args) == 0 {
		precision := 0
		return &MySqlNativeTypeInstance{
			Type:      MySqlTypeTime,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &MySqlNativeTypeInstance{
				Type:      MySqlTypeTime,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Time", 1, len(args), span))
	return nil
}

func parseMySqlDateTime(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	if len(args) == 0 {
		precision := 0
		return &MySqlNativeTypeInstance{
			Type:      MySqlTypeDateTime,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &MySqlNativeTypeInstance{
				Type:      MySqlTypeDateTime,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("DateTime", 1, len(args), span))
	return nil
}

func parseMySqlTimestamp(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MySqlNativeTypeInstance {
	if len(args) == 0 {
		precision := 0
		return &MySqlNativeTypeInstance{
			Type:      MySqlTypeTimestamp,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &MySqlNativeTypeInstance{
				Type:      MySqlTypeTimestamp,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Timestamp", 1, len(args), span))
	return nil
}

// MySqlNativeTypeToScalarType returns the scalar type for a MySQL native type.
func MySqlNativeTypeToScalarType(nativeType *MySqlNativeTypeInstance) *database.ScalarType {
	if nativeType == nil {
		return nil
	}

	var scalarType database.ScalarType
	switch nativeType.Type {
	case MySqlTypeInt, MySqlTypeUnsignedInt, MySqlTypeSmallInt, MySqlTypeUnsignedSmallInt,
		MySqlTypeMediumInt, MySqlTypeUnsignedMediumInt, MySqlTypeYear:
		scalarType = database.ScalarTypeInt
	case MySqlTypeBigInt, MySqlTypeUnsignedBigInt:
		scalarType = database.ScalarTypeBigInt
	case MySqlTypeTinyInt, MySqlTypeUnsignedTinyInt:
		// TinyInt can be Boolean or Int depending on size
		scalarType = database.ScalarTypeInt // Default to Int, can be Boolean for size 1
	case MySqlTypeDecimal:
		scalarType = database.ScalarTypeDecimal
	case MySqlTypeFloat, MySqlTypeDouble:
		scalarType = database.ScalarTypeFloat
	case MySqlTypeChar, MySqlTypeVarChar, MySqlTypeTinyText, MySqlTypeText,
		MySqlTypeMediumText, MySqlTypeLongText, MySqlTypeBit:
		scalarType = database.ScalarTypeString
	case MySqlTypeBinary, MySqlTypeVarBinary, MySqlTypeTinyBlob, MySqlTypeBlob,
		MySqlTypeMediumBlob, MySqlTypeLongBlob:
		scalarType = database.ScalarTypeBytes
	case MySqlTypeDate, MySqlTypeTime, MySqlTypeDateTime, MySqlTypeTimestamp:
		scalarType = database.ScalarTypeDateTime
	case MySqlTypeJson:
		scalarType = database.ScalarTypeJson
	default:
		return nil
	}

	return &scalarType
}

// MySqlScalarTypeToDefaultNativeType returns the default native type for a scalar type.
func MySqlScalarTypeToDefaultNativeType(scalarType database.ScalarType) *MySqlNativeTypeInstance {
	switch scalarType {
	case database.ScalarTypeInt:
		return &MySqlNativeTypeInstance{Type: MySqlTypeInt}
	case database.ScalarTypeBigInt:
		return &MySqlNativeTypeInstance{Type: MySqlTypeBigInt}
	case database.ScalarTypeFloat:
		return &MySqlNativeTypeInstance{Type: MySqlTypeDouble}
	case database.ScalarTypeDecimal:
		precision := 65
		scale := 30
		return &MySqlNativeTypeInstance{
			Type:      MySqlTypeDecimal,
			Precision: &precision,
			Scale:     &scale,
		}
	case database.ScalarTypeBoolean:
		return &MySqlNativeTypeInstance{Type: MySqlTypeTinyInt}
	case database.ScalarTypeString:
		length := 191
		return &MySqlNativeTypeInstance{
			Type:   MySqlTypeVarChar,
			Length: &length,
		}
	case database.ScalarTypeDateTime:
		precision := 0
		return &MySqlNativeTypeInstance{
			Type:      MySqlTypeDateTime,
			Precision: &precision,
		}
	case database.ScalarTypeBytes:
		return &MySqlNativeTypeInstance{Type: MySqlTypeLongBlob}
	case database.ScalarTypeJson:
		return &MySqlNativeTypeInstance{Type: MySqlTypeJson}
	default:
		return nil
	}
}

// MySqlNativeTypeToString returns the string representation of a native type instance.
func MySqlNativeTypeToString(nativeType *MySqlNativeTypeInstance) string {
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

// GetMySqlNativeTypeConstructors returns all available MySQL native type constructors.
func GetMySqlNativeTypeConstructors() []*NativeTypeConstructor {
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
		{Name: "Int", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "UnsignedInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "SmallInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "UnsignedSmallInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "TinyInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}, {FieldType: booleanFieldType}}},
		{Name: "UnsignedTinyInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}, {FieldType: booleanFieldType}}},
		{Name: "MediumInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "UnsignedMediumInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "BigInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bigIntFieldType}}},
		{Name: "UnsignedBigInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bigIntFieldType}}},

		// Decimal types
		{Name: "Decimal", NumberOfArgs: 2, NumberOfOptionalArgs: 2, AllowedTypes: []AllowedType{{FieldType: decimalFieldType, ExpectedArguments: []string{"precision", "scale"}}}},

		// Float types
		{Name: "Float", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},
		{Name: "Double", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},

		// String types
		{Name: "Char", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "VarChar", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "TinyText", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "Text", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "MediumText", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "LongText", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},

		// Bytes types
		{Name: "Binary", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: bytesFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "VarBinary", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: bytesFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "TinyBlob", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bytesFieldType}}},
		{Name: "Blob", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bytesFieldType}}},
		{Name: "MediumBlob", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bytesFieldType}}},
		{Name: "LongBlob", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bytesFieldType}}},

		// DateTime types
		{Name: "Date", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType}}},
		{Name: "Time", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "DateTime", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "Timestamp", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "Year", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},

		// JSON types
		{Name: "Json", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: jsonFieldType}}},
	}
}
