package pool

import (
	"context"
	"testing"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolCreation(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, 25, config.MaxOpenConns)
	assert.Equal(t, 5, config.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, config.ConnMaxLifetime)
}

func TestPoolStats(t *testing.T) {
	// Note: This test requires a real database connection
	// For unit testing, you'd use a mock or test database
	t.Skip("Requires database connection")

	config := Config{
		MaxOpenConns:        10,
		MaxIdleConns:        2,
		ConnMaxLifetime:     5 * time.Minute,
		ConnMaxIdleTime:     1 * time.Minute,
		HealthCheckInterval: 0, // Disable for test
	}

	pool, err := New("postgres", "postgresql://localhost:5432/test", config)
	require.NoError(t, err)
	defer pool.Close()

	stats := pool.Stats()
	assert.Equal(t, 10, stats.MaxOpenConnections)
	assert.GreaterOrEqual(t, stats.OpenConnections, 0)
}

func TestHealthCheck(t *testing.T) {
	t.Skip("Requires database connection")

	config := DefaultConfig()
	config.HealthCheckInterval = 0 // Disable automatic checks

	pool, err := New("postgres", "postgresql://localhost:5432/test", config)
	require.NoError(t, err)
	defer pool.Close()

	ctx := context.Background()
	err = pool.HealthCheck(ctx)
	assert.NoError(t, err)

	stats := pool.Stats()
	assert.False(t, stats.LastHealthCheck.IsZero())
}

func TestPoolClose(t *testing.T) {
	t.Skip("Requires database connection")

	config := DefaultConfig()
	pool, err := New("postgres", "postgresql://localhost:5432/test", config)
	require.NoError(t, err)

	err = pool.Close()
	assert.NoError(t, err)
}
