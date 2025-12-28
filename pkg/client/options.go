// Package client provides client configuration options.
package client

import (
	"time"
)

// Config contains all client configuration options.
type Config struct {
	// DatabaseURL is the database connection string.
	// Supports PostgreSQL, MySQL, and SQLite.
	DatabaseURL string

	// MaxOpenConnections is the maximum number of open connections.
	// Default: 25
	MaxOpenConnections int

	// MaxIdleConnections is the maximum number of idle connections.
	// Default: 5
	MaxIdleConnections int

	// ConnMaxLifetime is the maximum lifetime of a connection.
	// Default: 1 hour
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum idle time of a connection.
	// Default: 10 minutes
	ConnMaxIdleTime time.Duration

	// QueryTimeout is the default timeout for queries.
	// Default: 30 seconds
	QueryTimeout time.Duration

	// Logger is the logger instance for query logging.
	Logger Logger

	// LogQueries enables query logging when true.
	// Default: false
	LogQueries bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxOpenConnections: 25,
		MaxIdleConnections: 5,
		ConnMaxLifetime:    time.Hour,
		ConnMaxIdleTime:    10 * time.Minute,
		QueryTimeout:       30 * time.Second,
		LogQueries:         false,
	}
}

// Option is a function that configures the client.
type Option func(*Config)

// WithDatabaseURL sets the database URL.
func WithDatabaseURL(url string) Option {
	return func(c *Config) {
		c.DatabaseURL = url
	}
}

// WithMaxOpenConnections sets the maximum open connections.
func WithMaxOpenConnections(n int) Option {
	return func(c *Config) {
		c.MaxOpenConnections = n
	}
}

// WithMaxIdleConnections sets the maximum idle connections.
func WithMaxIdleConnections(n int) Option {
	return func(c *Config) {
		c.MaxIdleConnections = n
	}
}

// WithConnMaxLifetime sets the connection maximum lifetime.
func WithConnMaxLifetime(d time.Duration) Option {
	return func(c *Config) {
		c.ConnMaxLifetime = d
	}
}

// WithQueryTimeout sets the query timeout.
func WithQueryTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.QueryTimeout = d
	}
}

// WithLogger sets the logger.
func WithLogger(logger Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithLogQueries enables or disables query logging.
func WithLogQueries(enabled bool) Option {
	return func(c *Config) {
		c.LogQueries = enabled
	}
}

// ApplyOptions applies options to a config.
func ApplyOptions(config *Config, opts ...Option) {
	for _, opt := range opts {
		opt(config)
	}
}
