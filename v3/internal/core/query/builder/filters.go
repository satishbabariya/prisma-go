// Package builder implements filter construction helpers.
package builder

import "github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"

// Filter helpers for building conditions with a fluent API.

// Equals creates an equals condition.
func Equals(field string, value interface{}) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.Equals,
		Value:    value,
	}
}

// NotEquals creates a not equals condition.
func NotEquals(field string, value interface{}) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.NotEquals,
		Value:    value,
	}
}

// In creates an in condition.
func In(field string, values interface{}) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.In,
		Value:    values,
	}
}

// NotIn creates a not in condition.
func NotIn(field string, values interface{}) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.NotIn,
		Value:    values,
	}
}

// Lt creates a less than condition.
func Lt(field string, value interface{}) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.Lt,
		Value:    value,
	}
}

// Lte creates a less than or equal condition.
func Lte(field string, value interface{}) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.Lte,
		Value:    value,
	}
}

// Gt creates a greater than condition.
func Gt(field string, value interface{}) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.Gt,
		Value:    value,
	}
}

// Gte creates a greater than or equal condition.
func Gte(field string, value interface{}) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.Gte,
		Value:    value,
	}
}

// Contains creates a contains condition.
func Contains(field string, substring string) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.Contains,
		Value:    substring,
	}
}

// StartsWith creates a starts with condition.
func StartsWith(field string, prefix string) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.StartsWith,
		Value:    prefix,
	}
}

// EndsWith creates an ends with condition.
func EndsWith(field string, suffix string) domain.Condition {
	return domain.Condition{
		Field:    field,
		Operator: domain.EndsWith,
		Value:    suffix,
	}
}
