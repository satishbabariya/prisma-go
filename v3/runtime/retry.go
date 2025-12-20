// Package runtime provides retry utilities for database operations
package runtime

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts   int           // Maximum number of retry attempts
	InitialDelay  time.Duration // Initial delay before first retry
	MaxDelay      time.Duration // Maximum delay between retries
	BackoffFactor float64       // Exponential backoff multiplier
	Jitter        bool          // Add randomness to delay
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// RetryOptions allows customization of retry behavior
type RetryOptions func(*RetryConfig)

// WithMaxAttempts sets the maximum retry attempts
func WithMaxAttempts(n int) RetryOptions {
	return func(c *RetryConfig) {
		c.MaxAttempts = n
	}
}

// WithInitialDelay sets the initial retry delay
func WithInitialDelay(d time.Duration) RetryOptions {
	return func(c *RetryConfig) {
		c.InitialDelay = d
	}
}

// WithMaxDelay sets the maximum retry delay
func WithMaxDelay(d time.Duration) RetryOptions {
	return func(c *RetryConfig) {
		c.MaxDelay = d
	}
}

// WithBackoffFactor sets the exponential backoff factor
func WithBackoffFactor(f float64) RetryOptions {
	return func(c *RetryConfig) {
		c.BackoffFactor = f
	}
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, fn func() error, opts ...RetryOptions) error {
	config := DefaultRetryConfig()
	for _, opt := range opts {
		opt(config)
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		classified := ClassifyError(err)
		var prismaErr *PrismaError
		if errors.As(classified, &prismaErr) && !prismaErr.Retryable {
			// Non-retryable error, fail immediately
			return classified
		}

		// Don't retry on context cancellation
		if errors.Is(err, context.Canceled) {
			return err
		}

		// Check if we've exhausted attempts
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Apply jitter if enabled
		actualDelay := delay
		if config.Jitter {
			// Add ±25% jitter
			jitterRange := delay / 4
			jitterAmount := time.Duration(rand.Int63n(int64(jitterRange) * 2))
			actualDelay = delay - jitterRange + jitterAmount
		}

		// Wait with context awareness
		select {
		case <-time.After(actualDelay):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * config.BackoffFactor)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	// All retries exhausted
	return fmt.Errorf("%w after %d attempts: %v", ErrRetryExhausted, config.MaxAttempts, lastErr)
}

// RetryWithResult executes a function with retry logic and returns a result
func RetryWithResult[T any](ctx context.Context, fn func() (T, error), opts ...RetryOptions) (T, error) {
	var result T
	var resultErr error

	err := Retry(ctx, func() error {
		var fnErr error
		result, fnErr = fn()
		resultErr = fnErr
		return fnErr
	}, opts...)

	if err != nil {
		return result, err
	}

	return result, resultErr
}
