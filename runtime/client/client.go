// Package client provides the runtime client for Prisma Go.
package client

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver
)

// PrismaClient is the main database client
type PrismaClient struct {
	db       *sql.DB
	provider string
}

// NewPrismaClient creates a new Prisma client
func NewPrismaClient(provider string, connectionString string) (*PrismaClient, error) {
	driverName := getDriverName(provider)
	if driverName == "" {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	db, err := sql.Open(driverName, connectionString)
	if err != nil {
		return nil, err
	}

	return &PrismaClient{
		db:       db,
		provider: provider,
	}, nil
}

// NewPrismaClientFromDB creates a new Prisma client from a database connection
func NewPrismaClientFromDB(provider string, db *sql.DB) (*PrismaClient, error) {
	return &PrismaClient{
		db:       db,
		provider: provider,
	}, nil
}

// getDriverName maps Prisma provider names to Go database driver names
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

// Connect establishes the database connection
func (c *PrismaClient) Connect(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Disconnect closes the database connection
func (c *PrismaClient) Disconnect(ctx context.Context) error {
	return c.db.Close()
}

// DB returns the underlying database connection
func (c *PrismaClient) DB() *sql.DB {
	return c.db
}
