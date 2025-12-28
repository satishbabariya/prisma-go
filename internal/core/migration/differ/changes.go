// Package differ implements change types.
package differ

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/internal/core/migration/domain"
)

// quote quotes an identifier based on the dialect.
func quote(ident string, dialect domain.SQLDialect) string {
	switch dialect {
	case domain.PostgreSQL, domain.SQLite:
		return fmt.Sprintf("\"%s\"", ident)
	case domain.MySQL:
		return fmt.Sprintf("`%s`", ident)
	default:
		return ident
	}
}

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
		sql = fmt.Sprintf("CREATE TABLE %s (", quote(c.Table.Name, dialect))
		for i, col := range c.Table.Columns {
			if i > 0 {
				sql += ", "
			}

			colType := col.Type
			isAutoInc := false
			defaultVal := ""
			if col.DefaultValue != nil {
				defaultVal = fmt.Sprintf("%v", col.DefaultValue)
				if defaultVal == "autoincrement()" {
					isAutoInc = true
				}
			}

			// Handle type adjustments for autoincrement
			if isAutoInc && dialect == domain.PostgreSQL {
				if colType == "INTEGER" || colType == "INT" || colType == "Int" {
					colType = "SERIAL"
				} else if colType == "BIGINT" || colType == "BigInt" {
					colType = "BIGSERIAL"
				}
			}

			sql += fmt.Sprintf("%s %s", quote(col.Name, dialect), colType)
			if !col.IsNullable {
				sql += " NOT NULL"
			}

			// Handle default value and auto_increment suffix
			if isAutoInc {
				if dialect == domain.MySQL {
					sql += " AUTO_INCREMENT"
				} else if dialect == domain.SQLite {
					// SQLite requires INTEGER PRIMARY KEY AUTOINCREMENT
				}
			} else if col.DefaultValue != nil {
				sql += fmt.Sprintf(" DEFAULT %v", col.DefaultValue)
			}
		}

		// Add Primary Keys
		for _, constraint := range c.Table.Constraints {
			if constraint.Type == domain.PrimaryKey {
				pkCols := ""
				for i, col := range constraint.Columns {
					if i > 0 {
						pkCols += ", "
					}
					pkCols += quote(col, dialect)
				}
				sql += fmt.Sprintf(", PRIMARY KEY (%s)", pkCols)
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
	return []string{fmt.Sprintf("DROP TABLE %s", quote(c.TableName, dialect))}, nil
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
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", quote(c.TableName, dialect), quote(c.Column.Name, dialect), c.Column.Type)
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
	return []string{fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", quote(c.TableName, dialect), quote(c.ColumnName, dialect))}, nil
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
	tbl := quote(c.TableName, dialect)
	col := quote(c.ColumnName, dialect)

	// Type change
	if c.NewType != "" && c.OldType != c.NewType {
		switch dialect {
		case domain.PostgreSQL:
			sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", tbl, col, c.NewType))
		case domain.MySQL:
			sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s", tbl, col, c.NewType))
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
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", tbl, col))
			} else {
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", tbl, col))
			}
		case domain.MySQL:
			nullable := "NOT NULL"
			if c.NewNullable {
				nullable = "NULL"
			}
			sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s %s", tbl, col, c.NewType, nullable))
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
		columns += quote(col, dialect)
	}
	return []string{fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", unique, quote(c.Index.Name, dialect), quote(c.TableName, dialect), columns)}, nil
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
		return []string{fmt.Sprintf("DROP INDEX %s", quote(c.IndexName, dialect))}, nil
	case domain.MySQL:
		return []string{fmt.Sprintf("DROP INDEX %s ON %s", quote(c.IndexName, dialect), quote(c.TableName, dialect))}, nil
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}
}
