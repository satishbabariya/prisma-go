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

	"github.com/satishbabariya/prisma-go/query/cache"
)

// PrismaClient is the main database client
type PrismaClient struct {
	db          *sql.DB
	provider    string
	middlewares []Middleware
	queryCache  cache.Cache
	cacheConfig CacheConfig
	extensions  *ExtensionChain
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled    bool
	MaxSize    int
	DefaultTTL time.Duration
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
		queryCache:  nil,
		cacheConfig: CacheConfig{
			Enabled:    false,
			MaxSize:    1000,
			DefaultTTL: 5 * time.Minute,
		},
		extensions: NewExtensionChain(),
	}, nil
}

// NewPrismaClientFromDB creates a new Prisma client from a database connection
func NewPrismaClientFromDB(provider string, db *sql.DB) (*PrismaClient, error) {
	return &PrismaClient{
		db:          db,
		provider:    provider,
		middlewares: []Middleware{},
		queryCache:  nil,
		cacheConfig: CacheConfig{
			Enabled:    false,
			MaxSize:    1000,
			DefaultTTL: 5 * time.Minute,
		},
		extensions: NewExtensionChain(),
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

// Raw executes a raw SQL query and returns the result
func (c *PrismaClient) Raw(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

// RawScan executes a raw SQL query and scans the results into the destination
// Note: This is a placeholder - full implementation would require reflection or sqlx
func (c *PrismaClient) RawScan(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Basic implementation - can be enhanced with reflection or sqlx
	// For now, this is a placeholder that would need proper implementation
	// based on the destination type (slice, struct, etc.)
	return fmt.Errorf("RawScan not fully implemented - use Raw() and scan manually for now")
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

// EnableCache enables query caching with the specified configuration
func (c *PrismaClient) EnableCache(maxSize int, defaultTTL time.Duration) {
	c.cacheConfig.Enabled = true
	c.cacheConfig.MaxSize = maxSize
	c.cacheConfig.DefaultTTL = defaultTTL
	c.queryCache = cache.NewLRUCache(maxSize, defaultTTL)
}

// DisableCache disables query caching
func (c *PrismaClient) DisableCache() {
	c.cacheConfig.Enabled = false
	c.queryCache = nil
}

// SetCache sets a custom cache implementation
func (c *PrismaClient) SetCache(cacheInstance cache.Cache) {
	c.queryCache = cacheInstance
	if cacheInstance != nil {
		c.cacheConfig.Enabled = true
	} else {
		c.cacheConfig.Enabled = false
	}
}

// GetCacheStats returns cache statistics
func (c *PrismaClient) GetCacheStats() cache.Stats {
	if c.queryCache != nil {
		return c.queryCache.GetStats()
	}
	return cache.Stats{}
}

// ClearCache clears all cached query results
func (c *PrismaClient) ClearCache() {
	if c.queryCache != nil {
		c.queryCache.Clear()
	}
}

// UseExtension adds an extension to the client
func (c *PrismaClient) UseExtension(ext Extension) {
	c.extensions.Add(ext)
}

// Extensions returns the extension chain
func (c *PrismaClient) Extensions() *ExtensionChain {
	return c.extensions
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
