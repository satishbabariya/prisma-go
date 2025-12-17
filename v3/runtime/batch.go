// Package runtime provides batch operations for Prisma clients.
package runtime

import (
	"context"
	"fmt"
	"sync"
)

// BatchOperation represents a single operation in a batch.
type BatchOperation struct {
	// Model is the model to operate on.
	Model string

	// Operation is the operation type.
	Operation string

	// Query is the query string.
	Query string

	// Args are the query arguments.
	Args []interface{}

	// Result will hold the operation result.
	Result interface{}

	// Error will hold any error.
	Error error
}

// Batch represents a batch of operations to execute together.
type Batch struct {
	client     *Client
	operations []*BatchOperation
	mu         sync.Mutex
}

// NewBatch creates a new batch for the client.
func (c *Client) NewBatch() *Batch {
	return &Batch{
		client:     c,
		operations: []*BatchOperation{},
	}
}

// Add adds an operation to the batch.
func (b *Batch) Add(op *BatchOperation) *Batch {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.operations = append(b.operations, op)
	return b
}

// AddQuery adds a query operation to the batch.
func (b *Batch) AddQuery(model, query string, args ...interface{}) *Batch {
	return b.Add(&BatchOperation{
		Model:     model,
		Operation: "query",
		Query:     query,
		Args:      args,
	})
}

// AddExec adds an exec operation to the batch.
func (b *Batch) AddExec(model, query string, args ...interface{}) *Batch {
	return b.Add(&BatchOperation{
		Model:     model,
		Operation: "exec",
		Query:     query,
		Args:      args,
	})
}

// Execute executes all operations in the batch within a transaction.
func (b *Batch) Execute(ctx context.Context) error {
	if len(b.operations) == 0 {
		return nil
	}

	return b.client.Transaction(ctx, func(tx Transaction) error {
		for _, op := range b.operations {
			switch op.Operation {
			case "query":
				rows, err := tx.QueryContext(ctx, op.Query, op.Args...)
				if err != nil {
					op.Error = err
					return fmt.Errorf("batch query failed for %s: %w", op.Model, err)
				}
				defer rows.Close()

				// TODO: Map rows to result based on model type
				op.Result = rows

			case "exec":
				result, err := tx.ExecContext(ctx, op.Query, op.Args...)
				if err != nil {
					op.Error = err
					return fmt.Errorf("batch exec failed for %s: %w", op.Model, err)
				}
				op.Result = result

			default:
				return fmt.Errorf("unknown batch operation: %s", op.Operation)
			}
		}
		return nil
	})
}

// ExecuteParallel executes operations in parallel (not in a transaction).
func (b *Batch) ExecuteParallel(ctx context.Context) error {
	if len(b.operations) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(b.operations))

	for _, op := range b.operations {
		wg.Add(1)
		go func(op *BatchOperation) {
			defer wg.Done()

			switch op.Operation {
			case "query":
				rows, err := b.client.QueryContext(ctx, op.Query, op.Args...)
				if err != nil {
					op.Error = err
					errCh <- err
					return
				}
				defer rows.Close()
				op.Result = rows

			case "exec":
				result, err := b.client.ExecContext(ctx, op.Query, op.Args...)
				if err != nil {
					op.Error = err
					errCh <- err
					return
				}
				op.Result = result
			}
		}(op)
	}

	wg.Wait()
	close(errCh)

	// Collect first error
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// Results returns all operation results.
func (b *Batch) Results() []*BatchOperation {
	return b.operations
}

// Clear removes all operations from the batch.
func (b *Batch) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.operations = []*BatchOperation{}
}

// Size returns the number of operations in the batch.
func (b *Batch) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.operations)
}
