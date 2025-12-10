// Package sqlgen provides WHERE clause structures.
package sqlgen

// WhereClause represents a WHERE condition (can be nested)
type WhereClause struct {
	Conditions []Condition
	Groups     []*WhereClause // Nested WHERE clauses for AND/OR/NOT
	Operator   string         // "AND" or "OR"
	IsNot      bool           // true for NOT conditions
}

// Condition represents a single filter condition
type Condition struct {
	Field    string
	Operator string // "=", "!=", ">", "<", ">=", "<=", "IN", "NOT IN", "LIKE", "IS NULL", "IS NOT NULL"
	Value    interface{}
}

// NewWhereClause creates a new WHERE clause
func NewWhereClause() *WhereClause {
	return &WhereClause{
		Conditions: []Condition{},
		Groups:     []*WhereClause{},
		Operator:   "AND",
		IsNot:      false,
	}
}

// AddCondition adds a condition to the WHERE clause
func (w *WhereClause) AddCondition(condition Condition) {
	w.Conditions = append(w.Conditions, condition)
}

// AddGroup adds a nested WHERE clause
func (w *WhereClause) AddGroup(group *WhereClause) {
	w.Groups = append(w.Groups, group)
}

// SetOperator sets the logical operator
func (w *WhereClause) SetOperator(op string) {
	w.Operator = op
}

// SetNot sets the NOT flag
func (w *WhereClause) SetNot(isNot bool) {
	w.IsNot = isNot
}

// IsEmpty returns true if the WHERE clause is empty
func (w *WhereClause) IsEmpty() bool {
	return len(w.Conditions) == 0 && len(w.Groups) == 0
}

// HasConditions returns true if there are any conditions
func (w *WhereClause) HasConditions() bool {
	return len(w.Conditions) > 0
}

// HasGroups returns true if there are any nested groups
func (w *WhereClause) HasGroups() bool {
	return len(w.Groups) > 0
}
