// Package integration provides integration test helpers.
package integration

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// TestDatabase holds test database connection info.
type TestDatabase struct {
	DB       *sql.DB
	Provider string
	Name     string
	cleanup  func()
}

// SetupPostgresTest creates a PostgreSQL test database.
func SetupPostgresTest(t *testing.T) *TestDatabase {
	t.Helper()

	connStr := os.Getenv("POSTGRES_TEST_URL")
	if connStr == "" {
		connStr = "postgresql://prisma:prisma@localhost:5433/prisma_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Wait for connection
	for i := 0; i < 10; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	testDB := &TestDatabase{
		DB:       db,
		Provider: "postgresql",
		Name:     "prisma_test",
		cleanup: func() {
			CleanupDatabase(t, db, "postgresql")
			db.Close()
		},
	}

	// Clean before test
	CleanupDatabase(t, db, "postgresql")

	return testDB
}

// SetupMySQLTest creates a MySQL test database.
func SetupMySQLTest(t *testing.T) *TestDatabase {
	t.Helper()

	connStr := os.Getenv("MYSQL_TEST_URL")
	if connStr == "" {
		connStr = "prisma:prisma@tcp(localhost:3307)/prisma_test?parseTime=true"
	}

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Wait for connection
	for i := 0; i < 10; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	testDB := &TestDatabase{
		DB:       db,
		Provider: "mysql",
		Name:     "prisma_test",
		cleanup: func() {
			CleanupDatabase(t, db, "mysql")
			db.Close()
		},
	}

	CleanupDatabase(t, db, "mysql")
	return testDB
}

// SetupSQLiteTest creates a SQLite test database.
func SetupSQLiteTest(t *testing.T) *TestDatabase {
	t.Helper()

	dbPath := fmt.Sprintf("/tmp/prisma_test_%d.db", time.Now().UnixNano())

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	testDB := &TestDatabase{
		DB:       db,
		Provider: "sqlite",
		Name:     dbPath,
		cleanup: func() {
			db.Close()
			os.Remove(dbPath)
		},
	}

	return testDB
}

// Cleanup closes the database and cleans up resources.
func (td *TestDatabase) Cleanup() {
	if td.cleanup != nil {
		td.cleanup()
	}
}

// CleanupDatabase removes all tables from the test database.
func CleanupDatabase(t *testing.T, db *sql.DB, provider string) {
	t.Helper()

	switch provider {
	case "postgresql":
		cleanupPostgres(t, db)
	case "mysql":
		cleanupMySQL(t, db)
	case "sqlite":
		cleanupSQLite(t, db)
	}
}

func cleanupPostgres(t *testing.T, db *sql.DB) {
	// Drop all tables
	query := `
		DO $$ DECLARE
			r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
				EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;
	`
	if _, err := db.Exec(query); err != nil {
		t.Logf("Cleanup warning: %v", err)
	}
}

func cleanupMySQL(t *testing.T, db *sql.DB) {
	// Get all tables
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		t.Logf("Cleanup warning: %v", err)
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err == nil {
			tables = append(tables, table)
		}
	}

	// Disable foreign key checks and drop tables
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	for _, table := range tables {
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table))
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
}

func cleanupSQLite(t *testing.T, db *sql.DB) {
	// Get all tables
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		t.Logf("Cleanup warning: %v", err)
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err == nil {
			tables = append(tables, table)
		}
	}

	// Drop tables
	for _, table := range tables {
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	}
}

// RunMigrations applies schema migrations to the test database.
func (td *TestDatabase) RunMigrations(schema string) error {
	// Execute schema DDL
	_, err := td.DB.Exec(schema)
	return err
}

// LoadFixture loads test data from a fixture.
func (td *TestDatabase) LoadFixture(table string, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// Build INSERT statement
	var columns []string
	for col := range data[0] {
		columns = append(columns, col)
	}

	for _, row := range data {
		var values []interface{}
		for _, col := range columns {
			values = append(values, row[col])
		}

		placeholders := make([]string, len(columns))
		for i := range placeholders {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			table,
			joinStrings(columns, ", "),
			joinStrings(placeholders, ", "),
		)

		if _, err := td.DB.Exec(query, values...); err != nil {
			return fmt.Errorf("failed to load fixture: %w", err)
		}
	}

	return nil
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
