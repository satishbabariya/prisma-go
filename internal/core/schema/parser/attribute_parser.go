// Package parser provides enhanced attribute parsing for Prisma schema.
package parser

import (
	"fmt"
	"strings"

	pslast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// AttributeParser handles parsing of Prisma attributes.
type AttributeParser struct{}

// NewAttributeParser creates a new attribute parser.
func NewAttributeParser() *AttributeParser {
	return &AttributeParser{}
}

// ParseFieldAttributes parses field-level attributes.
// Supports: @id, @unique, @default(), @map(), @relation(), @updatedAt, @db.*
func (ap *AttributeParser) ParseFieldAttributes(pslField *pslast.Field) []domain.Attribute {
	var attributes []domain.Attribute

	for _, attr := range pslField.Attributes {
		if attr == nil {
			continue
		}

		parsedAttr := ap.parseAttribute(attr)
		if parsedAttr != nil {
			attributes = append(attributes, *parsedAttr)
		}
	}

	return attributes
}

// ParseBlockAttributes parses block-level attributes.
// Supports: @@id, @@unique, @@index, @@map, @@ignore, @@schema
func (ap *AttributeParser) ParseBlockAttributes(pslModel *pslast.Model) []domain.Attribute {
	var attributes []domain.Attribute

	for _, attr := range pslModel.BlockAttributes {
		if attr == nil {
			continue
		}

		parsedAttr := ap.parseAttribute(attr)
		if parsedAttr != nil {
			attributes = append(attributes, *parsedAttr)
		}
	}

	return attributes
}

// parseAttribute converts a PSL attribute to domain attribute.
func (ap *AttributeParser) parseAttribute(attr interface{}) *domain.Attribute {
	// Type assertion to get the attribute name
	var name string
	var args []interface{}

	switch a := attr.(type) {
	case *pslast.Attribute:
		if a == nil {
			return nil
		}
		name = a.GetName()
		args = ap.parseAttributeArgs(a.Arguments)

	case *pslast.BlockAttribute:
		if a == nil {
			return nil
		}
		name = a.GetName()
		args = ap.parseAttributeArgs(a.Arguments)

	default:
		return nil
	}

	return &domain.Attribute{
		Name:      name,
		Arguments: args,
	}
}

// parseAttributeArgs extracts arguments from attribute arguments list.
func (ap *AttributeParser) parseAttributeArgs(argList *pslast.ArgumentsList) []interface{} {
	if argList == nil || len(argList.Arguments) == 0 {
		return nil
	}

	var args []interface{}
	for _, arg := range argList.Arguments {
		if arg == nil || arg.Value == nil {
			continue
		}

		value := ap.parseArgumentValue(arg.Value)
		if value != nil {
			args = append(args, value)
		}
	}

	return args
}

// parseArgumentValue extracts the actual value from an expression.
func (ap *AttributeParser) parseArgumentValue(expr pslast.Expression) interface{} {
	if expr == nil {
		return nil
	}

	switch v := expr.(type) {
	case *pslast.StringValue:
		return v.Value

	case *pslast.NumericValue:
		return v.Value

	case *pslast.ConstantValue:
		// Handle boolean constants
		if boolVal, ok := v.AsBooleanValue(); ok {
			return boolVal
		}
		return v.Value

	case *pslast.ArrayExpression:
		return ap.parseArrayValue(v)

	case *pslast.FunctionCall:
		return ap.parseFunctionCall(v)

	default:
		// For unsupported types, return string representation
		return fmt.Sprintf("%v", expr)
	}
}

// parseArrayValue parses array expressions.
func (ap *AttributeParser) parseArrayValue(arr *pslast.ArrayExpression) []interface{} {
	if arr == nil || len(arr.Elements) == 0 {
		return nil
	}

	result := make([]interface{}, 0, len(arr.Elements))
	for _, val := range arr.Elements {
		if val != nil {
			parsed := ap.parseArgumentValue(val)
			if parsed != nil {
				result = append(result, parsed)
			}
		}
	}

	return result
}

// parseFunctionCall parses function call expressions like autoincrement(), now(), etc.
func (ap *AttributeParser) parseFunctionCall(fn *pslast.FunctionCall) map[string]interface{} {
	if fn == nil {
		return nil
	}

	result := map[string]interface{}{
		"function": fn.Name,
	}

	if fn.Arguments != nil && len(fn.Arguments.Arguments) > 0 {
		var args []interface{}
		for _, arg := range fn.Arguments.Arguments {
			if arg != nil && arg.Value != nil {
				args = append(args, ap.parseArgumentValue(arg.Value))
			}
		}
		result["args"] = args
	}

	return result
}

// ParseRelationAttribute parses @relation attribute and extracts relation metadata.
func (ap *AttributeParser) ParseRelationAttribute(pslField *pslast.Field) *domain.Relation {
	// Find @relation attribute
	var relationAttr *pslast.Attribute
	for _, attr := range pslField.Attributes {
		if attr != nil && attr.GetName() == "relation" {
			relationAttr = attr
			break
		}
	}

	if relationAttr == nil {
		return nil
	}

	relation := &domain.Relation{
		Name: pslField.GetName(),
	}

	// Parse relation arguments
	if relationAttr.Arguments != nil {
		for _, arg := range relationAttr.Arguments.Arguments {
			if arg == nil || arg.Name == nil {
				continue
			}

			argName := arg.Name.Name
			switch argName {
			case "fields":
				relation.FromFields = ap.extractFieldNames(arg.Value)

			case "references":
				relation.ToFields = ap.extractFieldNames(arg.Value)

			case "name":
				if strVal, ok := arg.Value.(*pslast.StringValue); ok {
					relation.Name = strVal.Value
				}

			case "onDelete":
				if strVal, ok := arg.Value.(*pslast.StringValue); ok {
					relation.OnDelete = domain.ReferentialAction(strVal.Value)
				}

			case "onUpdate":
				if strVal, ok := arg.Value.(*pslast.StringValue); ok {
					relation.OnUpdate = domain.ReferentialAction(strVal.Value)
				}
			}
		}
	}

	return relation
}

// extractFieldNames extracts field names from an array expression.
func (ap *AttributeParser) extractFieldNames(expr pslast.Expression) []string {
	arrExpr, ok := expr.(*pslast.ArrayExpression)
	if !ok || len(arrExpr.Elements) == 0 {
		return nil
	}

	var fields []string
	for _, val := range arrExpr.Elements {
		// Field references in arrays are typically identifiers
		if strVal := ap.extractIdentifier(val); strVal != "" {
			fields = append(fields, strVal)
		}
	}

	return fields
}

// extractIdentifier extracts string from identifier or string value.
func (ap *AttributeParser) extractIdentifier(expr pslast.Expression) string {
	switch v := expr.(type) {
	case *pslast.StringValue:
		return v.Value
	default:
		// Try to get string representation
		str := fmt.Sprint(expr)
		// Remove any wrapper formatting
		str = strings.Trim(str, "&{}")
		return str
	}
}

// GetDefaultValue extracts default value from field attributes.
func (ap *AttributeParser) GetDefaultValue(pslField *pslast.Field) interface{} {
	for _, attr := range pslField.Attributes {
		if attr != nil && attr.GetName() == "default" {
			if attr.Arguments != nil && len(attr.Arguments.Arguments) > 0 {
				firstArg := attr.Arguments.Arguments[0]
				if firstArg != nil && firstArg.Value != nil {
					return ap.parseArgumentValue(firstArg.Value)
				}
			}
		}
	}
	return nil
}

// IsUnique checks if field has @unique attribute.
func (ap *AttributeParser) IsUnique(pslField *pslast.Field) bool {
	for _, attr := range pslField.Attributes {
		if attr != nil && attr.GetName() == "unique" {
			return true
		}
	}
	return false
}

// IsID checks if field has @id attribute.
func (ap *AttributeParser) IsID(pslField *pslast.Field) bool {
	for _, attr := range pslField.Attributes {
		if attr != nil && attr.GetName() == "id" {
			return true
		}
	}
	return false
}

// IsUpdatedAt checks if field has @updatedAt attribute.
func (ap *AttributeParser) IsUpdatedAt(pslField *pslast.Field) bool {
	for _, attr := range pslField.Attributes {
		if attr != nil && attr.GetName() == "updatedAt" {
			return true
		}
	}
	return false
}

// GetMappedName extracts @map() value for field or @@map() for model.
func (ap *AttributeParser) GetMappedName(attrs []interface{}) string {
	for _, attr := range attrs {
		switch a := attr.(type) {
		case *pslast.Attribute:
			if a != nil && a.GetName() == "map" {
				if a.Arguments != nil && len(a.Arguments.Arguments) > 0 {
					if strVal, ok := a.Arguments.Arguments[0].Value.(*pslast.StringValue); ok {
						return strVal.Value
					}
				}
			}
		case *pslast.BlockAttribute:
			if a != nil && a.GetName() == "map" {
				if a.Arguments != nil && len(a.Arguments.Arguments) > 0 {
					if strVal, ok := a.Arguments.Arguments[0].Value.(*pslast.StringValue); ok {
						return strVal.Value
					}
				}
			}
		}
	}
	return ""
}
