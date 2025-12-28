package runtime

import "github.com/satishbabariya/prisma-go/pkg/domain"

// SQLDialect represents a SQL dialect.
type SQLDialect string

const (
	// PostgreSQL dialect
	PostgreSQL SQLDialect = "postgres"
	// MySQL dialect
	MySQL SQLDialect = "mysql"
	// SQLite dialect
	SQLite SQLDialect = "sqlite"
)

func (d SQLDialect) toDomain() domain.SQLDialect {
	return domain.SQLDialect(d)
}
