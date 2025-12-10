// Package types provides runtime types for Prisma Go.
package types

import "time"

// DateTime represents a timestamp
type DateTime = time.Time

// Json represents a JSON value
type Json = interface{}

// Decimal represents a decimal number
type Decimal struct {
	value string
}

// NewDecimal creates a new decimal from string
func NewDecimal(value string) Decimal {
	return Decimal{value: value}
}

// String returns the string representation
func (d Decimal) String() string {
	return d.value
}

// Filter represents a query filter
type Filter interface {
	isFilter()
}

// WhereInput is the base for all where inputs
type WhereInput struct{}

func (WhereInput) isFilter() {}
