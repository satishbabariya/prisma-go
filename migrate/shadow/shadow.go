// Package shadow provides shadow database management for safe migration diffing.
package shadow

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/satishbabariya/prisma-go/migrate/executor"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// ShadowDB manages shadow database operations
type ShadowDB struct {
	provider      string
	mainConnStr   string
	shadowConnStr string
	shadowDB      *sql.DB
	skipShadow    bool
}

// NewShadowDB creates a new shadow database manager
func NewShadowDB(provider string, mainConnStr string, shadowConnStr string, skipShadow bool) *ShadowDB {
	return &ShadowDB{
		provider:      provider,
		mainConnStr:   mainConnStr,
		shadowConnStr: shadowConnStr,
		skipShadow:    skipShadow,
	}
}

// Connect connects to the shadow database
func (s *ShadowDB) Connect(ctx context.Context) error {
	if s.skipShadow {
		return nil
	}

	if s.shadowConnStr == "" {
		// Generate shadow connection string from main connection string
		var err error
		s.shadowConnStr, err = s.generateShadowConnStr()
		if err != nil {
			return fmt.Errorf("failed to generate shadow connection string: %w", err)
		}
	}

	driverName := getDriverName(s.provider)
	if driverName == "" {
		return fmt.Errorf("unsupported provider: %s", s.provider)
	}

	db, err := sql.Open(driverName, s.shadowConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to shadow database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping shadow database: %w", err)
	}

	s.shadowDB = db
	return nil
}

// Create creates the shadow database if it doesn't exist
func (s *ShadowDB) Create(ctx context.Context) error {
	if s.skipShadow {
		return nil
	}

	if s.shadowDB != nil {
		// Already connected, assume it exists
		return nil
	}

	switch s.provider {
	case "postgresql", "postgres":
		return s.createPostgresShadow(ctx)
	case "mysql":
		return s.createMySQLShadow(ctx)
	case "sqlite":
		// SQLite doesn't need separate database creation
		return s.Connect(ctx)
	default:
		return fmt.Errorf("unsupported provider for shadow database: %s", s.provider)
	}
}

// Drop drops the shadow database
func (s *ShadowDB) Drop(ctx context.Context) error {
	if s.skipShadow {
		return nil
	}

	if s.shadowDB != nil {
		s.shadowDB.Close()
		s.shadowDB = nil
	}

	switch s.provider {
	case "postgresql", "postgres":
		return s.dropPostgresShadow(ctx)
	case "mysql":
		return s.dropMySQLShadow(ctx)
	case "sqlite":
		// SQLite: delete the file
		return s.dropSQLiteShadow(ctx)
	default:
		return fmt.Errorf("unsupported provider for shadow database: %s", s.provider)
	}
}

// GetDB returns the shadow database connection
func (s *ShadowDB) GetDB() *sql.DB {
	return s.shadowDB
}

// Close closes the shadow database connection
func (s *ShadowDB) Close() error {
	if s.shadowDB != nil {
		return s.shadowDB.Close()
	}
	return nil
}

// generateShadowConnStr generates a shadow database connection string from the main connection string
func (s *ShadowDB) generateShadowConnStr() (string, error) {
	switch s.provider {
	case "postgresql", "postgres":
		return s.generatePostgresShadowConnStr()
	case "mysql":
		return s.generateMySQLShadowConnStr()
	case "sqlite":
		return s.generateSQLiteShadowConnStr()
	default:
		return "", fmt.Errorf("unsupported provider: %s", s.provider)
	}
}

// generatePostgresShadowConnStr generates PostgreSQL shadow connection string
func (s *ShadowDB) generatePostgresShadowConnStr() (string, error) {
	// Parse main connection string and append _shadow to database name
	// Format: postgresql://user:password@host:port/dbname
	connStr := s.mainConnStr
	if strings.Contains(connStr, "/") {
		parts := strings.Split(connStr, "/")
		if len(parts) >= 2 {
			parts[len(parts)-1] = parts[len(parts)-1] + "_shadow"
			return strings.Join(parts, "/"), nil
		}
	}
	return connStr + "_shadow", nil
}

// generateMySQLShadowConnStr generates MySQL shadow connection string
func (s *ShadowDB) generateMySQLShadowConnStr() (string, error) {
	// Parse main connection string and append _shadow to database name
	// Format: user:password@tcp(host:port)/dbname
	connStr := s.mainConnStr
	if strings.Contains(connStr, "/") {
		parts := strings.Split(connStr, "/")
		if len(parts) >= 2 {
			parts[len(parts)-1] = parts[len(parts)-1] + "_shadow"
			return strings.Join(parts, "/"), nil
		}
	}
	return connStr + "_shadow", nil
}

// generateSQLiteShadowConnStr generates SQLite shadow connection string
func (s *ShadowDB) generateSQLiteShadowConnStr() (string, error) {
	// For SQLite, append _shadow to the file path
	connStr := s.mainConnStr
	if strings.HasPrefix(connStr, "file:") {
		return strings.Replace(connStr, ".db", "_shadow.db", 1), nil
	}
	return connStr + "_shadow.db", nil
}

// createPostgresShadow creates a PostgreSQL shadow database
func (s *ShadowDB) createPostgresShadow(ctx context.Context) error {
	// Connect to postgres database to create shadow database
	connStr := s.mainConnStr
	if strings.Contains(connStr, "/") {
		parts := strings.Split(connStr, "/")
		parts[len(parts)-1] = "postgres"
		connStr = strings.Join(parts, "/")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer db.Close()

	shadowDBName := s.getShadowDBName()
	createSQL := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(shadowDBName))
	_, err = db.ExecContext(ctx, createSQL)
	if err != nil {
		// Check if database already exists
		if strings.Contains(err.Error(), "already exists") {
			// Database exists, connect to it
			return s.Connect(ctx)
		}
		return fmt.Errorf("failed to create shadow database: %w", err)
	}

	return s.Connect(ctx)
}

// dropPostgresShadow drops a PostgreSQL shadow database
func (s *ShadowDB) dropPostgresShadow(ctx context.Context) error {
	// Connect to postgres database to drop shadow database
	connStr := s.mainConnStr
	if strings.Contains(connStr, "/") {
		parts := strings.Split(connStr, "/")
		parts[len(parts)-1] = "postgres"
		connStr = strings.Join(parts, "/")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer db.Close()

	shadowDBName := s.getShadowDBName()
	dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdentifier(shadowDBName))
	_, err = db.ExecContext(ctx, dropSQL)
	return err
}

// createMySQLShadow creates a MySQL shadow database
func (s *ShadowDB) createMySQLShadow(ctx context.Context) error {
	// Connect without database name
	connStr := s.mainConnStr
	if strings.Contains(connStr, "/") {
		parts := strings.Split(connStr, "/")
		connStr = strings.Join(parts[:len(parts)-1], "/")
	}

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer db.Close()

	shadowDBName := s.getShadowDBName()
	createSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", shadowDBName)
	_, err = db.ExecContext(ctx, createSQL)
	if err != nil {
		return fmt.Errorf("failed to create shadow database: %w", err)
	}

	return s.Connect(ctx)
}

// dropMySQLShadow drops a MySQL shadow database
func (s *ShadowDB) dropMySQLShadow(ctx context.Context) error {
	// Connect without database name
	connStr := s.mainConnStr
	if strings.Contains(connStr, "/") {
		parts := strings.Split(connStr, "/")
		connStr = strings.Join(parts[:len(parts)-1], "/")
	}

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer db.Close()

	shadowDBName := s.getShadowDBName()
	dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", shadowDBName)
	_, err = db.ExecContext(ctx, dropSQL)
	return err
}

// dropSQLiteShadow drops a SQLite shadow database file
func (s *ShadowDB) dropSQLiteShadow(ctx context.Context) error {
	// SQLite shadow is just a file, so we don't need to do anything special
	// The file will be overwritten on next creation
	return nil
}

// getShadowDBName extracts the shadow database name from connection string
func (s *ShadowDB) getShadowDBName() string {
	if s.shadowConnStr != "" {
		// Extract database name from shadow connection string
		switch s.provider {
		case "postgresql", "postgres":
			if strings.Contains(s.shadowConnStr, "/") {
				parts := strings.Split(s.shadowConnStr, "/")
				return parts[len(parts)-1]
			}
		case "mysql":
			if strings.Contains(s.shadowConnStr, "/") {
				parts := strings.Split(s.shadowConnStr, "/")
				dbPart := parts[len(parts)-1]
				// Remove query parameters
				if idx := strings.Index(dbPart, "?"); idx != -1 {
					dbPart = dbPart[:idx]
				}
				return dbPart
			}
		}
	}

	// Generate from main connection string
	switch s.provider {
	case "postgresql", "postgres":
		if strings.Contains(s.mainConnStr, "/") {
			parts := strings.Split(s.mainConnStr, "/")
			return parts[len(parts)-1] + "_shadow"
		}
	case "mysql":
		if strings.Contains(s.mainConnStr, "/") {
			parts := strings.Split(s.mainConnStr, "/")
			dbPart := parts[len(parts)-1]
			if idx := strings.Index(dbPart, "?"); idx != -1 {
				dbPart = dbPart[:idx]
			}
			return dbPart + "_shadow"
		}
	}

	return "shadow_db"
}

// ApplyMigrations applies migrations to the shadow database
func (s *ShadowDB) ApplyMigrations(ctx context.Context, migrations []string) error {
	if s.skipShadow || s.shadowDB == nil {
		return nil
	}

	executor := executor.NewMigrationExecutor(s.shadowDB, s.provider)

	// Ensure migration table exists
	if err := executor.EnsureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration table: %w", err)
	}

	// Apply each migration
	for _, migrationSQL := range migrations {
		if err := executor.ExecuteMigration(ctx, migrationSQL, "shadow_migration"); err != nil {
			return fmt.Errorf("failed to apply migration to shadow database: %w", err)
		}
	}

	return nil
}

// Introspect introspects the shadow database schema
func (s *ShadowDB) Introspect(ctx context.Context) (*introspect.DatabaseSchema, error) {
	if s.skipShadow || s.shadowDB == nil {
		return nil, fmt.Errorf("shadow database not available")
	}

	introspector, err := introspect.NewIntrospector(s.shadowDB, s.provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create introspector: %w", err)
	}

	return introspector.Introspect(ctx)
}

// getDriverName maps provider to driver name
func getDriverName(provider string) string {
	switch provider {
	case "postgresql", "postgres":
		return "postgres"
	case "mysql":
		return "mysql"
	case "sqlite":
		return "sqlite3"
	default:
		return ""
	}
}

// quoteIdentifier quotes a database identifier
func quoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}
