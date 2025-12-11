// Package client provides extensions API for Prisma client.
package client

import (
	"context"
	"time"
)

// ExtensionContext provides context for extension hooks
type ExtensionContext struct {
	Context   context.Context
	Model     string      // Model name (e.g., "User", "Post")
	Operation string      // Operation type (e.g., "findMany", "create", "update", "delete")
	Args      interface{} // Operation arguments
	Result    interface{} // Operation result (set in AfterQuery/AfterMutation)
	Error     error       // Operation error (set in AfterQuery/AfterMutation)
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
}

// QueryHook is called before and after query operations
type QueryHook func(ctx *ExtensionContext, next func() error) error

// MutationHook is called before and after mutation operations
type MutationHook func(ctx *ExtensionContext, next func() error) error

// Extension defines hooks for extending Prisma client behavior
type Extension struct {
	Name string

	// Query hooks
	BeforeQuery QueryHook
	AfterQuery  QueryHook

	// Mutation hooks
	BeforeMutation MutationHook
	AfterMutation  MutationHook
}

// ExtensionResult wraps the result of an extension chain
type ExtensionResult struct {
	Result interface{}
	Error  error
}

// ExtensionChain manages a chain of extensions
type ExtensionChain struct {
	extensions []Extension
}

// NewExtensionChain creates a new extension chain
func NewExtensionChain() *ExtensionChain {
	return &ExtensionChain{
		extensions: []Extension{},
	}
}

// Add adds an extension to the chain
func (ec *ExtensionChain) Add(ext Extension) {
	ec.extensions = append(ec.extensions, ext)
}

// ExecuteQuery executes a query operation through the extension chain
func (ec *ExtensionChain) ExecuteQuery(ctx context.Context, model string, operation string, args interface{}, exec func() (interface{}, error)) (interface{}, error) {
	extCtx := &ExtensionContext{
		Context:   ctx,
		Model:     model,
		Operation: operation,
		Args:      args,
		StartTime: time.Now(),
	}

	// Execute BeforeQuery hooks
	for _, ext := range ec.extensions {
		if ext.BeforeQuery != nil {
			if err := ext.BeforeQuery(extCtx, func() error { return nil }); err != nil {
				return nil, err
			}
		}
	}

	// Execute the actual query
	result, err := exec()
	extCtx.Result = result
	extCtx.Error = err
	extCtx.EndTime = time.Now()
	extCtx.Duration = extCtx.EndTime.Sub(extCtx.StartTime)

	// Execute AfterQuery hooks
	for i := len(ec.extensions) - 1; i >= 0; i-- {
		ext := ec.extensions[i]
		if ext.AfterQuery != nil {
			if err := ext.AfterQuery(extCtx, func() error { return err }); err != nil {
				return result, err
			}
		}
	}

	return result, err
}

// ExecuteMutation executes a mutation operation through the extension chain
func (ec *ExtensionChain) ExecuteMutation(ctx context.Context, model string, operation string, args interface{}, exec func() (interface{}, error)) (interface{}, error) {
	extCtx := &ExtensionContext{
		Context:   ctx,
		Model:     model,
		Operation: operation,
		Args:      args,
		StartTime: time.Now(),
	}

	// Execute BeforeMutation hooks
	for _, ext := range ec.extensions {
		if ext.BeforeMutation != nil {
			if err := ext.BeforeMutation(extCtx, func() error { return nil }); err != nil {
				return nil, err
			}
		}
	}

	// Execute the actual mutation
	result, err := exec()
	extCtx.Result = result
	extCtx.Error = err
	extCtx.EndTime = time.Now()
	extCtx.Duration = extCtx.EndTime.Sub(extCtx.StartTime)

	// Execute AfterMutation hooks
	for i := len(ec.extensions) - 1; i >= 0; i-- {
		ext := ec.extensions[i]
		if ext.AfterMutation != nil {
			if err := ext.AfterMutation(extCtx, func() error { return err }); err != nil {
				return result, err
			}
		}
	}

	return result, err
}

// LoggingExtension creates an extension that logs operations
func LoggingExtension(logger func(format string, args ...interface{})) Extension {
	return Extension{
		Name: "logging",
		BeforeQuery: func(ctx *ExtensionContext, next func() error) error {
			logger("[Query] %s.%s - Args: %v", ctx.Model, ctx.Operation, ctx.Args)
			return next()
		},
		AfterQuery: func(ctx *ExtensionContext, next func() error) error {
			if ctx.Error != nil {
				logger("[Query] %s.%s - Error: %v (Duration: %v)", ctx.Model, ctx.Operation, ctx.Error, ctx.Duration)
			} else {
				logger("[Query] %s.%s - Success (Duration: %v)", ctx.Model, ctx.Operation, ctx.Duration)
			}
			return next()
		},
		BeforeMutation: func(ctx *ExtensionContext, next func() error) error {
			logger("[Mutation] %s.%s - Args: %v", ctx.Model, ctx.Operation, ctx.Args)
			return next()
		},
		AfterMutation: func(ctx *ExtensionContext, next func() error) error {
			if ctx.Error != nil {
				logger("[Mutation] %s.%s - Error: %v (Duration: %v)", ctx.Model, ctx.Operation, ctx.Error, ctx.Duration)
			} else {
				logger("[Mutation] %s.%s - Success (Duration: %v)", ctx.Model, ctx.Operation, ctx.Duration)
			}
			return next()
		},
	}
}

// TimingExtension creates an extension that measures operation timing
func TimingExtension(onTiming func(model string, operation string, duration time.Duration)) Extension {
	return Extension{
		Name: "timing",
		AfterQuery: func(ctx *ExtensionContext, next func() error) error {
			if onTiming != nil {
				onTiming(ctx.Model, ctx.Operation, ctx.Duration)
			}
			return next()
		},
		AfterMutation: func(ctx *ExtensionContext, next func() error) error {
			if onTiming != nil {
				onTiming(ctx.Model, ctx.Operation, ctx.Duration)
			}
			return next()
		},
	}
}

// ErrorHandlingExtension creates an extension that handles errors
func ErrorHandlingExtension(onError func(model string, operation string, err error)) Extension {
	return Extension{
		Name: "error-handling",
		AfterQuery: func(ctx *ExtensionContext, next func() error) error {
			if ctx.Error != nil && onError != nil {
				onError(ctx.Model, ctx.Operation, ctx.Error)
			}
			return next()
		},
		AfterMutation: func(ctx *ExtensionContext, next func() error) error {
			if ctx.Error != nil && onError != nil {
				onError(ctx.Model, ctx.Operation, ctx.Error)
			}
			return next()
		},
	}
}

// ResultTransformationExtension creates an extension that transforms results
func ResultTransformationExtension(transform func(ctx *ExtensionContext, result interface{}) interface{}) Extension {
	return Extension{
		Name: "result-transformation",
		AfterQuery: func(ctx *ExtensionContext, next func() error) error {
			if ctx.Result != nil && transform != nil {
				ctx.Result = transform(ctx, ctx.Result)
			}
			return next()
		},
		AfterMutation: func(ctx *ExtensionContext, next func() error) error {
			if ctx.Result != nil && transform != nil {
				ctx.Result = transform(ctx, ctx.Result)
			}
			return next()
		},
	}
}
