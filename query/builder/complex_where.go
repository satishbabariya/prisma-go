// Package builder provides complex WHERE clause building with AND/OR/NOT support.
package builder

import (
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// AND creates a new WHERE clause group with AND operator
func (w *WhereBuilder) AND(builders ...*WhereBuilder) *WhereBuilder {
	group := sqlgen.NewWhereClause()
	group.SetOperator("AND")
	
	for _, builder := range builders {
		if builder != nil {
			subClause := builder.Build()
			if subClause != nil && !subClause.IsEmpty() {
				group.AddGroup(subClause)
			}
		}
	}
	
	// Add this group to the current builder
	if w.whereClause == nil {
		w.whereClause = sqlgen.NewWhereClause()
	}
	w.whereClause.AddGroup(group)
	
	return w
}

// OR creates a new WHERE clause group with OR operator
func (w *WhereBuilder) OR(builders ...*WhereBuilder) *WhereBuilder {
	group := sqlgen.NewWhereClause()
	group.SetOperator("OR")
	
	for _, builder := range builders {
		if builder != nil {
			subClause := builder.Build()
			if subClause != nil && !subClause.IsEmpty() {
				group.AddGroup(subClause)
			}
		}
	}
	
	// Add this group to the current builder
	if w.whereClause == nil {
		w.whereClause = sqlgen.NewWhereClause()
	}
	w.whereClause.AddGroup(group)
	
	return w
}

// NOT negates a WHERE clause
func (w *WhereBuilder) NOT(builder *WhereBuilder) *WhereBuilder {
	if builder != nil {
		subClause := builder.Build()
		if subClause != nil && !subClause.IsEmpty() {
			subClause.SetNot(true)
			
			if w.whereClause == nil {
				w.whereClause = sqlgen.NewWhereClause()
			}
			w.whereClause.AddGroup(subClause)
		}
	}
	
	return w
}

// NewSubWhereBuilder creates a new independent WHERE builder for use in AND/OR/NOT
func NewSubWhereBuilder() *WhereBuilder {
	return &WhereBuilder{
		conditions: []sqlgen.Condition{},
		operator:   "AND",
		whereClause: nil,
	}
}

