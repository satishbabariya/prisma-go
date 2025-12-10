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
	// ColumnMetadata contains column information for AddColumn and AlterColumn changes
	ColumnMetadata *ColumnMetadata
}

// ColumnMetadata contains column definition information
type ColumnMetadata struct {
	Type          string  // Column type (e.g., "INTEGER", "VARCHAR(255)", "TEXT")
	Nullable      bool    // Whether column allows NULL
	DefaultValue  *string // Default value (nil if no default)
	AutoIncrement bool    // Whether column is auto-increment
	OldType       string  // For AlterColumn: the old type (empty for AddColumn)
	OldNullable   *bool   // For AlterColumn: the old nullable state (nil for AddColumn)
}

// TableChange represents a change to a table
type TableChange struct {
	Name    string
	Action  string      // "CREATE", "DROP", "ALTER"
	Model   interface{} // Can be *ast.Model
	Changes []Change
}
