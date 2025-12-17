// Package client provides tests for the public client API.
package client

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxOpenConnections != 25 {
		t.Errorf("Expected MaxOpenConnections 25, got %d", config.MaxOpenConnections)
	}

	if config.MaxIdleConnections != 5 {
		t.Errorf("Expected MaxIdleConnections 5, got %d", config.MaxIdleConnections)
	}

	if config.QueryTimeout != 30*time.Second {
		t.Errorf("Expected QueryTimeout 30s, got %v", config.QueryTimeout)
	}
}

func TestConfigOptions(t *testing.T) {
	config := DefaultConfig()

	// Apply options
	ApplyOptions(config,
		WithDatabaseURL("postgres://localhost:5432/test"),
		WithMaxOpenConnections(100),
		WithQueryTimeout(60*time.Second),
		WithLogQueries(true),
	)

	if config.DatabaseURL != "postgres://localhost:5432/test" {
		t.Errorf("Unexpected DatabaseURL: %s", config.DatabaseURL)
	}

	if config.MaxOpenConnections != 100 {
		t.Errorf("Expected MaxOpenConnections 100, got %d", config.MaxOpenConnections)
	}

	if config.QueryTimeout != 60*time.Second {
		t.Errorf("Expected QueryTimeout 60s, got %v", config.QueryTimeout)
	}

	if !config.LogQueries {
		t.Error("Expected LogQueries to be true")
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		isFunc   func(error) bool
		expected bool
	}{
		{
			name:     "IsNotFound with ErrNotFound",
			err:      ErrNotFound,
			isFunc:   IsNotFound,
			expected: true,
		},
		{
			name:     "IsNotFound with other error",
			err:      ErrConnection,
			isFunc:   IsNotFound,
			expected: false,
		},
		{
			name:     "IsUniqueConstraint with ErrUniqueConstraint",
			err:      ErrUniqueConstraint,
			isFunc:   IsUniqueConstraint,
			expected: true,
		},
		{
			name:     "IsTimeout with ErrTimeout",
			err:      ErrTimeout,
			isFunc:   IsTimeout,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.isFunc(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPrismaError(t *testing.T) {
	err := NewNotFoundError("User")

	if err.Code != "P2001" {
		t.Errorf("Expected code P2001, got %s", err.Code)
	}

	if err.Model != "User" {
		t.Errorf("Expected model User, got %s", err.Model)
	}

	expectedMsg := "prisma [P2001] User: No User record was found"
	if err.Error() != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, err.Error())
	}

	if !IsNotFound(err) {
		t.Error("NewNotFoundError should match IsNotFound")
	}
}

func TestNewBatch(t *testing.T) {
	batch := NewBatch()

	if batch.Size() != 0 {
		t.Errorf("Expected size 0, got %d", batch.Size())
	}

	batch.Add(BatchOperation{Model: "User", Operation: "create"})
	batch.Add(BatchOperation{Model: "Post", Operation: "create"})

	if batch.Size() != 2 {
		t.Errorf("Expected size 2, got %d", batch.Size())
	}

	ops := batch.Operations()
	if len(ops) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(ops))
	}

	batch.Clear()
	if batch.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", batch.Size())
	}
}

func TestHooksRegistry(t *testing.T) {
	hooks := NewHooksRegistry()

	callCount := 0
	hooks.OnBeforeCreate("User", func(ctx context.Context, data *HookData) error {
		callCount++
		return nil
	})

	hooks.OnAfterCreate("User", func(ctx context.Context, data *HookData) error {
		callCount++
		return nil
	})

	ctx := context.Background()
	data := &HookData{
		Model:     "User",
		Operation: "create",
	}

	hooks.RunBeforeCreate(ctx, "User", data)
	hooks.RunAfterCreate(ctx, "User", data)

	if callCount != 2 {
		t.Errorf("Expected 2 hook calls, got %d", callCount)
	}
}

func TestIsolationLevel(t *testing.T) {
	tests := []struct {
		level    IsolationLevel
		expected string
	}{
		{IsolationLevelDefault, "Default"},
		{IsolationLevelReadCommitted, "Read Committed"},
		{IsolationLevelSerializable, "Serializable"},
	}

	for _, tt := range tests {
		sqlLevel := tt.level.ToSQLIsolationLevel()
		if sqlLevel.String() != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, sqlLevel.String())
		}
	}
}
