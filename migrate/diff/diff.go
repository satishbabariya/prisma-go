// Package diff provides schema comparison and diff generation.
package diff

// DiffResult represents the differences between schema and database
type DiffResult struct {
	TablesToCreate []TableChange
	TablesToAlter  []TableChange
	TablesToDrop   []TableChange
	Changes        []Change
}

// Change represents a single schema change
type Change struct {
	Type        string
	Table       string
	Column      string
	Index       string
	Description string
	SQL         string
	IsSafe      bool
	Warnings    []string
}

// TableChange represents a change to a table
type TableChange struct {
	Name    string
	Action  string // "CREATE", "DROP", "ALTER"
	Model   interface{} // Can be *ast.Model
	Changes []Change
}

