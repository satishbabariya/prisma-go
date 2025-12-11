// Package columns provides type-safe column expressions for query building.
package columns

import (
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// Column represents a database column with type safety
type Column interface {
	// Name returns the column name
	Name() string
	// Table returns the table name
	Table() string
	// BuildCondition builds a condition for this column
	BuildCondition(operator string, value interface{}) sqlgen.Condition
}

// BaseColumn is the base implementation for all column types
type BaseColumn struct {
	name  string
	table string
}

// Name returns the column name
func (c BaseColumn) Name() string {
	return c.name
}

// Table returns the table name
func (c BaseColumn) Table() string {
	return c.table
}

// BuildCondition builds a condition for this column
func (c BaseColumn) BuildCondition(operator string, value interface{}) sqlgen.Condition {
	return sqlgen.Condition{
		Field:    c.name,
		Operator: operator,
		Value:    value,
	}
}

// IntColumn represents an integer column
type IntColumn struct {
	BaseColumn
}

// NewIntColumn creates a new IntColumn
func NewIntColumn(table, name string) IntColumn {
	return IntColumn{
		BaseColumn: BaseColumn{
			name:  name,
			table: table,
		},
	}
}

// EQ creates an equality condition
func (c IntColumn) EQ(value int) Condition {
	return Condition{
		Column:   c,
		Operator: "=",
		Value:    value,
	}
}

// NOT_EQ creates a not-equal condition
func (c IntColumn) NOT_EQ(value int) Condition {
	return Condition{
		Column:   c,
		Operator: "!=",
		Value:    value,
	}
}

// GT creates a greater-than condition
func (c IntColumn) GT(value int) Condition {
	return Condition{
		Column:   c,
		Operator: ">",
		Value:    value,
	}
}

// GTE creates a greater-than-or-equal condition
func (c IntColumn) GTE(value int) Condition {
	return Condition{
		Column:   c,
		Operator: ">=",
		Value:    value,
	}
}

// LT creates a less-than condition
func (c IntColumn) LT(value int) Condition {
	return Condition{
		Column:   c,
		Operator: "<",
		Value:    value,
	}
}

// LTE creates a less-than-or-equal condition
func (c IntColumn) LTE(value int) Condition {
	return Condition{
		Column:   c,
		Operator: "<=",
		Value:    value,
	}
}

// IN creates an IN condition
func (c IntColumn) IN(values []int) Condition {
	interfaceValues := make([]interface{}, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return Condition{
		Column:   c,
		Operator: "IN",
		Value:    interfaceValues,
	}
}

// NOT_IN creates a NOT IN condition
func (c IntColumn) NOT_IN(values []int) Condition {
	interfaceValues := make([]interface{}, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return Condition{
		Column:   c,
		Operator: "NOT IN",
		Value:    interfaceValues,
	}
}

// StringColumn represents a string column
type StringColumn struct {
	BaseColumn
}

// NewStringColumn creates a new StringColumn
func NewStringColumn(table, name string) StringColumn {
	return StringColumn{
		BaseColumn: BaseColumn{
			name:  name,
			table: table,
		},
	}
}

// EQ creates an equality condition
func (c StringColumn) EQ(value string) Condition {
	return Condition{
		Column:   c,
		Operator: "=",
		Value:    value,
	}
}

// NOT_EQ creates a not-equal condition
func (c StringColumn) NOT_EQ(value string) Condition {
	return Condition{
		Column:   c,
		Operator: "!=",
		Value:    value,
	}
}

// Contains creates a LIKE condition with wildcards
func (c StringColumn) Contains(value string) Condition {
	return Condition{
		Column:   c,
		Operator: "LIKE",
		Value:    "%" + value + "%",
	}
}

// StartsWith creates a LIKE condition that matches the start
func (c StringColumn) StartsWith(value string) Condition {
	return Condition{
		Column:   c,
		Operator: "LIKE",
		Value:    value + "%",
	}
}

// EndsWith creates a LIKE condition that matches the end
func (c StringColumn) EndsWith(value string) Condition {
	return Condition{
		Column:   c,
		Operator: "LIKE",
		Value:    "%" + value,
	}
}

// IN creates an IN condition
func (c StringColumn) IN(values []string) Condition {
	interfaceValues := make([]interface{}, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return Condition{
		Column:   c,
		Operator: "IN",
		Value:    interfaceValues,
	}
}

// NOT_IN creates a NOT IN condition
func (c StringColumn) NOT_IN(values []string) Condition {
	interfaceValues := make([]interface{}, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return Condition{
		Column:   c,
		Operator: "NOT IN",
		Value:    interfaceValues,
	}
}

// NullableStringColumn represents a nullable string column
type NullableStringColumn struct {
	BaseColumn
}

// NewNullableStringColumn creates a new NullableStringColumn
func NewNullableStringColumn(table, name string) NullableStringColumn {
	return NullableStringColumn{
		BaseColumn: BaseColumn{
			name:  name,
			table: table,
		},
	}
}

// EQ creates an equality condition
func (c NullableStringColumn) EQ(value *string) Condition {
	return Condition{
		Column:   c,
		Operator: "=",
		Value:    value,
	}
}

// NOT_EQ creates a not-equal condition
func (c NullableStringColumn) NOT_EQ(value *string) Condition {
	return Condition{
		Column:   c,
		Operator: "!=",
		Value:    value,
	}
}

// Contains creates a LIKE condition with wildcards
func (c NullableStringColumn) Contains(value string) Condition {
	return Condition{
		Column:   c,
		Operator: "LIKE",
		Value:    "%" + value + "%",
	}
}

// IsNull creates an IS NULL condition
func (c NullableStringColumn) IsNull() Condition {
	return Condition{
		Column:   c,
		Operator: "IS NULL",
		Value:    nil,
	}
}

// IsNotNull creates an IS NOT NULL condition
func (c NullableStringColumn) IsNotNull() Condition {
	return Condition{
		Column:   c,
		Operator: "IS NOT NULL",
		Value:    nil,
	}
}

// BoolColumn represents a boolean column
type BoolColumn struct {
	BaseColumn
}

// NewBoolColumn creates a new BoolColumn
func NewBoolColumn(table, name string) BoolColumn {
	return BoolColumn{
		BaseColumn: BaseColumn{
			name:  name,
			table: table,
		},
	}
}

// EQ creates an equality condition
func (c BoolColumn) EQ(value bool) Condition {
	return Condition{
		Column:   c,
		Operator: "=",
		Value:    value,
	}
}

// NOT_EQ creates a not-equal condition
func (c BoolColumn) NOT_EQ(value bool) Condition {
	return Condition{
		Column:   c,
		Operator: "!=",
		Value:    value,
	}
}

// DateTimeColumn represents a datetime column
type DateTimeColumn struct {
	BaseColumn
}

// NewDateTimeColumn creates a new DateTimeColumn
func NewDateTimeColumn(table, name string) DateTimeColumn {
	return DateTimeColumn{
		BaseColumn: BaseColumn{
			name:  name,
			table: table,
		},
	}
}

// EQ creates an equality condition
func (c DateTimeColumn) EQ(value interface{}) Condition {
	return Condition{
		Column:   c,
		Operator: "=",
		Value:    value,
	}
}

// NOT_EQ creates a not-equal condition
func (c DateTimeColumn) NOT_EQ(value interface{}) Condition {
	return Condition{
		Column:   c,
		Operator: "!=",
		Value:    value,
	}
}

// GT creates a greater-than condition
func (c DateTimeColumn) GT(value interface{}) Condition {
	return Condition{
		Column:   c,
		Operator: ">",
		Value:    value,
	}
}

// GTE creates a greater-than-or-equal condition
func (c DateTimeColumn) GTE(value interface{}) Condition {
	return Condition{
		Column:   c,
		Operator: ">=",
		Value:    value,
	}
}

// LT creates a less-than condition
func (c DateTimeColumn) LT(value interface{}) Condition {
	return Condition{
		Column:   c,
		Operator: "<",
		Value:    value,
	}
}

// LTE creates a less-than-or-equal condition
func (c DateTimeColumn) LTE(value interface{}) Condition {
	return Condition{
		Column:   c,
		Operator: "<=",
		Value:    value,
	}
}

// Condition represents a column-based condition
type Condition struct {
	Column   Column
	Operator string
	Value    interface{}
}

// ToSQLCondition converts a Condition to sqlgen.Condition
func (c Condition) ToSQLCondition() sqlgen.Condition {
	return sqlgen.Condition{
		Field:    c.Column.Name(),
		Operator: c.Operator,
		Value:    c.Value,
	}
}

// AND combines multiple conditions with AND
func AND(conditions ...Condition) []Condition {
	return conditions
}

// OR combines multiple conditions with OR
func OR(conditions ...Condition) []Condition {
	// This will be handled by the WhereBuilder
	return conditions
}

