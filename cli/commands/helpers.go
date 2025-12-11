package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/satishbabariya/prisma-go/migrate/introspect"
	psl "github.com/satishbabariya/prisma-go/psl"
)

// extractConnectionInfo extracts provider and connection string from schema
// It also attempts to find DATABASE_URL from environment and .env files
func extractConnectionInfo(schema *psl.SchemaAst) (string, string) {
	provider, connStr, _ := extractConnectionInfoWithShadow(schema)
	return provider, connStr
}

// extractConnectionInfoWithShadow extracts provider, connection string, and shadow database URL from schema
func extractConnectionInfoWithShadow(schema *psl.SchemaAst) (string, string, string) {
	provider := "postgresql"
	connStr := ""
	shadowConnStr := ""

	for _, top := range schema.Tops {
		if source := top.AsSource(); source != nil {
			for _, prop := range source.Properties {
				if prop.Name.Name == "provider" {
					if strLit, _ := prop.Value.AsStringValue(); strLit != nil {
						provider = strLit.Value
					}
				}
				if prop.Name.Name == "url" {
					// Handle env("DATABASE_URL") function call
					if fnCall := prop.Value.AsFunction(); fnCall != nil && fnCall.Name.Name == "env" {
						if len(fnCall.Arguments) > 0 {
							if strLit, _ := fnCall.Arguments[0].AsStringValue(); strLit != nil {
								envVar := strLit.Value
								// Try to get from environment or .env files first
								connStr = getDatabaseURLFromEnv()
								// If still empty, try direct env lookup with the specified variable name
								if connStr == "" {
									connStr = os.Getenv(envVar)
								}
							}
						}
					} else if strLit, _ := prop.Value.AsStringValue(); strLit != nil {
						// Direct string literal
						connStr = strLit.Value
					}
				}
				if prop.Name.Name == "shadowDatabaseUrl" {
					// Handle shadow database URL
					if fnCall := prop.Value.AsFunction(); fnCall != nil && fnCall.Name.Name == "env" {
						if len(fnCall.Arguments) > 0 {
							if strLit, _ := fnCall.Arguments[0].AsStringValue(); strLit != nil {
								envVar := strLit.Value
								shadowConnStr = os.Getenv(envVar)
							}
						}
					} else if strLit, _ := prop.Value.AsStringValue(); strLit != nil {
						// Direct string literal
						shadowConnStr = strLit.Value
					}
				}
			}
		}
	}

	// If no connection string found in schema, try auto-detection from environment
	if connStr == "" {
		connStr = getDatabaseURLFromEnv()
	}

	return provider, connStr, shadowConnStr
}

func detectProvider(connStr string) string {
	if strings.Contains(connStr, "mysql") {
		return "mysql"
	} else if strings.Contains(connStr, "sqlite") || strings.Contains(connStr, "file:") {
		return "sqlite"
	}
	return "postgresql"
}

// normalizeProviderForDriver normalizes provider name for sql.Open
// PostgreSQL driver uses "postgres", not "postgresql"
// SQLite driver uses "sqlite3", not "sqlite"
func normalizeProviderForDriver(provider string) string {
	switch provider {
	case "postgresql", "postgres":
		return "postgres"
	case "sqlite":
		return "sqlite3"
	default:
		return provider
	}
}

func generatePrismaSchemaFromDB(schema *introspect.DatabaseSchema, provider string) string {
	var result strings.Builder

	// Datasource
	result.WriteString("datasource db {\n")
	result.WriteString(fmt.Sprintf("  provider = \"%s\"\n", provider))
	result.WriteString("  url      = env(\"DATABASE_URL\")\n")
	result.WriteString("}\n\n")

	// Generator
	result.WriteString("generator client {\n")
	result.WriteString("  provider = \"prisma-client-go\"\n")
	result.WriteString("  output   = \"./generated\"\n")
	result.WriteString("}\n\n")

	// Models
	for _, table := range schema.Tables {
		result.WriteString(fmt.Sprintf("model %s {\n", toPascalCase(table.Name)))

		for _, col := range table.Columns {
			fieldType := mapDBTypeToPrisma(col.Type)
			nullable := ""
			if col.Nullable {
				nullable = "?"
			}

			attrs := ""
			// Check if primary key
			if table.PrimaryKey != nil && len(table.PrimaryKey.Columns) == 1 && table.PrimaryKey.Columns[0] == col.Name {
				attrs += " @id"
				if col.AutoIncrement {
					attrs += " @default(autoincrement())"
				}
			}

			// Check for unique indexes
			for _, idx := range table.Indexes {
				if idx.IsUnique && len(idx.Columns) == 1 && idx.Columns[0] == col.Name {
					attrs += " @unique"
					break
				}
			}

			result.WriteString(fmt.Sprintf("  %s %s%s%s\n", col.Name, fieldType, nullable, attrs))
		}

		result.WriteString("}\n\n")
	}

	return result.String()
}

func mapDBTypeToPrisma(dbType string) string {
	dbType = strings.ToUpper(dbType)
	switch {
	case strings.Contains(dbType, "INT"), strings.Contains(dbType, "SERIAL"):
		return "Int"
	case strings.Contains(dbType, "BOOL"):
		return "Boolean"
	case strings.Contains(dbType, "VARCHAR"), strings.Contains(dbType, "TEXT"), strings.Contains(dbType, "CHAR"):
		return "String"
	case strings.Contains(dbType, "TIMESTAMP"), strings.Contains(dbType, "DATE"):
		return "DateTime"
	case strings.Contains(dbType, "DECIMAL"), strings.Contains(dbType, "NUMERIC"), strings.Contains(dbType, "FLOAT"), strings.Contains(dbType, "DOUBLE"), strings.Contains(dbType, "REAL"):
		return "Float"
	case strings.Contains(dbType, "JSON"):
		return "Json"
	default:
		return "String"
	}
}

func toPascalCase(s string) string {
	words := strings.Split(s, "_")
	result := ""
	for _, word := range words {
		if len(word) > 0 {
			result += strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return result
}

// getSchemaPath returns the schema path using consistent logic:
// 1. Use explicit flag value if set
// 2. Use first argument if provided
// 3. Default to "schema.prisma"
func getSchemaPath(flagValue string, args []string) string {
	if flagValue != "" && flagValue != "schema.prisma" {
		return flagValue
	}
	if len(args) > 0 {
		return args[0]
	}
	return "schema.prisma"
}

// findSchemaFile attempts to find a schema file in common locations
func findSchemaFile() string {
	commonPaths := []string{
		"schema.prisma",
		"prisma/schema.prisma",
		"./schema.prisma",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath
		}
	}
	return ""
}

// getDatabaseURLFromEnv attempts to find DATABASE_URL from environment and .env files:
// 1. Environment variable DATABASE_URL
// 2. .env file in current directory
// 3. .env.local file in current directory (higher priority)
func getDatabaseURLFromEnv() string {
	// First check environment variable
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	// Check .env.local file first (higher priority)
	if data, err := os.ReadFile(".env.local"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Skip comments
			if strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "DATABASE_URL=") {
				url := strings.TrimPrefix(line, "DATABASE_URL=")
				url = strings.Trim(url, `"'"`)
				if url != "" {
					return url
				}
			}
		}
	}

	// Check .env file
	if data, err := os.ReadFile(".env"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Skip comments
			if strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "DATABASE_URL=") {
				url := strings.TrimPrefix(line, "DATABASE_URL=")
				url = strings.Trim(url, `"'"`)
				if url != "" {
					return url
				}
			}
		}
	}

	return ""
}
