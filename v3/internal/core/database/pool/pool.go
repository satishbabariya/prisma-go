// Package pool provides database connection pooling.
package pool

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// Config holds connection pool configuration.
type Config struct {
	// MaxOpenConns is the maximum number of open connections (0 = unlimited).
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int
	// ConnMaxLifetime is the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum idle time of a connection.
	ConnMaxIdleTime time.Duration
	// HealthCheckInterval is how often to run health checks.
	HealthCheckInterval time.Duration
}

// DefaultConfig returns sensible default pool configuration.
func DefaultConfig() Config {
	return Config{
		MaxOpenConns:        25,
		MaxIdleConns:        5,
		ConnMaxLifetime:     30 * time.Minute,
		ConnMaxIdleTime:     10 * time.Minute,
		HealthCheckInterval: 1 * time.Minute,
	}
}

// Pool manages database connections with lifecycle management.
type Pool struct {
	db     *sql.DB
	config Config

	// Metrics
	mu              sync.RWMutex
	totalConns      int64
	activeConns     int64
	idleConns       int64
	failedChecks    int64
	lastHealthCheck time.Time

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new connection pool.
func New(driverName, dataSourceName string, config Config) (*Pool, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply pool configuration
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	ctx, cancel := context.WithCancel(context.Background())

	pool := &Pool{
		db:     db,
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start health check routine if configured
	if config.HealthCheckInterval > 0 {
		pool.wg.Add(1)
		go pool.healthCheckLoop()
	}

	return pool, nil
}

// DB returns the underlying *sql.DB.
func (p *Pool) DB() *sql.DB {
	return p.db
}

// Stats returns current pool statistics.
func (p *Pool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	dbStats := p.db.Stats()

	return PoolStats{
		MaxOpenConnections: p.config.MaxOpenConns,
		OpenConnections:    dbStats.OpenConnections,
		InUse:              dbStats.InUse,
		Idle:               dbStats.Idle,
		WaitCount:          dbStats.WaitCount,
		WaitDuration:       dbStats.WaitDuration,
		MaxIdleClosed:      dbStats.MaxIdleClosed,
		MaxLifetimeClosed:  dbStats.MaxLifetimeClosed,
		FailedHealthChecks: p.failedChecks,
		LastHealthCheck:    p.lastHealthCheck,
	}
}

// PoolStats represents pool statistics.
type PoolStats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
	WaitCount          int64
	WaitDuration       time.Duration
	MaxIdleClosed      int64
	MaxLifetimeClosed  int64
	FailedHealthChecks int64
	LastHealthCheck    time.Time
}

// HealthCheck performs a health check on the connection pool.
func (p *Pool) HealthCheck(ctx context.Context) error {
	p.mu.Lock()
	p.lastHealthCheck = time.Now()
	p.mu.Unlock()

	if err := p.db.PingContext(ctx); err != nil {
		p.mu.Lock()
		p.failedChecks++
		p.mu.Unlock()
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// healthCheckLoop runs periodic health checks.
func (p *Pool) healthCheckLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = p.HealthCheck(ctx)
			cancel()
		}
	}
}

// Close closes the pool and waits for background routines to finish.
func (p *Pool) Close() error {
	p.cancel()
	p.wg.Wait()
	return p.db.Close()
}

// Exec executes a query without returning rows.
func (p *Pool) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.db.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
func (p *Pool) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row.
func (p *Pool) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

// Begin starts a transaction.
func (p *Pool) Begin(ctx context.Context) (*sql.Tx, error) {
	return p.db.BeginTx(ctx, nil)
}

// BeginTx starts a transaction with options.
func (p *Pool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return p.db.BeginTx(ctx, opts)
}
