// Package diff provides schema comparison and diffing functionality.
package diff

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// Differ compares schemas and generates diff results
type Differ struct {
	provider string
}

// NewDiffer creates a new schema differ
func NewDiffer(provider string) *Differ {
	return &Differ{provider: provider}
}

// Compare compares a Prisma schema AST with a database schema
func (d *Differ) Compare(prismaSchema *ast.SchemaAst, dbSchema *introspect.DatabaseSchema) (*DiffResult, error) {
	result := &DiffResult{
		TablesToCreate: []TableChange{},
		TablesToAlter:  []TableChange{},
		TablesToDrop:   []TableChange{},
		Changes:        []Change{},
	}

	// Build map of database tables
	dbTables := make(map[string]introspect.Table)
	for _, table := range dbSchema.Tables {
		dbTables[table.Name] = table
	}

	// Build map of Prisma models
	prismaModels := make(map[string]*ast.Model)
	for _, top := range prismaSchema.Tops {
		if model, ok := top.AsModel(); ok {
			// Convert model name to table name (snake_case)
			tableName := toSnakeCase(model.Name.Name)
			prismaModels[tableName] = model
		}
	}

	// Check for tables to create (in Prisma, not in DB)
	for tableName, model := range prismaModels {
		if _, exists := dbTables[tableName]; !exists {
			result.TablesToCreate = append(result.TablesToCreate, TableChange{
				Name:   tableName,
				Model:  model,
				Action: "CREATE",
			})
			result.Changes = append(result.Changes, Change{
				Type:        "CreateTable",
				Table:       tableName,
				Description: fmt.Sprintf("Create table '%s'", tableName),
				IsSafe:      true,
			})
		}
	}

	// Check for tables to drop (in DB, not in Prisma)
	for tableName, table := range dbTables {
		if _, exists := prismaModels[tableName]; !exists {
			result.TablesToDrop = append(result.TablesToDrop, TableChange{
				Name:   tableName,
				Action: "DROP",
			})
			result.Changes = append(result.Changes, Change{
				Type:        "DropTable",
				Table:       tableName,
				Description: fmt.Sprintf("Drop table '%s'", tableName),
				IsSafe:      false,
				Warnings:    []string{"Dropping table will delete all data"},
			})
		}
	}

	// Check for tables to alter (in both, but different)
	for tableName, model := range prismaModels {
		if dbTable, exists := dbTables[tableName]; exists {
			changes := d.compareTable(model, &dbTable, tableName)
			if len(changes) > 0 {
				result.TablesToAlter = append(result.TablesToAlter, TableChange{
					Name:    tableName,
					Model:   model,
					Action:  "ALTER",
					Changes: changes,
				})
				result.Changes = append(result.Changes, changes...)
			}
		}
	}

	return result, nil
}

// compareTable compares a Prisma model with a database table
func (d *Differ) compareTable(model *ast.Model, dbTable *introspect.Table, tableName string) []Change {
	var changes []Change

	// Build map of database columns
	dbColumns := make(map[string]introspect.Column)
	for _, col := range dbTable.Columns {
		dbColumns[col.Name] = col
	}

	// Build map of Prisma fields
	prismaFields := make(map[string]*ast.Field)
	for _, field := range model.Fields {
		// Skip relation fields
		if isRelationField(field) {
			continue
		}
		columnName := toSnakeCase(field.Name.Name)
		prismaFields[columnName] = field
	}

	// Check for columns to add
	for columnName, field := range prismaFields {
		if _, exists := dbColumns[columnName]; !exists {
			changes = append(changes, Change{
				Type:        "AddColumn",
				Table:       tableName,
				Column:      columnName,
				Description: fmt.Sprintf("Add column '%s.%s'", tableName, columnName),
				IsSafe:      true,
			})
		}
	}

	// Check for columns to drop
	for columnName := range dbColumns {
		if _, exists := prismaFields[columnName]; !exists {
			changes = append(changes, Change{
				Type:        "DropColumn",
				Table:       tableName,
				Column:      columnName,
				Description: fmt.Sprintf("Drop column '%s.%s'", tableName, columnName),
				IsSafe:      false,
				Warnings:    []string{"Dropping column will delete all data in that column"},
			})
		}
	}

	// Check for columns to alter
	for columnName, field := range prismaFields {
		if dbCol, exists := dbColumns[columnName]; exists {
			if colChanges := d.compareColumn(field, &dbCol, tableName, columnName); len(colChanges) > 0 {
				changes = append(changes, colChanges...)
			}
		}
	}

	// Check for index changes
	indexChanges := d.compareIndexes(model, dbTable, tableName)
	changes = append(changes, indexChanges...)

	return changes
}

// compareColumn compares a Prisma field with a database column
func (d *Differ) compareColumn(field *ast.Field, dbCol *introspect.Column, tableName, columnName string) []Change {
	var changes []Change

	// Compare type
	prismaType := getPrismaType(field)
	if !typesMatch(prismaType, dbCol.Type) {
		changes = append(changes, Change{
			Type:        "AlterColumn",
			Table:       tableName,
			Column:      columnName,
			Description: fmt.Sprintf("Change type of '%s.%s' from %s to %s", tableName, columnName, dbCol.Type, prismaType),
			IsSafe:      false,
			Warnings:    []string{"Changing column type may cause data loss"},
		})
	}

	// Compare nullable
	prismaIsOptional := field.Arity == ast.Optional
	if prismaIsOptional != dbCol.Nullable {
		action := "make nullable"
		if !prismaIsOptional {
			action = "make required"
		}
		changes = append(changes, Change{
			Type:        "AlterColumn",
			Table:       tableName,
			Column:      columnName,
			Description: fmt.Sprintf("%s column '%s.%s'", strings.Title(action), tableName, columnName),
			IsSafe:      prismaIsOptional, // Making nullable is safe, making required is not
		})
	}

	// Compare default value
	prismaDefault := getDefaultValue(field)
	if prismaDefault != dbCol.DefaultValue {
		changes = append(changes, Change{
			Type:        "AlterColumn",
			Table:       tableName,
			Column:      columnName,
			Description: fmt.Sprintf("Change default value of '%s.%s'", tableName, columnName),
			IsSafe:      true,
		})
	}

	return changes
}

// compareIndexes compares indexes between Prisma model and database table
func (d *Differ) compareIndexes(model *ast.Model, dbTable *introspect.Table, tableName string) []Change {
	var changes []Change

	// Build map of database indexes
	dbIndexes := make(map[string]introspect.Index)
	for _, idx := range dbTable.Indexes {
		dbIndexes[idx.Name] = idx
	}

	// Build map of Prisma indexes
	prismaIndexes := make(map[string][]string)
	for _, attr := range model.Attributes {
		if attr.Name.Name == "index" {
			// Extract fields from @@index([field1, field2])
			if len(attr.Arguments.Arguments) > 0 {
				arg := attr.Arguments.Arguments[0]
				if arr, ok := arg.Value.AsArray(); ok {
					var fields []string
					for _, elem := range arr.Elements {
						if fieldRef, ok := elem.AsConstantValue(); ok {
							fields = append(fields, toSnakeCase(fieldRef))
						}
					}
					indexName := fmt.Sprintf("%s_%s_idx", tableName, strings.Join(fields, "_"))
					prismaIndexes[indexName] = fields
				}
			}
		}
	}

	// Check for indexes to create
	for indexName, fields := range prismaIndexes {
		if _, exists := dbIndexes[indexName]; !exists {
			changes = append(changes, Change{
				Type:        "CreateIndex",
				Table:       tableName,
				Index:       indexName,
				Description: fmt.Sprintf("Create index '%s' on %s(%s)", indexName, tableName, strings.Join(fields, ", ")),
				IsSafe:      true,
			})
		}
	}

	// Check for indexes to drop
	for indexName := range dbIndexes {
		if _, exists := prismaIndexes[indexName]; !exists {
			changes = append(changes, Change{
				Type:        "DropIndex",
				Table:       tableName,
				Index:       indexName,
				Description: fmt.Sprintf("Drop index '%s'", indexName),
				IsSafe:      true,
			})
		}
	}

	return changes
}

// Helper functions

func isRelationField(field *ast.Field) bool {
	// Check if field has @relation attribute
	for _, attr := range field.Attributes {
		if attr.Name.Name == "relation" {
			return true
		}
	}
	
	// Check if field type is another model (starts with uppercase)
	fieldType := field.FieldType.(*ast.FieldType)
	if len(fieldType.Name) > 0 && fieldType.Name[0] >= 'A' && fieldType.Name[0] <= 'Z' {
		return true
	}
	
	return false
}

func getPrismaType(field *ast.Field) string {
	fieldType := field.FieldType.(*ast.FieldType)
	return fieldType.Name
}

func getDefaultValue(field *ast.Field) *string {
	for _, attr := range field.Attributes {
		if attr.Name.Name == "default" {
			if len(attr.Arguments.Arguments) > 0 {
				arg := attr.Arguments.Arguments[0]
				if val, ok := arg.Value.AsConstantValue(); ok {
					return &val
				}
			}
		}
	}
	return nil
}

func typesMatch(prismaType, dbType string) bool {
	// Normalize types for comparison
	prismaType = strings.ToUpper(prismaType)
	dbType = strings.ToUpper(dbType)
	
	// Simple type mappings
	mappings := map[string][]string{
		"INT":       {"INTEGER", "INT4", "SERIAL"},
		"BIGINT":    {"INT8", "BIGSERIAL"},
		"STRING":    {"VARCHAR", "TEXT", "CHARACTER VARYING"},
		"BOOLEAN":   {"BOOL"},
		"DATETIME":  {"TIMESTAMP", "TIMESTAMP WITHOUT TIME ZONE"},
		"FLOAT":     {"DOUBLE PRECISION", "FLOAT8", "REAL"},
		"DECIMAL":   {"NUMERIC"},
	}
	
	for key, variants := range mappings {
		if prismaType == key {
			for _, variant := range variants {
				if strings.HasPrefix(dbType, variant) {
					return true
				}
			}
		}
	}
	
	return prismaType == dbType
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

