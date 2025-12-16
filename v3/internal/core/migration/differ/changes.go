// Package differ implements change types.
package differ

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
)

// CreateTableChange represents creating a new table.
type CreateTableChange struct {
	Table domain.Table
}

func (c *CreateTableChange) Type() domain.ChangeType { return domain.CreateTable }
func (c *CreateTableChange) Description() string {
	return fmt.Sprintf("Create table %s", c.Table.Name)
}
func (c *CreateTableChange) IsDestructive() bool { return false }
func (c *CreateTableChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	var sql string
	switch dialect {
	case domain.PostgreSQL, domain.MySQL, domain.SQLite:
		sql = fmt.Sprintf("CREATE TABLE %s (", c.Table.Name)
		for i, col := range c.Table.Columns {
			if i > 0 {
				sql += ", "
			}
			sql += fmt.Sprintf("%s %s", col.Name, col.Type)
			if !col.IsNullable {
				sql += " NOT NULL"
			}
			if col.DefaultValue != nil {
				sql += fmt.Sprintf(" DEFAULT %v", col.DefaultValue)
			}
		}
		sql += ")"
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}
	return []string{sql}, nil
}

// DropTableChange represents dropping a table.
type DropTableChange struct {
	TableName string
}

func (c *DropTableChange) Type() domain.ChangeType { return domain.DropTable }
func (c *DropTableChange) Description() string {
	return fmt.Sprintf("Drop table %s", c.TableName)
}
func (c *DropTableChange) IsDestructive() bool { return true }
func (c *DropTableChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	return []string{fmt.Sprintf("DROP TABLE %s", c.TableName)}, nil
}

// AddColumnChange represents adding a column to a table.
type AddColumnChange struct {
	TableName string
	Column    domain.Column
}

func (c *AddColumnChange) Type() domain.ChangeType { return domain.AddColumn }
func (c *AddColumnChange) Description() string {
	return fmt.Sprintf("Add column %s to table %s", c.Column.Name, c.TableName)
}
func (c *AddColumnChange) IsDestructive() bool { return false }
func (c *AddColumnChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", c.TableName, c.Column.Name, c.Column.Type)
	if !c.Column.IsNullable {
		sql += " NOT NULL"
	}
	if c.Column.DefaultValue != nil {
		sql += fmt.Sprintf(" DEFAULT %v", c.Column.DefaultValue)
	}
	return []string{sql}, nil
}

// DropColumnChange represents dropping a column from a table.
type DropColumnChange struct {
	TableName  string
	ColumnName string
}

func (c *DropColumnChange) Type() domain.ChangeType { return domain.DropColumn }
func (c *DropColumnChange) Description() string {
	return fmt.Sprintf("Drop column %s from table %s", c.ColumnName, c.TableName)
}
func (c *DropColumnChange) IsDestructive() bool { return true }
func (c *DropColumnChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	return []string{fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", c.TableName, c.ColumnName)}, nil
}

// AlterColumnChange represents altering a column.
type AlterColumnChange struct {
	TableName   string
	ColumnName  string
	OldType     string
	NewType     string
	OldNullable bool
	NewNullable bool
}

func (c *AlterColumnChange) Type() domain.ChangeType { return domain.AlterColumn }
func (c *AlterColumnChange) Description() string {
	return fmt.Sprintf("Alter column %s in table %s", c.ColumnName, c.TableName)
}
func (c *AlterColumnChange) IsDestructive() bool { return true }
func (c *AlterColumnChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	var sqls []string

	// Type change
	if c.NewType != "" && c.OldType != c.NewType {
		switch dialect {
		case domain.PostgreSQL:
			sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", c.TableName, c.ColumnName, c.NewType))
		case domain.MySQL:
			sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s", c.TableName, c.ColumnName, c.NewType))
		case domain.SQLite:
			// SQLite doesn't support ALTER COLUMN, would need table recreation
			return nil, fmt.Errorf("SQLite does not support ALTER COLUMN")
		}
	}

	// Nullable change
	if c.OldNullable != c.NewNullable {
		switch dialect {
		case domain.PostgreSQL:
			if c.NewNullable {
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", c.TableName, c.ColumnName))
			} else {
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", c.TableName, c.ColumnName))
			}
		case domain.MySQL:
			nullable := "NOT NULL"
			if c.NewNullable {
				nullable = "NULL"
			}
			sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s %s", c.TableName, c.ColumnName, c.NewType, nullable))
		}
	}

	return sqls, nil
}

// CreateIndexChange represents creating an index.
type CreateIndexChange struct {
	TableName string
	Index     domain.Index
}

func (c *CreateIndexChange) Type() domain.ChangeType { return domain.CreateIndex }
func (c *CreateIndexChange) Description() string {
	return fmt.Sprintf("Create index %s on table %s", c.Index.Name, c.TableName)
}
func (c *CreateIndexChange) IsDestructive() bool { return false }
func (c *CreateIndexChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	unique := ""
	if c.Index.IsUnique {
		unique = "UNIQUE "
	}
	columns := ""
	for i, col := range c.Index.Columns {
		if i > 0 {
			columns += ", "
		}
		columns += col
	}
	return []string{fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", unique, c.Index.Name, c.TableName, columns)}, nil
}

// DropIndexChange represents dropping an index.
type DropIndexChange struct {
	TableName string
	IndexName string
}

func (c *DropIndexChange) Type() domain.ChangeType { return domain.DropIndex }
func (c *DropIndexChange) Description() string {
	return fmt.Sprintf("Drop index %s from table %s", c.IndexName, c.TableName)
}
func (c *DropIndexChange) IsDestructive() bool { return false }
func (c *DropIndexChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	switch dialect {
	case domain.PostgreSQL, domain.SQLite:
		return []string{fmt.Sprintf("DROP INDEX %s", c.IndexName)}, nil
	case domain.MySQL:
		return []string{fmt.Sprintf("DROP INDEX %s ON %s", c.IndexName, c.TableName)}, nil
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}
}
