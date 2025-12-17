// Package client provides lifecycle hooks.
package client

import (
	"context"
)

// HooksClient provides hook registration methods.
type HooksClient interface {
	// Hooks returns the hooks registry.
	Hooks() *HooksRegistry
}

// HooksRegistry manages model lifecycle hooks.
type HooksRegistry struct {
	beforeCreate map[string][]HookFunc
	afterCreate  map[string][]HookFunc
	beforeUpdate map[string][]HookFunc
	afterUpdate  map[string][]HookFunc
	beforeDelete map[string][]HookFunc
	afterDelete  map[string][]HookFunc
}

// NewHooksRegistry creates a new hooks registry.
func NewHooksRegistry() *HooksRegistry {
	return &HooksRegistry{
		beforeCreate: make(map[string][]HookFunc),
		afterCreate:  make(map[string][]HookFunc),
		beforeUpdate: make(map[string][]HookFunc),
		afterUpdate:  make(map[string][]HookFunc),
		beforeDelete: make(map[string][]HookFunc),
		afterDelete:  make(map[string][]HookFunc),
	}
}

// HookFunc is the function signature for hooks.
type HookFunc func(ctx context.Context, data *HookData) error

// HookData contains data passed to hooks.
type HookData struct {
	// Model is the model name.
	Model string

	// Operation is the operation type.
	Operation string

	// Data is the operation data.
	Data map[string]interface{}

	// Result is the operation result (for after hooks).
	Result interface{}
}

// OnBeforeCreate registers a before create hook.
func (h *HooksRegistry) OnBeforeCreate(model string, fn HookFunc) {
	h.beforeCreate[model] = append(h.beforeCreate[model], fn)
}

// OnAfterCreate registers an after create hook.
func (h *HooksRegistry) OnAfterCreate(model string, fn HookFunc) {
	h.afterCreate[model] = append(h.afterCreate[model], fn)
}

// OnBeforeUpdate registers a before update hook.
func (h *HooksRegistry) OnBeforeUpdate(model string, fn HookFunc) {
	h.beforeUpdate[model] = append(h.beforeUpdate[model], fn)
}

// OnAfterUpdate registers an after update hook.
func (h *HooksRegistry) OnAfterUpdate(model string, fn HookFunc) {
	h.afterUpdate[model] = append(h.afterUpdate[model], fn)
}

// OnBeforeDelete registers a before delete hook.
func (h *HooksRegistry) OnBeforeDelete(model string, fn HookFunc) {
	h.beforeDelete[model] = append(h.beforeDelete[model], fn)
}

// OnAfterDelete registers an after delete hook.
func (h *HooksRegistry) OnAfterDelete(model string, fn HookFunc) {
	h.afterDelete[model] = append(h.afterDelete[model], fn)
}

// RunBeforeCreate runs before create hooks.
func (h *HooksRegistry) RunBeforeCreate(ctx context.Context, model string, data *HookData) error {
	return h.runHooks(ctx, h.beforeCreate[model], data)
}

// RunAfterCreate runs after create hooks.
func (h *HooksRegistry) RunAfterCreate(ctx context.Context, model string, data *HookData) error {
	return h.runHooks(ctx, h.afterCreate[model], data)
}

// RunBeforeUpdate runs before update hooks.
func (h *HooksRegistry) RunBeforeUpdate(ctx context.Context, model string, data *HookData) error {
	return h.runHooks(ctx, h.beforeUpdate[model], data)
}

// RunAfterUpdate runs after update hooks.
func (h *HooksRegistry) RunAfterUpdate(ctx context.Context, model string, data *HookData) error {
	return h.runHooks(ctx, h.afterUpdate[model], data)
}

// RunBeforeDelete runs before delete hooks.
func (h *HooksRegistry) RunBeforeDelete(ctx context.Context, model string, data *HookData) error {
	return h.runHooks(ctx, h.beforeDelete[model], data)
}

// RunAfterDelete runs after delete hooks.
func (h *HooksRegistry) RunAfterDelete(ctx context.Context, model string, data *HookData) error {
	return h.runHooks(ctx, h.afterDelete[model], data)
}

func (h *HooksRegistry) runHooks(ctx context.Context, hooks []HookFunc, data *HookData) error {
	for _, fn := range hooks {
		if err := fn(ctx, data); err != nil {
			return err
		}
	}
	return nil
}
