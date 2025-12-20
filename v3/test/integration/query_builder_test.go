//go:build integration

package test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/builder"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/runtime"
)

// TestQueryBuilderIntegration tests the complete query builder workflow
func TestQueryBuilderIntegration(t *testing.T) {
	// Setup test database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE,
			age INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		INSERT INTO users (name, email, age) VALUES 
			('John Doe', 'john@example.com', 25),
			('Jane Smith', 'jane@example.com', 30),
			('Bob Wilson', 'bob@example.com', 35),
			('Alice Brown', 'alice@example.com', 28);
	`)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create compiler
	compiler := compiler.NewSQLCompiler(domain.SQLite)

	// Create query executor
	executor := runtime.NewQueryExecutor(db, domain.SQLite)

	tests := []struct {
		name      string
		queryFunc func() *domain.Query
		wantCount int
	}{
		{
			name: "Find all users",
			queryFunc: func() *domain.Query {
				return builder.NewQueryBuilder("users").
					FindMany().
					GetQuery()
			},
			wantCount: 4,
		},
		{
			name: "Find users with age > 25",
			queryFunc: func() *domain.Query {
				return builder.NewQueryBuilder("users").
					FindMany().
					Where(
						domain.Condition{
							Field:    "age",
							Operator: domain.Gt,
							Value:    25,
						},
					).
					GetQuery()
			},
			wantCount: 3,
		},
		{
			name: "Find users with age >= 30 OR name = 'John Doe'",
			queryFunc: func() *domain.Query {
				return builder.NewQueryBuilder("users").
					FindMany().
					Or(
						domain.Condition{
							Field:    "age",
							Operator: domain.Gte,
							Value:    30,
						},
						domain.Condition{
							Field:    "name",
							Operator: domain.Equals,
							Value:    "John Doe",
						},
					).
					GetQuery()
			},
			wantCount: 3,
		},
		{
			name: "Find users with name NOT LIKE 'J%'",
			queryFunc: func() *domain.Query {
				return builder.NewQueryBuilder("users").
					FindMany().
					Not(
						domain.Condition{
							Field:    "name",
							Operator: domain.StartsWith,
							Value:    "J",
						},
					).
					GetQuery()
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Build query
			query := tt.queryFunc()

			// Compile to SQL
			compiled, err := compiler.Compile(ctx, query)
			if err != nil {
				t.Fatalf("Failed to compile query: %v", err)
			}

			// Execute query
			results, err := executor.Execute(ctx, compiled)
			if err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			// Convert to slice of maps
			rows, ok := results.([]map[string]interface{})
			if !ok {
				t.Fatalf("Expected []map[string]interface{}, got %T", results)
			}

			if len(rows) != tt.wantCount {
				t.Errorf("Expected %d rows, got %d", tt.wantCount, len(rows))
			}

			// Print SQL for debugging
			t.Logf("SQL: %s", compiled.SQL.Query)
			t.Logf("Args: %v", compiled.SQL.Args)
		})
	}
}

// TestComplexLogicalOperators tests complex logical operator combinations
func TestComplexLogicalOperators(t *testing.T) {
	// Setup test database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			category TEXT,
			price DECIMAL(10,2),
			in_stock BOOLEAN,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		INSERT INTO products (name, category, price, in_stock) VALUES 
			('Laptop', 'Electronics', 999.99, true),
			('Mouse', 'Electronics', 29.99, true),
			('Keyboard', 'Electronics', 79.99, false),
			('Book', 'Books', 19.99, true),
			('Pen', 'Stationery', 2.99, true),
			('Notebook', 'Stationery', 5.99, false);
	`)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create compiler and executor
	compiler := compiler.NewSQLCompiler(domain.SQLite)
	executor := runtime.NewQueryExecutor(db, domain.SQLite)

	ctx := context.Background()

	// Test simple AND: Find Electronics products that are in stock
	query := builder.NewQueryBuilder("products").
		FindMany().
		And(
			domain.Condition{
				Field:    "category",
				Operator: domain.Equals,
				Value:    "Electronics",
			},
			domain.Condition{
				Field:    "in_stock",
				Operator: domain.Equals,
				Value:    true,
			},
		).
		GetQuery()

	// Compile query
	compiled, err := compiler.Compile(ctx, query)
	if err != nil {
		t.Fatalf("Failed to compile query: %v", err)
	}

	// Execute query
	results, err := executor.Execute(ctx, compiled)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	rows, ok := results.([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected []map[string]interface{}, got %T", results)
	}

	// Should find Laptop and Mouse (Electronics AND in stock)
	if len(rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(rows))
	}

	// Print results for debugging
	t.Logf("SQL: %s", compiled.SQL.Query)
	t.Logf("Args: %v", compiled.SQL.Args)
	for _, row := range rows {
		t.Logf("Result: %+v", row)
	}
}
