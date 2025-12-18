package commands

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
)

// addColumnChange implements domain.Change for ADD COLUMN.
type addColumnChange struct {
	TableName string
	Column    domain.Column
}

func (c *addColumnChange) Type() domain.ChangeType { return domain.AddColumn }
func (c *addColumnChange) Description() string {
	return fmt.Sprintf("Add column %s.%s (%s)", c.TableName, c.Column.Name, c.Column.Type)
}
func (c *addColumnChange) IsDestructive() bool { return false }
func (c *addColumnChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	nullable := "NOT NULL"
	if c.Column.IsNullable {
		nullable = "NULL"
	}
	defaultClause := ""
	if c.Column.DefaultValue != "" {
		defaultClause = fmt.Sprintf(" DEFAULT %s", c.Column.DefaultValue)
	}
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s %s%s",
		c.TableName, c.Column.Name, c.Column.Type, nullable, defaultClause)
	return []string{sql}, nil
}

// dropColumnChange implements domain.Change for DROP COLUMN.
type dropColumnChange struct {
	TableName  string
	ColumnName string
}

func (c *dropColumnChange) Type() domain.ChangeType { return domain.DropColumn }
func (c *dropColumnChange) Description() string {
	return fmt.Sprintf("Drop column %s.%s", c.TableName, c.ColumnName)
}
func (c *dropColumnChange) IsDestructive() bool { return true }
func (c *dropColumnChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", c.TableName, c.ColumnName)
	return []string{sql}, nil
}

// alterColumnChange implements domain.Change for ALTER COLUMN.
type alterColumnChange struct {
	TableName string
	OldColumn domain.Column
	NewColumn domain.Column
}

func (c *alterColumnChange) Type() domain.ChangeType { return domain.AlterColumn }
func (c *alterColumnChange) Description() string {
	return fmt.Sprintf("Alter column %s.%s: %s → %s", c.TableName, c.NewColumn.Name, c.OldColumn.Type, c.NewColumn.Type)
}
func (c *alterColumnChange) IsDestructive() bool {
	// Type changes are potentially destructive
	return c.OldColumn.Type != c.NewColumn.Type
}
func (c *alterColumnChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	var sqls []string

	// Alter type if changed
	if c.OldColumn.Type != c.NewColumn.Type {
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s",
			c.TableName, c.NewColumn.Name, c.NewColumn.Type)
		sqls = append(sqls, sql)
	}

	// Alter nullability if changed
	if c.OldColumn.IsNullable != c.NewColumn.IsNullable {
		action := "DROP NOT NULL"
		if !c.NewColumn.IsNullable {
			action = "SET NOT NULL"
		}
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s",
			c.TableName, c.NewColumn.Name, action)
		sqls = append(sqls, sql)
	}

	return sqls, nil
}
