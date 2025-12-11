// Package builder provides CTE (Common Table Expression) building functionality
package builder

import (
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// CTE represents a Common Table Expression
type CTE struct {
	Name      string
	Query     *sqlgen.Query
	Columns   []string // Optional column names for the CTE
	Recursive bool     // Whether this is a RECURSIVE CTE
}

// CTEBuilder builds CTEs
type CTEBuilder struct {
	ctes []CTE
}

// NewCTEBuilder creates a new CTE builder
func NewCTEBuilder() *CTEBuilder {
	return &CTEBuilder{
		ctes: []CTE{},
	}
}

// With adds a CTE to the builder
func (c *CTEBuilder) With(name string, query *sqlgen.Query) *CTEBuilder {
	c.ctes = append(c.ctes, CTE{
		Name:    name,
		Query:   query,
		Columns: []string{},
	})
	return c
}

// WithColumns adds a CTE with explicit column names
func (c *CTEBuilder) WithColumns(name string, columns []string, query *sqlgen.Query) *CTEBuilder {
	c.ctes = append(c.ctes, CTE{
		Name:    name,
		Query:   query,
		Columns: columns,
	})
	return c
}

// WithRecursive marks the CTE as recursive
func (c *CTEBuilder) WithRecursive(name string, query *sqlgen.Query) *CTEBuilder {
	c.ctes = append(c.ctes, CTE{
		Name:      name,
		Query:     query,
		Columns:   []string{},
		Recursive: true,
	})
	return c
}

// WithRecursiveColumns adds a recursive CTE with explicit column names
func (c *CTEBuilder) WithRecursiveColumns(name string, columns []string, query *sqlgen.Query) *CTEBuilder {
	c.ctes = append(c.ctes, CTE{
		Name:      name,
		Query:     query,
		Columns:   columns,
		Recursive: true,
	})
	return c
}

// Build returns the list of CTEs
func (c *CTEBuilder) Build() []CTE {
	return c.ctes
}

// IsRecursive returns true if any CTE is recursive
func (c *CTEBuilder) IsRecursive() bool {
	for _, cte := range c.ctes {
		if cte.Recursive {
			return true
		}
	}
	return false
}
