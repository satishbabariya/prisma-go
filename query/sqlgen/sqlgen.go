// Package sqlgen generates SQL for different database providers.
package sqlgen

// Generator generates SQL for a specific provider
type Generator interface {
	GenerateSelect(table string, columns []string, where string, orderBy string, limit, offset int) string
	GenerateInsert(table string, columns []string, values []interface{}) string
	GenerateUpdate(table string, set map[string]interface{}, where string) string
	GenerateDelete(table string, where string) string
}

// NewGenerator creates a new SQL generator for the given provider
func NewGenerator(provider string) Generator {
	switch provider {
	case "postgresql", "postgres":
		return &PostgresGenerator{}
	case "mysql":
		return &MySQLGenerator{}
	case "sqlite":
		return &SQLiteGenerator{}
	default:
		return &PostgresGenerator{} // default to postgres
	}
}

// PostgresGenerator generates PostgreSQL SQL
type PostgresGenerator struct{}

func (g *PostgresGenerator) GenerateSelect(table string, columns []string, where string, orderBy string, limit, offset int) string {
	// TODO: Implement
	return ""
}

func (g *PostgresGenerator) GenerateInsert(table string, columns []string, values []interface{}) string {
	// TODO: Implement with RETURNING
	return ""
}

func (g *PostgresGenerator) GenerateUpdate(table string, set map[string]interface{}, where string) string {
	// TODO: Implement with RETURNING
	return ""
}

func (g *PostgresGenerator) GenerateDelete(table string, where string) string {
	// TODO: Implement
	return ""
}

// MySQLGenerator generates MySQL SQL
type MySQLGenerator struct{}

func (g *MySQLGenerator) GenerateSelect(table string, columns []string, where string, orderBy string, limit, offset int) string {
	// TODO: Implement
	return ""
}

func (g *MySQLGenerator) GenerateInsert(table string, columns []string, values []interface{}) string {
	// TODO: Implement
	return ""
}

func (g *MySQLGenerator) GenerateUpdate(table string, set map[string]interface{}, where string) string {
	// TODO: Implement
	return ""
}

func (g *MySQLGenerator) GenerateDelete(table string, where string) string {
	// TODO: Implement
	return ""
}

// SQLiteGenerator generates SQLite SQL
type SQLiteGenerator struct{}

func (g *SQLiteGenerator) GenerateSelect(table string, columns []string, where string, orderBy string, limit, offset int) string {
	// TODO: Implement
	return ""
}

func (g *SQLiteGenerator) GenerateInsert(table string, columns []string, values []interface{}) string {
	// TODO: Implement
	return ""
}

func (g *SQLiteGenerator) GenerateUpdate(table string, set map[string]interface{}, where string) string {
	// TODO: Implement
	return ""
}

func (g *SQLiteGenerator) GenerateDelete(table string, where string) string {
	// TODO: Implement
	return ""
}

