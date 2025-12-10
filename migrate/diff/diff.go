// Package diff provides schema comparison and diff generation.
package diff

// Change type constants
const (
	ChangeTypeCreateTable      = "CreateTable"
	ChangeTypeDropTable        = "DropTable"
	ChangeTypeAlterTable       = "AlterTable"
	ChangeTypeRedefineTable    = "RedefineTable"
	ChangeTypeAddColumn        = "AddColumn"
	ChangeTypeDropColumn       = "DropColumn"
	ChangeTypeAlterColumn      = "AlterColumn"
	ChangeTypeCreateIndex      = "CreateIndex"
	ChangeTypeDropIndex        = "DropIndex"
	ChangeTypeRenameIndex      = "RenameIndex"
	ChangeTypeCreateForeignKey = "CreateForeignKey"
	ChangeTypeDropForeignKey   = "DropForeignKey"
	ChangeTypeRenameForeignKey = "RenameForeignKey"
)

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
	// For rename operations
	OldName string // Old name for RenameIndex, RenameForeignKey
	NewName string // New name for RenameIndex, RenameForeignKey
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
	Action  string      // "CREATE", "DROP", "ALTER", "REDEFINE"
	Model   interface{} // Can be *ast.Model
	Changes []Change
}

// MigrationPair is a helper type for tracking previous/next schema elements
// T should be a pointer type (e.g., *introspect.Table)
type MigrationPair[T any] struct {
	Previous T
	Next     T
}

// NewMigrationPair creates a new MigrationPair
func NewMigrationPair[T any](previous, next T) MigrationPair[T] {
	return MigrationPair[T]{
		Previous: previous,
		Next:     next,
	}
}

// HasBoth returns true if both Previous and Next are non-nil (for pointer types)
func HasBoth[T comparable](p MigrationPair[T]) bool {
	var zero T
	return p.Previous != zero && p.Next != zero
}

// HasPrevious returns true if Previous is non-nil (for pointer types)
func HasPrevious[T comparable](p MigrationPair[T]) bool {
	var zero T
	return p.Previous != zero
}

// HasNext returns true if Next is non-nil (for pointer types)
func HasNext[T comparable](p MigrationPair[T]) bool {
	var zero T
	return p.Next != zero
}
