// Package client provides the runtime client for Prisma Go.
package client

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver
)

// PrismaClient is the main database client
type PrismaClient struct {
	db          *sql.DB
	provider    string
	middlewares []Middleware
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
		db:          db,
		provider:    provider,
		middlewares: []Middleware{},
	}, nil
}

// NewPrismaClientFromDB creates a new Prisma client from a database connection
func NewPrismaClientFromDB(provider string, db *sql.DB) (*PrismaClient, error) {
	return &PrismaClient{
		db:          db,
		provider:    provider,
		middlewares: []Middleware{},
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

// SetMaxOpenConns sets the maximum number of open connections to the database
func (c *PrismaClient) SetMaxOpenConns(n int) {
	c.db.SetMaxOpenConns(n)
}

// SetMaxIdleConns sets the maximum number of idle connections in the pool
func (c *PrismaClient) SetMaxIdleConns(n int) {
	c.db.SetMaxIdleConns(n)
}

// SetConnMaxLifetime sets the maximum amount of time a connection may be reused
func (c *PrismaClient) SetConnMaxLifetime(d time.Duration) {
	c.db.SetConnMaxLifetime(d)
}

// SetConnMaxIdleTime sets the maximum amount of time a connection may be idle
func (c *PrismaClient) SetConnMaxIdleTime(d time.Duration) {
	c.db.SetConnMaxIdleTime(d)
}

// Disconnect closes the database connection
func (c *PrismaClient) Disconnect(ctx context.Context) error {
	return c.db.Close()
}

// DB returns the underlying database connection
func (c *PrismaClient) DB() *sql.DB {
	return c.db
}

// RawQuery executes a raw SQL query with parameters and maps results to structs
func (c *PrismaClient) RawQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

// RawQueryRow executes a raw SQL query that returns a single row
func (c *PrismaClient) RawQueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

// RawExec executes a raw SQL statement (INSERT, UPDATE, DELETE)
func (c *PrismaClient) RawExec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error

	if len(c.middlewares) > 0 {
		err = c.executeWithMiddleware(ctx, query, args, func() error {
			var execErr error
			result, execErr = c.db.ExecContext(ctx, query, args...)
			return execErr
		})
	} else {
		result, err = c.db.ExecContext(ctx, query, args...)
	}

	return result, err
}

// Use adds a middleware to the client
func (c *PrismaClient) Use(middleware Middleware) {
	c.middlewares = append(c.middlewares, middleware)
}

// executeWithMiddleware executes a query with middleware chain
func (c *PrismaClient) executeWithMiddleware(ctx context.Context, query string, args []interface{}, exec func() error) error {
	if len(c.middlewares) == 0 {
		return exec()
	}

	event := &QueryEvent{
		Query: query,
		Args:  args,
		Start: time.Now(),
	}

	var next func() error
	index := 0

	next = func() error {
		if index >= len(c.middlewares) {
			// Last middleware, execute the actual query
			err := exec()
			event.End = time.Now()
			event.Duration = event.End.Sub(event.Start)
			event.Error = err
			return err
		}

		middleware := c.middlewares[index]
		index++
		return middleware(ctx, event, next)
	}

	return next()
}
