// Package sqlgen provides rollback SQL generation for SQL Server
package sqlgen

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// generateRollbackAlterTable generates rollback SQL for ALTER TABLE changes
func (g *SQLServerMigrationGenerator) generateRollbackAlterTable(change diff.TableChange, dbSchema *introspect.DatabaseSchema) string {
	if len(change.Changes) == 0 {
		return ""
	}

	var sql strings.Builder

	// Process changes in reverse order
	for i := len(change.Changes) - 1; i >= 0; i-- {
		ch := change.Changes[i]
		switch ch.Type {
		case diff.ChangeTypeAddColumn:
			sql.WriteString(fmt.Sprintf("ALTER TABLE [%s] DROP COLUMN [%s];\n",
				change.Name, ch.Column))

		case diff.ChangeTypeDropColumn:
			// Rollback: add column back
			if ch.ColumnMetadata != nil {
				colDef := g.generateColumnDefinitionFromMetadata(ch.ColumnMetadata, ch.Column)
				sql.WriteString(fmt.Sprintf("ALTER TABLE [%s] ADD [%s] %s;\n",
					change.Name, ch.Column, colDef))
			} else {
				sql.WriteString(fmt.Sprintf("-- TODO: Add back dropped column [%s].[%s] (metadata missing)\n",
					change.Name, ch.Column))
			}

		case diff.ChangeTypeAlterColumn:
			// Rollback: restore old column definition
			if ch.ColumnMetadata != nil && ch.ColumnMetadata.OldType != "" {
				colType := g.mapPrismaTypeToSQLServer(ch.ColumnMetadata.OldType)
				if ch.ColumnMetadata.OldNullable != nil {
					if *ch.ColumnMetadata.OldNullable {
						sql.WriteString(fmt.Sprintf("ALTER TABLE [%s] ALTER COLUMN [%s] %s NULL;\n",
							change.Name, ch.Column, colType))
					} else {
						sql.WriteString(fmt.Sprintf("ALTER TABLE [%s] ALTER COLUMN [%s] %s NOT NULL;\n",
							change.Name, ch.Column, colType))
					}
				}
			}

		case diff.ChangeTypeCreateIndex:
			sql.WriteString(fmt.Sprintf("DROP INDEX [%s] ON [%s];\n",
				ch.Index, change.Name))

		case diff.ChangeTypeDropIndex:
			sql.WriteString(fmt.Sprintf("-- TODO: Recreate dropped index [%s] (requires schema history)\n", ch.Index))
			sql.WriteString(fmt.Sprintf("-- CREATE INDEX [%s] ON [%s] (...);\n", ch.Index, change.Name))

		case diff.ChangeTypeCreateForeignKey:
			sql.WriteString(fmt.Sprintf("ALTER TABLE [%s] DROP CONSTRAINT [%s];\n",
				change.Name, ch.Index))

		case diff.ChangeTypeDropForeignKey:
			sql.WriteString(fmt.Sprintf("-- TODO: Recreate dropped foreign key [%s] (requires schema history)\n", ch.Index))
			sql.WriteString(fmt.Sprintf("-- ALTER TABLE [%s] ADD CONSTRAINT [%s] FOREIGN KEY (...) REFERENCES ...;\n",
				change.Name, ch.Index))
		}
	}

	return sql.String()
}
