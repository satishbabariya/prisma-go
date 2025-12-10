// Package diff provides comprehensive schema comparison
package diff

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/migrate/converter"
	"github.com/satishbabariya/prisma-go/migrate/diff/flavour"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// Differ compares database schemas with advanced features
type Differ struct {
	provider string
	flavour  flavour.DifferFlavour
}

// NewDiffer creates a new Differ
func NewDiffer(provider string) (*Differ, error) {
	f, err := getFlavour(provider)
	if err != nil {
		return nil, err
	}

	return &Differ{
		provider: provider,
		flavour:  f,
	}, nil
}

// CompareSchemas compares two database schemas
func (d *Differ) CompareSchemas(source, target *introspect.DatabaseSchema) *DiffResult {
	result := &DiffResult{
		TablesToCreate: []TableChange{},
		TablesToAlter:  []TableChange{},
		TablesToDrop:   []TableChange{},
		Changes:        []Change{},
	}

	// Create the differ database
	db := NewDifferDatabase(source, target, d.flavour)

	// Process tables to create
	for _, table := range db.CreatedTables() {
		result.TablesToCreate = append(result.TablesToCreate, TableChange{
			Name:   table.Name,
			Action: "CREATE",
		})
		result.Changes = append(result.Changes, Change{
			Type:        ChangeTypeCreateTable,
			Table:       table.Name,
			Description: fmt.Sprintf("Create table '%s'", table.Name),
			IsSafe:      true,
		})
	}

	// Process tables to drop
	for _, table := range db.DroppedTables() {
		result.TablesToDrop = append(result.TablesToDrop, TableChange{
			Name:   table.Name,
			Action: "DROP",
		})
		result.Changes = append(result.Changes, Change{
			Type:        ChangeTypeDropTable,
			Table:       table.Name,
			Description: fmt.Sprintf("Drop table '%s'", table.Name),
			IsSafe:      false,
			Warnings:    []string{"Dropping table will delete all data"},
		})
	}

	// Process tables to alter
	for _, tablePair := range db.TablePairs() {
		tableName := tablePair.Name
		prevTable := tablePair.Table.Previous
		nextTable := tablePair.Table.Next

		// Check if table needs redefinition
		if db.ShouldRedefineTable(tableName) {
			result.TablesToAlter = append(result.TablesToAlter, TableChange{
				Name:   tableName,
				Action: "REDEFINE",
			})
			result.Changes = append(result.Changes, Change{
				Type:        ChangeTypeRedefineTable,
				Table:       tableName,
				Description: fmt.Sprintf("Redefine table '%s' (requires drop and recreate)", tableName),
				IsSafe:      false,
				Warnings:    []string{"Table redefinition will delete all data"},
			})
			continue
		}

		// Compare table changes
		tableDiffer := NewTableDiffer(prevTable, nextTable, db)
		changes := tableDiffer.Compare()

		if len(changes) > 0 {
			result.TablesToAlter = append(result.TablesToAlter, TableChange{
				Name:    tableName,
				Action:  "ALTER",
				Changes: changes,
			})
			result.Changes = append(result.Changes, changes...)
		}
	}

	// Order changes based on dependencies
	result.Changes = OrderChanges(result.Changes)

	return result
}

// CompareASTWithDatabase compares a Prisma schema AST with a database schema
func (d *Differ) CompareASTWithDatabase(schemaAST *ast.SchemaAst, dbSchema *introspect.DatabaseSchema) (*DiffResult, error) {
	// Convert AST to database schema format
	astDBSchema, err := converter.ConvertASTToDBSchema(schemaAST, d.provider)
	if err != nil {
		return nil, fmt.Errorf("failed to convert AST to database schema: %w", err)
	}

	// Compare the two database schemas
	return d.CompareSchemas(dbSchema, astDBSchema), nil
}

// getFlavour returns the appropriate flavour for the provider
func getFlavour(provider string) (flavour.DifferFlavour, error) {
	switch provider {
	case "postgresql", "postgres":
		return flavour.NewPostgresFlavour(), nil
	case "mysql":
		return flavour.NewMySQLFlavour(), nil
	case "sqlite":
		return flavour.NewSQLiteFlavour(), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
