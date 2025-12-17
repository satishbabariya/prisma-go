// Package runtime provides tests for the runtime package.
package runtime

import (
	"context"
	"testing"
	"time"
)

func TestHooks_Register_Execute(t *testing.T) {
	hooks := NewHooks()

	// Track hook calls
	var calls []string

	// Register hooks
	hooks.OnBeforeCreate("User", func(ctx *HookContext) error {
		calls = append(calls, "beforeCreate")
		return nil
	})

	hooks.OnAfterCreate("User", func(ctx *HookContext) error {
		calls = append(calls, "afterCreate")
		return nil
	})

	// Execute hooks
	ctx := &HookContext{
		Model:     "User",
		Operation: "create",
		Context:   context.Background(),
	}

	hooks.Execute(ctx, BeforeCreate)
	hooks.Execute(ctx, AfterCreate)

	if len(calls) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(calls))
	}

	if calls[0] != "beforeCreate" || calls[1] != "afterCreate" {
		t.Errorf("Unexpected call order: %v", calls)
	}
}

func TestMiddlewareChain(t *testing.T) {
	chain := NewMiddlewareChain()

	var order []string

	// Add middlewares
	chain.Use(func(ctx context.Context, info QueryInfo, next Next) QueryResult {
		order = append(order, "mw1-before")
		result := next(ctx, info)
		order = append(order, "mw1-after")
		return result
	})

	chain.Use(func(ctx context.Context, info QueryInfo, next Next) QueryResult {
		order = append(order, "mw2-before")
		result := next(ctx, info)
		order = append(order, "mw2-after")
		return result
	})

	// Execute
	handler := func(ctx context.Context, info QueryInfo) QueryResult {
		order = append(order, "handler")
		return QueryResult{Data: "result"}
	}

	result := chain.Execute(context.Background(), QueryInfo{
		Model:     "User",
		Operation: "findMany",
	}, handler)

	// Check order
	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d steps, got %d: %v", len(expected), len(order), order)
	}

	for i, step := range expected {
		if order[i] != step {
			t.Errorf("Step %d: expected %q, got %q", i, step, order[i])
		}
	}

	if result.Data != "result" {
		t.Errorf("Expected result 'result', got %v", result.Data)
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Test transaction context
	tx := &Tx{}
	ctx = WithTransaction(ctx, tx)

	gotTx, ok := TransactionFromContext(ctx)
	if !ok {
		t.Error("Expected transaction in context")
	}
	if gotTx != tx {
		t.Error("Got different transaction from context")
	}

	// Test trace ID context
	traceID := "trace-123"
	ctx = WithTraceID(ctx, traceID)

	gotID, ok := TraceIDFromContext(ctx)
	if !ok {
		t.Error("Expected trace ID in context")
	}
	if gotID != traceID {
		t.Errorf("Expected trace ID %q, got %q", traceID, gotID)
	}
}

func TestErrors(t *testing.T) {
	// Test IsNotFound
	err := &NotFoundError{Model: "User"}
	if !IsNotFound(err) {
		t.Error("NotFoundError should match IsNotFound")
	}

	if err.Error() != "no User found" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}

	// Test QueryError
	queryErr := NewQueryError("findFirst", "User", ErrNotFound)
	if queryErr.Error() != "findFirst on User: record not found" {
		t.Errorf("Unexpected error message: %s", queryErr.Error())
	}

	if !IsNotFound(queryErr) {
		t.Error("QueryError wrapping ErrNotFound should match IsNotFound")
	}
}

func TestClientConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxConnections != 25 {
		t.Errorf("Expected MaxConnections 25, got %d", config.MaxConnections)
	}

	if config.QueryTimeout != 30*time.Second {
		t.Errorf("Expected QueryTimeout 30s, got %v", config.QueryTimeout)
	}

	// Test options
	WithMaxConnections(50)(config)
	if config.MaxConnections != 50 {
		t.Errorf("Expected MaxConnections 50, got %d", config.MaxConnections)
	}

	WithQueryTimeout(10 * time.Second)(config)
	if config.QueryTimeout != 10*time.Second {
		t.Errorf("Expected QueryTimeout 10s, got %v", config.QueryTimeout)
	}
}
