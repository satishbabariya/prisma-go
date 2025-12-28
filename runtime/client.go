// Package runtime provides the base runtime for generated Prisma clients.
package runtime

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// Client is the base Prisma client that generated clients embed.
type Client struct {
	db          *sql.DB
	config      *ClientConfig
	middlewares []Middleware
	hooks       *Hooks
	connected   bool
	mu          sync.RWMutex
}

// ClientConfig holds client configuration options.
type ClientConfig struct {
	// DatabaseURL is the database connection string.
	DatabaseURL string

	// MaxConnections is the maximum number of open connections.
	MaxConnections int

	// MaxIdleConnections is the maximum number of idle connections.
	MaxIdleConnections int

	// ConnMaxLifetime is the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum idle time of a connection.
	ConnMaxIdleTime time.Duration

	// QueryTimeout is the default timeout for queries.
	QueryTimeout time.Duration

	// LogQueries enables query logging.
	LogQueries bool

	// Logger is the logger instance.
	Logger Logger
}

// DefaultConfig returns the default client configuration.
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		MaxConnections:     25,
		MaxIdleConnections: 5,
		ConnMaxLifetime:    time.Hour,
		ConnMaxIdleTime:    10 * time.Minute,
		QueryTimeout:       30 * time.Second,
		LogQueries:         false,
	}
}

// Option is a function that configures the client.
type Option func(*ClientConfig)

// WithDatabaseURL sets the database URL.
func WithDatabaseURL(url string) Option {
	return func(c *ClientConfig) {
		c.DatabaseURL = url
	}
}

// WithMaxConnections sets the maximum connections.
func WithMaxConnections(n int) Option {
	return func(c *ClientConfig) {
		c.MaxConnections = n
	}
}

// WithQueryTimeout sets the query timeout.
func WithQueryTimeout(d time.Duration) Option {
	return func(c *ClientConfig) {
		c.QueryTimeout = d
	}
}

// WithLogQueries enables or disables query logging.
func WithLogQueries(enabled bool) Option {
	return func(c *ClientConfig) {
		c.LogQueries = enabled
	}
}

// WithLogger sets the logger.
func WithLogger(logger Logger) Option {
	return func(c *ClientConfig) {
		c.Logger = logger
	}
}

// NewClient creates a new Prisma client.
func NewClient(opts ...Option) *Client {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	return &Client{
		config:      config,
		middlewares: []Middleware{},
		hooks:       NewHooks(),
	}
}

// Connect establishes a database connection.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	if c.config.DatabaseURL == "" {
		return fmt.Errorf("%w: database URL is required", ErrConnectionFailed)
	}

	db, err := sql.Open("postgres", c.config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(c.config.MaxConnections)
	db.SetMaxIdleConns(c.config.MaxIdleConnections)
	db.SetConnMaxLifetime(c.config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(c.config.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	c.db = db
	c.connected = true

	return nil
}

// Disconnect closes the database connection.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.db == nil {
		return nil
	}

	err := c.db.Close()
	c.connected = false
	c.db = nil

	return err
}

// IsConnected returns true if the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// DB returns the underlying database connection.
func (c *Client) DB() *sql.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

// Use adds middleware to the client.
func (c *Client) Use(mw Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middlewares = append(c.middlewares, mw)
}

// Hooks returns the client hooks.
func (c *Client) Hooks() *Hooks {
	return c.hooks
}

// QueryContext executes a query with context.
func (c *Client) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if !c.IsConnected() {
		return nil, ErrConnectionFailed
	}

	// Apply timeout if set
	if c.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.QueryTimeout)
		defer cancel()
	}

	// Log query if enabled
	if c.config.LogQueries && c.config.Logger != nil {
		c.config.Logger.Debug("Executing query", "query", query, "args", args)
	}

	return c.db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns a single row.
func (c *Client) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if !c.IsConnected() {
		return nil
	}

	// Apply timeout if set
	if c.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.QueryTimeout)
		defer cancel()
	}

	// Log query if enabled
	if c.config.LogQueries && c.config.Logger != nil {
		c.config.Logger.Debug("Executing query", "query", query, "args", args)
	}

	return c.db.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a statement.
func (c *Client) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if !c.IsConnected() {
		return nil, ErrConnectionFailed
	}

	// Apply timeout if set
	if c.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.QueryTimeout)
		defer cancel()
	}

	// Log query if enabled
	if c.config.LogQueries && c.config.Logger != nil {
		c.config.Logger.Debug("Executing statement", "query", query, "args", args)
	}

	return c.db.ExecContext(ctx, query, args...)
}
