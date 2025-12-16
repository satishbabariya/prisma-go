// Package mapper implements result mapping to Go structs.
package mapper

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ResultMapper maps database results to Go structs.
type ResultMapper struct{}

// NewResultMapper creates a new result mapper.
func NewResultMapper() *ResultMapper {
	return &ResultMapper{}
}

// MapToStruct maps a single row to a struct.
func (m *ResultMapper) MapToStruct(row map[string]interface{}, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	destValue = destValue.Elem()
	if destValue.Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	destType := destValue.Type()

	// Map each field
	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		fieldValue := destValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get the column name (use json tag if present, otherwise lowercase field name)
		columnName := m.getColumnName(field)

		// Get the value from the row
		value, ok := row[columnName]
		if !ok {
			// Try case-insensitive match
			value, ok = m.findValueCaseInsensitive(row, columnName)
			if !ok {
				continue
			}
		}

		// Set the field value
		if err := m.setFieldValue(fieldValue, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
	}

	return nil
}

// MapToStructSlice maps multiple rows to a slice of structs.
func (m *ResultMapper) MapToStructSlice(rows []map[string]interface{}, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	destValue = destValue.Elem()
	if destValue.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	// Get the element type
	elemType := destValue.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	// Create a new slice
	newSlice := reflect.MakeSlice(destValue.Type(), 0, len(rows))

	for _, row := range rows {
		// Create a new struct instance
		elem := reflect.New(elemType)

		// Map the row to the struct
		if err := m.MapToStruct(row, elem.Interface()); err != nil {
			return err
		}

		// Append to slice
		if destValue.Type().Elem().Kind() == reflect.Ptr {
			newSlice = reflect.Append(newSlice, elem)
		} else {
			newSlice = reflect.Append(newSlice, elem.Elem())
		}
	}

	destValue.Set(newSlice)
	return nil
}

// ScanRow scans a sql.Row into a struct.
func (m *ResultMapper) ScanRow(rows *sql.Rows, dest interface{}) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Create value holders
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan the row
	if err := rows.Scan(valuePtrs...); err != nil {
		return fmt.Errorf("failed to scan row: %w", err)
	}

	// Create map
	row := make(map[string]interface{})
	for i, col := range columns {
		val := values[i]
		// Convert []byte to string
		if b, ok := val.([]byte); ok {
			row[col] = string(b)
		} else {
			row[col] = val
		}
	}

	// Map to struct
	return m.MapToStruct(row, dest)
}

// getColumnName gets the column name for a struct field.
func (m *ResultMapper) getColumnName(field reflect.StructField) string {
	// Check for json tag
	if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
		// Parse the tag (handle "name,omitempty" format)
		parts := strings.Split(tag, ",")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Check for db tag
	if tag := field.Tag.Get("db"); tag != "" && tag != "-" {
		return tag
	}

	// Default to lowercase field name
	return strings.ToLower(field.Name)
}

// findValueCaseInsensitive finds a value in the map case-insensitively.
func (m *ResultMapper) findValueCaseInsensitive(row map[string]interface{}, key string) (interface{}, bool) {
	lowerKey := strings.ToLower(key)
	for k, v := range row {
		if strings.ToLower(k) == lowerKey {
			return v, true
		}
	}
	return nil, false
}

// setFieldValue sets a field value with type conversion.
func (m *ResultMapper) setFieldValue(field reflect.Value, value interface{}) error {
	if value == nil {
		// Set zero value for nil
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	valueReflect := reflect.ValueOf(value)
	fieldType := field.Type()

	// Handle pointer fields
	if fieldType.Kind() == reflect.Ptr {
		if valueReflect.Type().AssignableTo(fieldType.Elem()) {
			ptr := reflect.New(fieldType.Elem())
			ptr.Elem().Set(valueReflect)
			field.Set(ptr)
			return nil
		}
		fieldType = fieldType.Elem()
	}

	// Direct assignment if types match
	if valueReflect.Type().AssignableTo(fieldType) {
		field.Set(valueReflect)
		return nil
	}

	// Type conversions
	switch fieldType.Kind() {
	case reflect.String:
		field.SetString(fmt.Sprintf("%v", value))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intVal int64
		switch v := value.(type) {
		case int64:
			intVal = v
		case int32:
			intVal = int64(v)
		case int:
			intVal = int64(v)
		case float64:
			intVal = int64(v)
		case string:
			return fmt.Errorf("cannot convert string to int")
		default:
			return fmt.Errorf("cannot convert %T to int", value)
		}
		field.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintVal uint64
		switch v := value.(type) {
		case uint64:
			uintVal = v
		case int64:
			uintVal = uint64(v)
		case float64:
			uintVal = uint64(v)
		default:
			return fmt.Errorf("cannot convert %T to uint", value)
		}
		field.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		var floatVal float64
		switch v := value.(type) {
		case float64:
			floatVal = v
		case float32:
			floatVal = float64(v)
		case int64:
			floatVal = float64(v)
		default:
			return fmt.Errorf("cannot convert %T to float", value)
		}
		field.SetFloat(floatVal)

	case reflect.Bool:
		var boolVal bool
		switch v := value.(type) {
		case bool:
			boolVal = v
		case int64:
			boolVal = v != 0
		default:
			return fmt.Errorf("cannot convert %T to bool", value)
		}
		field.SetBool(boolVal)

	case reflect.Struct:
		// Special handling for time.Time
		if fieldType == reflect.TypeOf(time.Time{}) {
			switch v := value.(type) {
			case time.Time:
				field.Set(reflect.ValueOf(v))
			case string:
				t, err := time.Parse(time.RFC3339, v)
				if err != nil {
					return fmt.Errorf("cannot parse time: %w", err)
				}
				field.Set(reflect.ValueOf(t))
			default:
				return fmt.Errorf("cannot convert %T to time.Time", value)
			}
		} else {
			return fmt.Errorf("unsupported struct type: %s", fieldType)
		}

	default:
		return fmt.Errorf("unsupported field type: %s", fieldType)
	}

	return nil
}
