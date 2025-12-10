// Package pslcore provides SQL Server native type definitions.
package validation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// MsSqlTypeParameter represents a SQL Server type parameter (number or Max).
type MsSqlTypeParameter struct {
	IsMax bool
	Value *int
}

// String returns the string representation of the parameter.
func (p MsSqlTypeParameter) String() string {
	if p.IsMax {
		return "Max"
	}
	if p.Value != nil {
		return strconv.Itoa(*p.Value)
	}
	return ""
}

// MsSqlType represents a SQL Server native type.
type MsSqlType int

const (
	MsSqlTypeTinyInt MsSqlType = iota
	MsSqlTypeSmallInt
	MsSqlTypeInt
	MsSqlTypeBigInt
	MsSqlTypeDecimal
	MsSqlTypeMoney
	MsSqlTypeSmallMoney
	MsSqlTypeBit
	MsSqlTypeFloat
	MsSqlTypeReal
	MsSqlTypeDate
	MsSqlTypeTime
	MsSqlTypeDateTime
	MsSqlTypeDateTime2
	MsSqlTypeDateTimeOffset
	MsSqlTypeSmallDateTime
	MsSqlTypeChar
	MsSqlTypeNChar
	MsSqlTypeVarChar
	MsSqlTypeText
	MsSqlTypeNVarChar
	MsSqlTypeNText
	MsSqlTypeBinary
	MsSqlTypeVarBinary
	MsSqlTypeImage
	MsSqlTypeXml
	MsSqlTypeUniqueIdentifier
)

// String returns the string representation of the SQL Server type.
func (t MsSqlType) String() string {
	switch t {
	case MsSqlTypeTinyInt:
		return "TinyInt"
	case MsSqlTypeSmallInt:
		return "SmallInt"
	case MsSqlTypeInt:
		return "Int"
	case MsSqlTypeBigInt:
		return "BigInt"
	case MsSqlTypeDecimal:
		return "Decimal"
	case MsSqlTypeMoney:
		return "Money"
	case MsSqlTypeSmallMoney:
		return "SmallMoney"
	case MsSqlTypeBit:
		return "Bit"
	case MsSqlTypeFloat:
		return "Float"
	case MsSqlTypeReal:
		return "Real"
	case MsSqlTypeDate:
		return "Date"
	case MsSqlTypeTime:
		return "Time"
	case MsSqlTypeDateTime:
		return "DateTime"
	case MsSqlTypeDateTime2:
		return "DateTime2"
	case MsSqlTypeDateTimeOffset:
		return "DateTimeOffset"
	case MsSqlTypeSmallDateTime:
		return "SmallDateTime"
	case MsSqlTypeChar:
		return "Char"
	case MsSqlTypeNChar:
		return "NChar"
	case MsSqlTypeVarChar:
		return "VarChar"
	case MsSqlTypeText:
		return "Text"
	case MsSqlTypeNVarChar:
		return "NVarChar"
	case MsSqlTypeNText:
		return "NText"
	case MsSqlTypeBinary:
		return "Binary"
	case MsSqlTypeVarBinary:
		return "VarBinary"
	case MsSqlTypeImage:
		return "Image"
	case MsSqlTypeXml:
		return "Xml"
	case MsSqlTypeUniqueIdentifier:
		return "UniqueIdentifier"
	default:
		return "Unknown"
	}
}

// MsSqlNativeTypeInstance represents a SQL Server native type instance with parameters.
type MsSqlNativeTypeInstance struct {
	Type      MsSqlType
	Precision *int
	Scale     *int
	Length    *int
	Parameter *MsSqlTypeParameter // For VarChar/VarBinary with Max
}

// ParseMsSqlNativeType parses a SQL Server native type from name and arguments.
func ParseMsSqlNativeType(name string, args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	name = strings.ToLower(name)

	switch name {
	case "tinyint":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeTinyInt}
	case "smallint":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeSmallInt}
	case "int", "integer":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeInt}
	case "bigint":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeBigInt}
	case "decimal", "numeric":
		return parseMsSqlDecimal(args, span, diags)
	case "money":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeMoney}
	case "smallmoney":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeSmallMoney}
	case "bit":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeBit}
	case "float":
		return parseMsSqlFloat(args, span, diags)
	case "real":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeReal}
	case "date":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeDate}
	case "time":
		return parseMsSqlTime(args, span, diags)
	case "datetime":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeDateTime}
	case "datetime2":
		return parseMsSqlDateTime2(args, span, diags)
	case "datetimeoffset":
		return parseMsSqlDateTimeOffset(args, span, diags)
	case "smalldatetime":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeSmallDateTime}
	case "char", "character":
		return parseMsSqlChar(args, span, diags)
	case "nchar":
		return parseMsSqlNChar(args, span, diags)
	case "varchar", "character varying":
		return parseMsSqlVarChar(args, span, diags)
	case "text":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeText}
	case "nvarchar":
		return parseMsSqlNVarChar(args, span, diags)
	case "ntext":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeNText}
	case "binary":
		return parseMsSqlBinary(args, span, diags)
	case "varbinary":
		return parseMsSqlVarBinary(args, span, diags)
	case "image":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeImage}
	case "xml":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeXml}
	case "uniqueidentifier":
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeUniqueIdentifier}
	default:
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("SQL Server", name, span))
		return nil
	}
}

func parseMsSqlDecimal(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeDecimal}
	}
	if len(args) == 2 {
		precision, err1 := strconv.Atoi(args[0])
		scale, err2 := strconv.Atoi(args[1])
		if err1 == nil && err2 == nil {
			return &MsSqlNativeTypeInstance{
				Type:      MsSqlTypeDecimal,
				Precision: &precision,
				Scale:     &scale,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Decimal", 2, len(args), span))
	return nil
}

func parseMsSqlFloat(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeFloat}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:      MsSqlTypeFloat,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Float", 1, len(args), span))
	return nil
}

func parseMsSqlTime(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		precision := 7 // Default precision for Time
		return &MsSqlNativeTypeInstance{
			Type:      MsSqlTypeTime,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:      MsSqlTypeTime,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Time", 1, len(args), span))
	return nil
}

func parseMsSqlDateTime2(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		precision := 7 // Default precision for DateTime2
		return &MsSqlNativeTypeInstance{
			Type:      MsSqlTypeDateTime2,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:      MsSqlTypeDateTime2,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("DateTime2", 1, len(args), span))
	return nil
}

func parseMsSqlDateTimeOffset(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		precision := 7 // Default precision for DateTimeOffset
		return &MsSqlNativeTypeInstance{
			Type:      MsSqlTypeDateTimeOffset,
			Precision: &precision,
		}
	}
	if len(args) == 1 {
		precision, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:      MsSqlTypeDateTimeOffset,
				Precision: &precision,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("DateTimeOffset", 1, len(args), span))
	return nil
}

func parseMsSqlChar(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeChar}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:   MsSqlTypeChar,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Char", 1, len(args), span))
	return nil
}

func parseMsSqlNChar(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeNChar}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:   MsSqlTypeNChar,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("NChar", 1, len(args), span))
	return nil
}

func parseMsSqlVarChar(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeVarChar}
	}
	if len(args) == 1 {
		arg := strings.ToLower(args[0])
		if arg == "max" {
			return &MsSqlNativeTypeInstance{
				Type:      MsSqlTypeVarChar,
				Parameter: &MsSqlTypeParameter{IsMax: true},
			}
		}
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:   MsSqlTypeVarChar,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("VarChar", 1, len(args), span))
	return nil
}

func parseMsSqlNVarChar(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeNVarChar}
	}
	if len(args) == 1 {
		arg := strings.ToLower(args[0])
		if arg == "max" {
			return &MsSqlNativeTypeInstance{
				Type:      MsSqlTypeNVarChar,
				Parameter: &MsSqlTypeParameter{IsMax: true},
			}
		}
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:   MsSqlTypeNVarChar,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("NVarChar", 1, len(args), span))
	return nil
}

func parseMsSqlBinary(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeBinary}
	}
	if len(args) == 1 {
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:   MsSqlTypeBinary,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("Binary", 1, len(args), span))
	return nil
}

func parseMsSqlVarBinary(args []string, span diagnostics.Span, diags *diagnostics.Diagnostics) *MsSqlNativeTypeInstance {
	if len(args) == 0 {
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeVarBinary}
	}
	if len(args) == 1 {
		arg := strings.ToLower(args[0])
		if arg == "max" {
			return &MsSqlNativeTypeInstance{
				Type:      MsSqlTypeVarBinary,
				Parameter: &MsSqlTypeParameter{IsMax: true},
			}
		}
		length, err := strconv.Atoi(args[0])
		if err == nil {
			return &MsSqlNativeTypeInstance{
				Type:   MsSqlTypeVarBinary,
				Length: &length,
			}
		}
	}
	diags.PushError(diagnostics.NewNativeTypeArgumentCountMismatchError("VarBinary", 1, len(args), span))
	return nil
}

// MsSqlNativeTypeToScalarType returns the scalar type for a SQL Server native type.
func MsSqlNativeTypeToScalarType(nativeType *MsSqlNativeTypeInstance) *database.ScalarType {
	if nativeType == nil {
		return nil
	}

	var scalarType database.ScalarType
	switch nativeType.Type {
	case MsSqlTypeTinyInt, MsSqlTypeSmallInt, MsSqlTypeInt:
		scalarType = database.ScalarTypeInt
	case MsSqlTypeBigInt:
		scalarType = database.ScalarTypeBigInt
	case MsSqlTypeDecimal:
		scalarType = database.ScalarTypeDecimal
	case MsSqlTypeMoney, MsSqlTypeSmallMoney, MsSqlTypeFloat, MsSqlTypeReal:
		scalarType = database.ScalarTypeFloat
	case MsSqlTypeBit:
		scalarType = database.ScalarTypeBoolean // Can also be Int, but Boolean is primary
	case MsSqlTypeChar, MsSqlTypeNChar, MsSqlTypeVarChar, MsSqlTypeText, MsSqlTypeNVarChar, MsSqlTypeNText, MsSqlTypeXml, MsSqlTypeUniqueIdentifier:
		scalarType = database.ScalarTypeString
	case MsSqlTypeBinary, MsSqlTypeVarBinary, MsSqlTypeImage:
		scalarType = database.ScalarTypeBytes
	case MsSqlTypeDate, MsSqlTypeTime, MsSqlTypeDateTime, MsSqlTypeDateTime2, MsSqlTypeDateTimeOffset, MsSqlTypeSmallDateTime:
		scalarType = database.ScalarTypeDateTime
	default:
		return nil
	}

	return &scalarType
}

// MsSqlScalarTypeToDefaultNativeType returns the default native type for a scalar type.
func MsSqlScalarTypeToDefaultNativeType(scalarType database.ScalarType) *MsSqlNativeTypeInstance {
	switch scalarType {
	case database.ScalarTypeInt:
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeInt}
	case database.ScalarTypeBigInt:
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeBigInt}
	case database.ScalarTypeFloat:
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeFloat}
	case database.ScalarTypeDecimal:
		precision := 18
		scale := 0
		return &MsSqlNativeTypeInstance{
			Type:      MsSqlTypeDecimal,
			Precision: &precision,
			Scale:     &scale,
		}
	case database.ScalarTypeBoolean:
		return &MsSqlNativeTypeInstance{Type: MsSqlTypeBit}
	case database.ScalarTypeString:
		length := 255
		return &MsSqlNativeTypeInstance{
			Type:   MsSqlTypeNVarChar,
			Length: &length,
		}
	case database.ScalarTypeDateTime:
		precision := 7
		return &MsSqlNativeTypeInstance{
			Type:      MsSqlTypeDateTime2,
			Precision: &precision,
		}
	case database.ScalarTypeBytes:
		length := 8000
		return &MsSqlNativeTypeInstance{
			Type:   MsSqlTypeVarBinary,
			Length: &length,
		}
	default:
		return nil
	}
}

// MsSqlNativeTypeToString returns the string representation of a native type instance.
func MsSqlNativeTypeToString(nativeType *MsSqlNativeTypeInstance) string {
	if nativeType == nil {
		return ""
	}

	typeName := nativeType.Type.String()

	// Handle Max parameter for VarChar/NVarChar/VarBinary
	if nativeType.Parameter != nil && nativeType.Parameter.IsMax {
		return fmt.Sprintf("%s(Max)", typeName)
	}

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

// GetMsSqlNativeTypeConstructors returns all available SQL Server native type constructors.
func GetMsSqlNativeTypeConstructors() []*NativeTypeConstructor {
	intType := database.ScalarTypeInt
	bigIntType := database.ScalarTypeBigInt
	decimalType := database.ScalarTypeDecimal
	floatType := database.ScalarTypeFloat
	stringType := database.ScalarTypeString
	bytesType := database.ScalarTypeBytes
	dateTimeType := database.ScalarTypeDateTime
	booleanType := database.ScalarTypeBoolean

	intFieldType := &database.ScalarFieldType{BuiltInScalar: &intType}
	bigIntFieldType := &database.ScalarFieldType{BuiltInScalar: &bigIntType}
	decimalFieldType := &database.ScalarFieldType{BuiltInScalar: &decimalType}
	floatFieldType := &database.ScalarFieldType{BuiltInScalar: &floatType}
	stringFieldType := &database.ScalarFieldType{BuiltInScalar: &stringType}
	bytesFieldType := &database.ScalarFieldType{BuiltInScalar: &bytesType}
	dateTimeFieldType := &database.ScalarFieldType{BuiltInScalar: &dateTimeType}
	booleanFieldType := &database.ScalarFieldType{BuiltInScalar: &booleanType}

	return []*NativeTypeConstructor{
		// Integer types
		{Name: "TinyInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "SmallInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "Int", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: intFieldType}}},
		{Name: "BigInt", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bigIntFieldType}}},

		// Decimal types
		{Name: "Decimal", NumberOfArgs: 2, NumberOfOptionalArgs: 2, AllowedTypes: []AllowedType{{FieldType: decimalFieldType, ExpectedArguments: []string{"precision", "scale"}}}},

		// Money types
		{Name: "Money", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},
		{Name: "SmallMoney", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},

		// Bit type
		{Name: "Bit", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: booleanFieldType}, {FieldType: intFieldType}}},

		// Float types
		{Name: "Float", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: floatFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "Real", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: floatFieldType}}},

		// DateTime types
		{Name: "Date", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType}}},
		{Name: "Time", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "DateTime", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType}}},
		{Name: "DateTime2", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "DateTimeOffset", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType, ExpectedArguments: []string{"precision"}}}},
		{Name: "SmallDateTime", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: dateTimeFieldType}}},

		// String types
		{Name: "Char", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "NChar", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "VarChar", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length or Max"}}}},
		{Name: "Text", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "NVarChar", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: stringFieldType, ExpectedArguments: []string{"length or Max"}}}},
		{Name: "NText", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},

		// Bytes types
		{Name: "Binary", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: bytesFieldType, ExpectedArguments: []string{"length"}}}},
		{Name: "VarBinary", NumberOfArgs: 1, NumberOfOptionalArgs: 1, AllowedTypes: []AllowedType{{FieldType: bytesFieldType, ExpectedArguments: []string{"length or Max"}}}},
		{Name: "Image", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: bytesFieldType}}},

		// Other types
		{Name: "Xml", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
		{Name: "UniqueIdentifier", NumberOfArgs: 0, NumberOfOptionalArgs: 0, AllowedTypes: []AllowedType{{FieldType: stringFieldType}}},
	}
}
