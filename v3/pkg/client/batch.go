// Package client provides batch operations support.
package client

import (
	"context"
)

// BatchClient provides batch operation methods.
type BatchClient interface {
	// Batch creates a new batch of operations.
	Batch() *Batch
}

// Batch represents a batch of database operations.
type Batch struct {
	operations []BatchOperation
}

// BatchOperation represents a single operation in a batch.
type BatchOperation struct {
	// Model is the model being operated on.
	Model string

	// Operation is the operation type (create, update, delete).
	Operation string

	// Query is the SQL query string.
	Query string

	// Args are the query arguments.
	Args []interface{}

	// Result holds the operation result after execution.
	Result interface{}

	// Error holds any error from execution.
	Error error
}

// NewBatch creates a new batch.
func NewBatch() *Batch {
	return &Batch{
		operations: []BatchOperation{},
	}
}

// Add adds an operation to the batch.
func (b *Batch) Add(op BatchOperation) *Batch {
	b.operations = append(b.operations, op)
	return b
}

// Clear removes all operations from the batch.
func (b *Batch) Clear() *Batch {
	b.operations = []BatchOperation{}
	return b
}

// Size returns the number of operations in the batch.
func (b *Batch) Size() int {
	return len(b.operations)
}

// Operations returns all operations in the batch.
func (b *Batch) Operations() []BatchOperation {
	return b.operations
}

// BatchExecutor executes batch operations.
type BatchExecutor interface {
	// Execute runs all operations in the batch within a transaction.
	Execute(ctx context.Context, batch *Batch) error

	// ExecuteParallel runs all operations in parallel (not transactional).
	ExecuteParallel(ctx context.Context, batch *Batch) error
}
