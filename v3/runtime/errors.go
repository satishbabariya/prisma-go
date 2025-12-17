// Package runtime provides the base runtime for generated Prisma clients.
package runtime

import (
	"errors"
	"fmt"
)

// Error types for runtime operations.
var (
	// ErrNotFound is returned when a record is not found.
	ErrNotFound = errors.New("record not found")

	// ErrUniqueConstraint is returned when a unique constraint is violated.
	ErrUniqueConstraint = errors.New("unique constraint violation")

	// ErrForeignKeyConstraint is returned when a foreign key constraint is violated.
	ErrForeignKeyConstraint = errors.New("foreign key constraint violation")

	// ErrNullConstraint is returned when a null constraint is violated.
	ErrNullConstraint = errors.New("null constraint violation")

	// ErrConnectionFailed is returned when database connection fails.
	ErrConnectionFailed = errors.New("database connection failed")

	// ErrTransactionFailed is returned when a transaction fails.
	ErrTransactionFailed = errors.New("transaction failed")

	// ErrInvalidQuery is returned when a query is invalid.
	ErrInvalidQuery = errors.New("invalid query")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("operation timeout")

	// ErrCanceled is returned when an operation is canceled.
	ErrCanceled = errors.New("operation canceled")
)

// QueryError represents a query execution error with context.
type QueryError struct {
	Operation string
	Model     string
	Cause     error
	Query     string
	Args      []interface{}
}

// Error implements the error interface.
func (e *QueryError) Error() string {
	if e.Model != "" {
		return fmt.Sprintf("%s on %s: %v", e.Operation, e.Model, e.Cause)
	}
	return fmt.Sprintf("%s: %v", e.Operation, e.Cause)
}

// Unwrap returns the underlying error.
func (e *QueryError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target.
func (e *QueryError) Is(target error) bool {
	return errors.Is(e.Cause, target)
}

// NewQueryError creates a new QueryError.
func NewQueryError(op, model string, cause error) *QueryError {
	return &QueryError{
		Operation: op,
		Model:     model,
		Cause:     cause,
	}
}

// NotFoundError is returned when a record is not found.
type NotFoundError struct {
	Model string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("no %s found", e.Model)
}

// Is checks if the error is ErrNotFound.
func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUniqueConstraint checks if an error is a unique constraint violation.
func IsUniqueConstraint(err error) bool {
	return errors.Is(err, ErrUniqueConstraint)
}

// IsForeignKeyConstraint checks if an error is a foreign key constraint violation.
func IsForeignKeyConstraint(err error) bool {
	return errors.Is(err, ErrForeignKeyConstraint)
}
