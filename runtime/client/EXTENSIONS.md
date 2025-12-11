# Extensions API Documentation

The Extensions API provides a structured way to extend Prisma client behavior with hooks for query and mutation operations.

## Overview

Extensions allow you to:
- Intercept queries and mutations before and after execution
- Transform results
- Add logging, timing, error handling
- Implement custom business logic

## Basic Usage

### Adding an Extension

```go
import "github.com/satishbabariya/prisma-go/runtime/client"

client, _ := client.NewPrismaClient("postgresql", "postgresql://...")

// Add a logging extension
client.UseExtension(client.LoggingExtension(func(format string, args ...interface{}) {
    log.Printf(format, args...)
}))
```

### Extension Interface

```go
type Extension struct {
    Name string
    
    // Query hooks
    BeforeQuery QueryHook
    AfterQuery  QueryHook
    
    // Mutation hooks
    BeforeMutation MutationHook
    AfterMutation  MutationHook
}
```

### Extension Context

Each hook receives an `ExtensionContext` with:
- `Context`: The request context
- `Model`: Model name (e.g., "User", "Post")
- `Operation`: Operation type (e.g., "findMany", "create", "update", "delete")
- `Args`: Operation arguments
- `Result`: Operation result (set in AfterQuery/AfterMutation)
- `Error`: Operation error (set in AfterQuery/AfterMutation)
- `Duration`: Operation duration
- `StartTime`: Operation start time
- `EndTime`: Operation end time

## Built-in Extensions

### LoggingExtension

Logs all operations:

```go
client.UseExtension(client.LoggingExtension(func(format string, args ...interface{}) {
    log.Printf(format, args...)
}))
```

### TimingExtension

Measures operation timing:

```go
client.UseExtension(client.TimingExtension(func(model string, operation string, duration time.Duration) {
    metrics.RecordDuration(model, operation, duration)
}))
```

### ErrorHandlingExtension

Handles errors:

```go
client.UseExtension(client.ErrorHandlingExtension(func(model string, operation string, err error) {
    sentry.CaptureException(err)
}))
```

### ResultTransformationExtension

Transforms results:

```go
client.UseExtension(client.ResultTransformationExtension(func(ctx *client.ExtensionContext, result interface{}) interface{} {
    // Transform result
    return result
}))
```

## Custom Extensions

Create custom extensions by implementing the `Extension` interface:

```go
customExt := client.Extension{
    Name: "custom",
    BeforeQuery: func(ctx *client.ExtensionContext, next func() error) error {
        // Pre-query logic
        log.Printf("Executing query: %s.%s", ctx.Model, ctx.Operation)
        return next()
    },
    AfterQuery: func(ctx *client.ExtensionContext, next func() error) error {
        // Post-query logic
        if ctx.Error != nil {
            log.Printf("Query failed: %v", ctx.Error)
        } else {
            log.Printf("Query succeeded in %v", ctx.Duration)
        }
        return next()
    },
    BeforeMutation: func(ctx *client.ExtensionContext, next func() error) error {
        // Pre-mutation logic
        return next()
    },
    AfterMutation: func(ctx *client.ExtensionContext, next func() error) error {
        // Post-mutation logic
        return next()
    },
}

client.UseExtension(customExt)
```

## Extension Chaining

Extensions are executed in order:
1. BeforeQuery/BeforeMutation hooks execute in registration order
2. The actual operation executes
3. AfterQuery/AfterMutation hooks execute in reverse order

This allows you to:
- Set up context in BeforeQuery
- Clean up in AfterQuery
- Transform results in AfterQuery (last extension gets final result)

## Usage with Generated Code

When using generated model clients, extensions work automatically:

```go
// Extensions are applied to all operations
client.UseExtension(client.LoggingExtension(log.Printf))

// Generated client methods automatically use extensions
userClient := generated.NewUserClient(client)
users, err := userClient.FindMany(ctx) // Extension hooks are called
```

## Best Practices

1. **Keep extensions lightweight**: Extensions run on every operation
2. **Handle errors properly**: Don't swallow errors in hooks
3. **Use context**: Pass context through hooks for cancellation/timeout
4. **Avoid side effects**: Extensions should be idempotent when possible
5. **Order matters**: Register extensions in the order you want them executed

## Examples

### Audit Logging

```go
client.UseExtension(client.Extension{
    Name: "audit",
    AfterMutation: func(ctx *client.ExtensionContext, next func() error) error {
        if ctx.Error == nil {
            audit.Log(ctx.Context, ctx.Model, ctx.Operation, ctx.Args, ctx.Result)
        }
        return next()
    },
})
```

### Performance Monitoring

```go
client.UseExtension(client.TimingExtension(func(model, operation string, duration time.Duration) {
    if duration > 1*time.Second {
        alert.SlowQuery(model, operation, duration)
    }
}))
```

### Result Caching

```go
client.UseExtension(client.Extension{
    Name: "cache",
    BeforeQuery: func(ctx *client.ExtensionContext, next func() error) error {
        // Check cache
        if cached := cache.Get(ctx.Model, ctx.Operation, ctx.Args); cached != nil {
            ctx.Result = cached
            return nil // Skip execution
        }
        return next()
    },
    AfterQuery: func(ctx *client.ExtensionContext, next func() error) error {
        // Store in cache
        if ctx.Error == nil {
            cache.Set(ctx.Model, ctx.Operation, ctx.Args, ctx.Result)
        }
        return next()
    },
})
```

