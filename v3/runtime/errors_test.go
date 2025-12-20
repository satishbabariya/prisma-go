package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  string
		wantCause error
	}{
		{
			name:      "nil error",
			err:       nil,
			wantCode:  "",
			wantCause: nil,
		},
		{
			name:      "context deadline exceeded",
			err:       context.DeadlineExceeded,
			wantCode:  "P1008",
			wantCause: ErrTimeout,
		},
		{
			name:      "context canceled",
			err:       context.Canceled,
			wantCode:  "P1017",
			wantCause: ErrCanceled,
		},
		{
			name:      "connection error",
			err:       errors.New("connection refused"),
			wantCode:  "P1001",
			wantCause: ErrConnectionFailed,
		},
		{
			name:      "unique constraint",
			err:       errors.New("duplicate key value violates unique constraint"),
			wantCode:  "P2002",
			wantCause: ErrUniqueConstraint,
		},
		{
			name:      "foreign key constraint",
			err:       errors.New("foreign key constraint failed"),
			wantCode:  "P2003",
			wantCause: ErrForeignKeyConstraint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)

			if tt.err == nil {
				assert.Nil(t, result)
				return
			}

			var prismaErr *PrismaError
			require.True(t, errors.As(result, &prismaErr))
			assert.Equal(t, tt.wantCode, prismaErr.Code)

			if tt.wantCause != nil {
				assert.True(t, errors.Is(result, tt.wantCause))
			}
		})
	}
}

func TestPrismaError_Methods(t *testing.T) {
	t.Run("WithCause adds cause and sets retryable", func(t *testing.T) {
		err := NewPrismaError("P1001", "Connection failed").
			WithCause(ErrConnectionFailed)

		assert.Equal(t, ErrConnectionFailed, err.Cause)
		assert.True(t, err.Retryable)
	})

	t.Run("WithModel adds model name", func(t *testing.T) {
		err := NewPrismaError("P2025", "Not found").
			WithModel("User")

		assert.Equal(t, "User", err.Model)
	})

	t.Run("WithField adds field name", func(t *testing.T) {
		err := NewPrismaError("P2002", "Unique violation").
			WithField("email")

		assert.Equal(t, "email", err.Field)
	})

	t.Run("WithMeta adds metadata", func(t *testing.T) {
		err := NewPrismaError("P0000", "Error").
			WithMeta("key", "value")

		assert.Equal(t, "value", err.Meta["key"])
	})

	t.Run("Error formats correctly", func(t *testing.T) {
		err := NewPrismaError("P2025", "Record not found")
		assert.Equal(t, "[P2025] Record not found", err.Error())
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"connection failed", ErrConnectionFailed, true},
		{"timeout", ErrTimeout, true},
		{"context deadline", context.DeadlineExceeded, true},
		{"connection refused", errors.New("connection refused"), true},
		{"deadlock", errors.New("deadlock detected"), true},
		{"unique constraint", ErrUniqueConstraint, false},
		{"foreign key", ErrForeignKeyConstraint, false},
		{"validation error", ErrValidationFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRetry_Success(t *testing.T) {
	attempts := 0
	err := Retry(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return ErrConnectionFailed // Use a retryable error instead
		}
		return nil
	}, WithMaxAttempts(3))

	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "Should succeed on second attempt")
}

func TestRetry_NonRetryableError(t *testing.T) {
	attempts := 0
	err := Retry(context.Background(), func() error {
		attempts++
		return NewPrismaError("P2002", "Unique violation").
			WithCause(ErrUniqueConstraint)
	}, WithMaxAttempts(3))

	require.Error(t, err)
	assert.Equal(t, 1, attempts, "Should not retry non-retryable errors")
}

func TestRetry_ExhaustedAttempts(t *testing.T) {
	attempts := 0
	err := Retry(context.Background(), func() error {
		attempts++
		return ErrConnectionFailed
	}, WithMaxAttempts(3), WithInitialDelay(1*time.Millisecond))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRetryExhausted))
	assert.Equal(t, 3, attempts)
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	errChan := make(chan error, 1)

	go func() {
		err := Retry(ctx, func() error {
			attempts++
			return ErrConnectionFailed
		}, WithMaxAttempts(10), WithInitialDelay(100*time.Millisecond))
		errChan <- err
	}()

	// Cancel after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-errChan
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Less(t, attempts, 10, "Should stop retrying after cancellation")
}

func TestRetryWithResult(t *testing.T) {
	attempts := 0
	result, err := RetryWithResult(context.Background(), func() (string, error) {
		attempts++
		if attempts < 2 {
			return "", ErrConnectionFailed
		}
		return "success", nil
	}, WithMaxAttempts(3), WithInitialDelay(1*time.Millisecond))

	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 2, attempts)
}

func TestRetryConfig_Options(t *testing.T) {
	config := DefaultRetryConfig()

	WithMaxAttempts(5)(config)
	assert.Equal(t, 5, config.MaxAttempts)

	WithInitialDelay(200 * time.Millisecond)(config)
	assert.Equal(t, 200*time.Millisecond, config.InitialDelay)

	WithMaxDelay(10 * time.Second)(config)
	assert.Equal(t, 10*time.Second, config.MaxDelay)

	WithBackoffFactor(3.0)(config)
	assert.Equal(t, 3.0, config.BackoffFactor)
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	attempts := 0
	startTime := time.Now()

	Retry(context.Background(), func() error {
		attempts++
		return ErrConnectionFailed
	}, WithMaxAttempts(3), WithInitialDelay(10*time.Millisecond), WithBackoffFactor(2.0))

	elapsed := time.Since(startTime)

	// With 3 attempts and 2x backoff: 0ms + 10ms + 20ms = at least 30ms
	assert.GreaterOrEqual(t, elapsed, 30*time.Millisecond)
	assert.Equal(t, 3, attempts)
}
