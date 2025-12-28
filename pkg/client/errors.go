// Package client provides error types for Prisma client operations.
package client

import (
	"errors"
	"fmt"
)

// Sentinel errors for common error conditions.
var (
	// ErrNotFound indicates that a record was not found.
	ErrNotFound = errors.New("prisma: record not found")

	// ErrUniqueConstraint indicates a unique constraint violation.
	ErrUniqueConstraint = errors.New("prisma: unique constraint violation")

	// ErrForeignKeyConstraint indicates a foreign key constraint violation.
	ErrForeignKeyConstraint = errors.New("prisma: foreign key constraint violation")

	// ErrNullConstraint indicates a null constraint violation.
	ErrNullConstraint = errors.New("prisma: null constraint violation")

	// ErrConnection indicates a database connection error.
	ErrConnection = errors.New("prisma: connection error")

	// ErrTransaction indicates a transaction error.
	ErrTransaction = errors.New("prisma: transaction error")

	// ErrTimeout indicates a query timeout.
	ErrTimeout = errors.New("prisma: query timeout")

	// ErrInvalidInput indicates invalid input data.
	ErrInvalidInput = errors.New("prisma: invalid input")
)

// PrismaError is a rich error type with additional context.
type PrismaError struct {
	// Code is the error code.
	Code string

	// Message is the human-readable error message.
	Message string

	// Model is the affected model (if applicable).
	Model string

	// Cause is the underlying error.
	Cause error
}

// Error implements the error interface.
func (e *PrismaError) Error() string {
	if e.Model != "" {
		return fmt.Sprintf("prisma [%s] %s: %s", e.Code, e.Model, e.Message)
	}
	return fmt.Sprintf("prisma [%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *PrismaError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target.
func (e *PrismaError) Is(target error) bool {
	return errors.Is(e.Cause, target)
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

// IsTimeout checks if an error is a timeout error.
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// NewNotFoundError creates a new not found error for a model.
func NewNotFoundError(model string) *PrismaError {
	return &PrismaError{
		Code:    "P2001",
		Message: fmt.Sprintf("No %s record was found", model),
		Model:   model,
		Cause:   ErrNotFound,
	}
}

// NewUniqueConstraintError creates a new unique constraint error.
func NewUniqueConstraintError(model, field string) *PrismaError {
	return &PrismaError{
		Code:    "P2002",
		Message: fmt.Sprintf("Unique constraint failed on %s.%s", model, field),
		Model:   model,
		Cause:   ErrUniqueConstraint,
	}
}
