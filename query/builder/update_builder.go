// Package builder provides update builder functionality.
package builder

import (
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// UpdateBuilder builds UPDATE queries
type UpdateBuilder struct {
	set   map[string]interface{}
	where *WhereBuilder
}

// NewUpdateBuilder creates a new update builder
func NewUpdateBuilder() *UpdateBuilder {
	return &UpdateBuilder{
		set: make(map[string]interface{}),
	}
}

// Set sets a field value
func (u *UpdateBuilder) Set(field string, value interface{}) *UpdateBuilder {
	u.set[field] = value
	return u
}

// Where adds a WHERE clause
func (u *UpdateBuilder) Where() *WhereBuilder {
	if u.where == nil {
		u.where = NewWhereBuilder()
	}
	return u.where
}

// GetSet returns the SET values
func (u *UpdateBuilder) GetSet() map[string]interface{} {
	return u.set
}

// GetWhere returns the WHERE clause
func (u *UpdateBuilder) GetWhere() *sqlgen.WhereClause {
	if u.where == nil {
		return nil
	}
	return u.where.Build()
}
