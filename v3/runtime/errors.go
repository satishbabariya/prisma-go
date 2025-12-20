// Package runtime provides the base runtime for generated Prisma clients.
package runtime

import (
	"context"
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

// Additional error types
var (
	// ErrValidationFailed indicates data validation failure
	ErrValidationFailed = errors.New("validation failed")

	// ErrRetryExhausted indicates all retry attempts failed
	ErrRetryExhausted = errors.New("retry attempts exhausted")
)

// PrismaError represents a detailed error from Prisma operations
type PrismaError struct {
	Code      string                 // Error code (e.g., "P2002", "P2025")
	Message   string                 // Human-readable error message
	Meta      map[string]interface{} // Additional error metadata
	Cause     error                  // Underlying error cause
	Retryable bool                   // Whether the error is retryable
	Model     string                 // Model name if applicable
	Field     string                 // Field name if applicable
}

// Error implements the error interface
func (e *PrismaError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return e.Message
}

// Unwrap implements error unwrapping
func (e *PrismaError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison
func (e *PrismaError) Is(target error) bool {
	return errors.Is(e.Cause, target)
}

// NewPrismaError creates a new PrismaError
func NewPrismaError(code, message string) *PrismaError {
	return &PrismaError{
		Code:    code,
		Message: message,
		Meta:    make(map[string]interface{}),
	}
}

// WithCause adds the underlying cause
func (e *PrismaError) WithCause(err error) *PrismaError {
	e.Cause = err
	e.Retryable = isRetryableError(err)
	return e
}

// WithModel adds the model name
func (e *PrismaError) WithModel(model string) *PrismaError {
	e.Model = model
	return e
}

// WithField adds the field name
func (e *PrismaError) WithField(field string) *PrismaError {
	e.Field = field
	return e
}

// WithMeta adds metadata
func (e *PrismaError) WithMeta(key string, value interface{}) *PrismaError {
	e.Meta[key] = value
	return e
}

// ClassifyError classifies a database error into a PrismaError
func ClassifyError(err error) error {
	if err == nil {
		return nil
	}

	// Check if already a PrismaError
	var prismaErr *PrismaError
	if errors.As(err, &prismaErr) {
		return prismaErr
	}

	// Check for context errors first
	if errors.Is(err, context.DeadlineExceeded) {
		return NewPrismaError("P1008", "Operation timed out").
			WithCause(ErrTimeout).
			WithMeta("retryable", true)
	}
	if errors.Is(err, context.Canceled) {
		return NewPrismaError("P1017", "Operation was canceled").
			WithCause(ErrCanceled).
			WithMeta("retryable", false)
	}

	// Parse database-specific errors
	errMsg := fmt.Sprintf("%v", err)

	// Connection errors
	if containsAny(errMsg, []string{"connection", "connect"}) {
		return NewPrismaError("P1001", "Cannot connect to database").
			WithCause(ErrConnectionFailed).
			WithMeta("retryable", true)
	}

	// Unique constraint violations
	if containsAny(errMsg, []string{"unique", "duplicate"}) {
		return NewPrismaError("P2002", "Unique constraint violation").
			WithCause(ErrUniqueConstraint).
			WithMeta("retryable", false)
	}

	// Foreign key constraint violations
	if containsAny(errMsg, []string{"foreign key"}) {
		return NewPrismaError("P2003", "Foreign key constraint violation").
			WithCause(ErrForeignKeyConstraint).
			WithMeta("retryable", false)
	}

	// Default error
	return NewPrismaError("P0000", "Unknown error").
		WithCause(err).
		WithMeta("retryable", false)
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Always retry connection errors
	if errors.Is(err, ErrConnectionFailed) {
		return true
	}

	// Retry timeouts
	if errors.Is(err, ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check error message for retryable patterns
	errMsg := fmt.Sprintf("%v", err)
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"timeout",
		"timed out",
		"deadlock",
		"lock wait timeout",
		"too many connections",
	}

	for _, pattern := range retryablePatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}
