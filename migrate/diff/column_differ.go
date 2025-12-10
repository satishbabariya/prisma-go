// Package diff provides column comparison logic
package diff

import (
	"encoding/json"
	"strings"

	"github.com/satishbabariya/prisma-go/migrate/diff/flavour"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// ColumnChanges tracks all changes to a column
type ColumnChanges struct {
	TypeChanged          bool
	NullableChanged      bool
	DefaultChanged       bool
	AutoIncrementChanged bool
	TypeChange           *flavour.ColumnTypeChange
}

// allColumnChanges detects all changes between two columns
func allColumnChanges(prev, next *introspect.Column, f flavour.DifferFlavour) *ColumnChanges {
	changes := &ColumnChanges{}

	// Check type change
	typeChange := f.ColumnTypeChange(prev, next)
	if typeChange != nil {
		changes.TypeChanged = true
		changes.TypeChange = typeChange
	}

	// Check nullable change
	if prev.Nullable != next.Nullable {
		changes.NullableChanged = true
	}

	// Check default change
	if !defaultsMatch(prev, next, f) {
		changes.DefaultChanged = true
	}

	// Check auto-increment change
	if prev.AutoIncrement != next.AutoIncrement {
		changes.AutoIncrementChanged = true
	}

	return changes
}

// defaultsMatch compares default values between two columns
func defaultsMatch(prev, next *introspect.Column, f flavour.DifferFlavour) bool {
	// Both nil
	if prev.DefaultValue == nil && next.DefaultValue == nil {
		return true
	}

	// One nil, one not
	if prev.DefaultValue == nil || next.DefaultValue == nil {
		return false
	}

	prevVal := *prev.DefaultValue
	nextVal := *next.DefaultValue

	// Direct match
	if prevVal == nextVal {
		return true
	}

	// Check for JSON defaults - parse and compare
	if isJSONType(prev.Type) || isJSONType(next.Type) {
		return jsonDefaultsMatch(prevVal, nextVal)
	}

	// Check for DateTime defaults - treat as equal if both are datetime functions
	if isDateTimeType(prev.Type) || isDateTimeType(next.Type) {
		// DateTime defaults like now(), CURRENT_TIMESTAMP, etc. are considered equal
		// if they're both datetime functions
		if isDateTimeFunction(prevVal) && isDateTimeFunction(nextVal) {
			return true
		}
		// Otherwise do string comparison
		return prevVal == nextVal
	}

	// Check for enum defaults - compare string values
	if isEnumType(prev.Type) || isEnumType(next.Type) {
		// Remove quotes if present
		prevClean := strings.Trim(prevVal, `"'`)
		nextClean := strings.Trim(nextVal, `"'`)
		return prevClean == nextClean
	}

	// Check for numeric defaults - compare ignoring type differences
	if isNumericType(prev.Type) && isNumericType(next.Type) {
		// Remove whitespace and compare
		prevClean := strings.TrimSpace(prevVal)
		nextClean := strings.TrimSpace(nextVal)
		return prevClean == nextClean
	}

	// Default: string comparison
	return prevVal == nextVal
}

// jsonDefaultsMatch compares JSON default values by parsing them
func jsonDefaultsMatch(prev, next string) bool {
	var prevJSON, nextJSON interface{}

	if err := json.Unmarshal([]byte(prev), &prevJSON); err != nil {
		// If not valid JSON, fall back to string comparison
		return prev == next
	}

	if err := json.Unmarshal([]byte(next), &nextJSON); err != nil {
		// If not valid JSON, fall back to string comparison
		return prev == next
	}

	// Compare parsed JSON values
	return compareJSONValues(prevJSON, nextJSON)
}

// compareJSONValues compares two JSON values
func compareJSONValues(a, b interface{}) bool {
	// Simple equality check - could be enhanced
	aBytes, err1 := json.Marshal(a)
	bBytes, err2 := json.Marshal(b)

	if err1 != nil || err2 != nil {
		return false
	}

	return string(aBytes) == string(bBytes)
}

// Helper functions to detect column types
func isJSONType(colType string) bool {
	upper := strings.ToUpper(colType)
	return strings.Contains(upper, "JSON")
}

func isDateTimeType(colType string) bool {
	upper := strings.ToUpper(colType)
	return strings.Contains(upper, "TIMESTAMP") ||
		strings.Contains(upper, "DATETIME") ||
		strings.Contains(upper, "DATE") ||
		strings.Contains(upper, "TIME")
}

func isEnumType(colType string) bool {
	upper := strings.ToUpper(colType)
	return strings.Contains(upper, "ENUM")
}

func isNumericType(colType string) bool {
	upper := strings.ToUpper(colType)
	return strings.Contains(upper, "INT") ||
		strings.Contains(upper, "FLOAT") ||
		strings.Contains(upper, "DOUBLE") ||
		strings.Contains(upper, "DECIMAL") ||
		strings.Contains(upper, "NUMERIC") ||
		strings.Contains(upper, "REAL")
}

func isDateTimeFunction(val string) bool {
	upper := strings.ToUpper(strings.TrimSpace(val))
	return strings.HasPrefix(upper, "NOW()") ||
		strings.HasPrefix(upper, "CURRENT_TIMESTAMP") ||
		strings.HasPrefix(upper, "CURRENT_DATE") ||
		strings.HasPrefix(upper, "CURRENT_TIME") ||
		strings.Contains(upper, "NOW()") ||
		strings.Contains(upper, "CURRENT_TIMESTAMP")
}

// DiffersInSomething returns true if any change was detected
func (c *ColumnChanges) DiffersInSomething() bool {
	return c.TypeChanged || c.NullableChanged || c.DefaultChanged || c.AutoIncrementChanged
}
