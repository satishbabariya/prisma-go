// Package runtime provides lifecycle hooks for Prisma clients.
package runtime

import (
	"context"
	"sync"
)

// HookType represents the type of hook event.
type HookType string

const (
	// BeforeCreate is called before a create operation.
	BeforeCreate HookType = "beforeCreate"
	// AfterCreate is called after a create operation.
	AfterCreate HookType = "afterCreate"

	// BeforeUpdate is called before an update operation.
	BeforeUpdate HookType = "beforeUpdate"
	// AfterUpdate is called after an update operation.
	AfterUpdate HookType = "afterUpdate"

	// BeforeDelete is called before a delete operation.
	BeforeDelete HookType = "beforeDelete"
	// AfterDelete is called after a delete operation.
	AfterDelete HookType = "afterDelete"

	// BeforeQuery is called before a query operation.
	BeforeQuery HookType = "beforeQuery"
	// AfterQuery is called after a query operation.
	AfterQuery HookType = "afterQuery"
)

// HookContext contains information passed to hooks.
type HookContext struct {
	// Model is the model being operated on.
	Model string

	// Operation is the operation type.
	Operation string

	// Data is the input data (for create/update).
	Data map[string]interface{}

	// Result is the result data (for after hooks).
	Result interface{}

	// Error is any error that occurred (for after hooks).
	Error error

	// Context is the request context.
	Context context.Context
}

// HookFunc is a function that can be registered as a hook.
type HookFunc func(ctx *HookContext) error

// Hooks manages lifecycle hooks for models.
type Hooks struct {
	hooks map[string]map[HookType][]HookFunc
	mu    sync.RWMutex
}

// NewHooks creates a new Hooks instance.
func NewHooks() *Hooks {
	return &Hooks{
		hooks: make(map[string]map[HookType][]HookFunc),
	}
}

// Register registers a hook for a model and hook type.
func (h *Hooks) Register(model string, hookType HookType, fn HookFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.hooks[model] == nil {
		h.hooks[model] = make(map[HookType][]HookFunc)
	}

	h.hooks[model][hookType] = append(h.hooks[model][hookType], fn)
}

// Execute executes all hooks for a model and hook type.
func (h *Hooks) Execute(ctx *HookContext, hookType HookType) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	modelHooks, ok := h.hooks[ctx.Model]
	if !ok {
		return nil
	}

	hooks, ok := modelHooks[hookType]
	if !ok {
		return nil
	}

	for _, fn := range hooks {
		if err := fn(ctx); err != nil {
			return err
		}
	}

	return nil
}

// OnBeforeCreate registers a before create hook.
func (h *Hooks) OnBeforeCreate(model string, fn HookFunc) {
	h.Register(model, BeforeCreate, fn)
}

// OnAfterCreate registers an after create hook.
func (h *Hooks) OnAfterCreate(model string, fn HookFunc) {
	h.Register(model, AfterCreate, fn)
}

// OnBeforeUpdate registers a before update hook.
func (h *Hooks) OnBeforeUpdate(model string, fn HookFunc) {
	h.Register(model, BeforeUpdate, fn)
}

// OnAfterUpdate registers an after update hook.
func (h *Hooks) OnAfterUpdate(model string, fn HookFunc) {
	h.Register(model, AfterUpdate, fn)
}

// OnBeforeDelete registers a before delete hook.
func (h *Hooks) OnBeforeDelete(model string, fn HookFunc) {
	h.Register(model, BeforeDelete, fn)
}

// OnAfterDelete registers an after delete hook.
func (h *Hooks) OnAfterDelete(model string, fn HookFunc) {
	h.Register(model, AfterDelete, fn)
}

// OnBeforeQuery registers a before query hook.
func (h *Hooks) OnBeforeQuery(model string, fn HookFunc) {
	h.Register(model, BeforeQuery, fn)
}

// OnAfterQuery registers an after query hook.
func (h *Hooks) OnAfterQuery(model string, fn HookFunc) {
	h.Register(model, AfterQuery, fn)
}

// Clear removes all hooks.
func (h *Hooks) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hooks = make(map[string]map[HookType][]HookFunc)
}

// ClearModel removes all hooks for a specific model.
func (h *Hooks) ClearModel(model string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.hooks, model)
}
