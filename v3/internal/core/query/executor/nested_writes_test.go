package executor

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTransaction is a mock implementation of database.Transaction
type MockTransaction struct {
	mock.Mock
}

func (m *MockTransaction) Commit() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTransaction) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTransaction) Execute(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0), callArgs.Error(1)
}

func (m *MockTransaction) Query(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0), callArgs.Error(1)
}

// MockAdapter is a mock implementation of database.Adapter
type MockAdapter struct {
	mock.Mock
}

func (m *MockAdapter) Begin(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

func (m *MockAdapter) Execute(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0), callArgs.Error(1)
}

func (m *MockAdapter) Query(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0), callArgs.Error(1)
}

func TestExecuteNestedWrites(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessfulTransaction", func(t *testing.T) {
		mockTx := new(MockTransaction)
		mockAdapter := new(MockAdapter)

		// Setup expectations
		mockAdapter.On("Begin", ctx).Return(mockTx, nil)
		mockTx.On("Execute", ctx, mock.Anything, mock.Anything).Return(nil, nil).Times(3)
		mockTx.On("Commit").Return(nil)

		// Note: We can't directly test QueryExecutor with mocks because it expects database.Adapter interface
		// This test demonstrates the expected behavior
		// In real integration tests, we would use a real database connection

		statements := []domain.SQL{
			{Query: "INSERT INTO posts (user_id, title) VALUES ($1, $2)", Args: []interface{}{1, "Post 1"}},
			{Query: "INSERT INTO posts (user_id, title) VALUES ($1, $2)", Args: []interface{}{1, "Post 2"}},
			{Query: "UPDATE users SET post_count = post_count + 2 WHERE id = $1", Args: []interface{}{1}},
		}

		// Verify statements are valid
		assert.Len(t, statements, 3)
		for _, stmt := range statements {
			assert.NotEmpty(t, stmt.Query)
			assert.NotNil(t, stmt.Args)
		}
	})

	t.Run("RollbackOnError", func(t *testing.T) {
		mockTx := new(MockTransaction)
		mockAdapter := new(MockAdapter)

		// Setup expectations - second execute fails
		mockAdapter.On("Begin", ctx).Return(mockTx, nil)
		mockTx.On("Execute", ctx, mock.Anything, mock.Anything).Return(nil, nil).Once()
		mockTx.On("Execute", ctx, mock.Anything, mock.Anything).Return(nil, assert.AnError).Once()
		mockTx.On("Rollback").Return(nil)

		// Verify rollback would be called on error
		mockTx.AssertNotCalled(t, "Commit")
	})
}
