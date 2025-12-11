// Package builder provides window function building functionality
package builder

import (
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// WindowFunction represents a window function (e.g., ROW_NUMBER, RANK, SUM OVER)
type WindowFunction struct {
	Function string // "ROW_NUMBER", "RANK", "DENSE_RANK", "SUM", "AVG", "COUNT", "MAX", "MIN", "LAG", "LEAD", "FIRST_VALUE", "LAST_VALUE"
	Field    string // Field to operate on (empty for ROW_NUMBER, RANK, etc.)
	Alias    string // Alias for the result
	Window   *WindowDefinition
}

// WindowDefinition defines the window frame
type WindowDefinition struct {
	PartitionByFields []string         // PARTITION BY columns
	OrderByFields     []sqlgen.OrderBy // ORDER BY for the window
	FrameSpec         *WindowFrame     // Frame specification (ROWS/RANGE BETWEEN)
}

// WindowFrame defines the window frame
type WindowFrame struct {
	Type          string // "ROWS" or "RANGE"
	Start         *FrameBound
	End           *FrameBound
	ExclusionType string // "EXCLUDE CURRENT ROW", "EXCLUDE GROUP", "EXCLUDE TIES", "EXCLUDE NO OTHERS"
}

// FrameBound defines a frame boundary
type FrameBound struct {
	Type   string // "UNBOUNDED PRECEDING", "PRECEDING", "CURRENT ROW", "FOLLOWING", "UNBOUNDED FOLLOWING"
	Offset *int   // Offset for PRECEDING/FOLLOWING (nil for UNBOUNDED/CURRENT ROW)
}

// WindowBuilder builds window functions
type WindowBuilder struct {
	functions []WindowFunction
}

// NewWindowBuilder creates a new window function builder
func NewWindowBuilder() *WindowBuilder {
	return &WindowBuilder{
		functions: []WindowFunction{},
	}
}

// RowNumber adds ROW_NUMBER() OVER window function
func (w *WindowBuilder) RowNumber(alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "ROW_NUMBER",
		Field:    "",
		Alias:    alias,
		Window:   window,
	})
	return w
}

// Rank adds RANK() OVER window function
func (w *WindowBuilder) Rank(alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "RANK",
		Field:    "",
		Alias:    alias,
		Window:   window,
	})
	return w
}

// DenseRank adds DENSE_RANK() OVER window function
func (w *WindowBuilder) DenseRank(alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "DENSE_RANK",
		Field:    "",
		Alias:    alias,
		Window:   window,
	})
	return w
}

// Sum adds SUM(field) OVER window function
func (w *WindowBuilder) Sum(field string, alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "SUM",
		Field:    field,
		Alias:    alias,
		Window:   window,
	})
	return w
}

// Avg adds AVG(field) OVER window function
func (w *WindowBuilder) Avg(field string, alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "AVG",
		Field:    field,
		Alias:    alias,
		Window:   window,
	})
	return w
}

// Count adds COUNT(field) OVER window function
func (w *WindowBuilder) Count(field string, alias string, window *WindowDefinition) *WindowBuilder {
	if field == "" {
		field = "*"
	}
	w.functions = append(w.functions, WindowFunction{
		Function: "COUNT",
		Field:    field,
		Alias:    alias,
		Window:   window,
	})
	return w
}

// Max adds MAX(field) OVER window function
func (w *WindowBuilder) Max(field string, alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "MAX",
		Field:    field,
		Alias:    alias,
		Window:   window,
	})
	return w
}

// Min adds MIN(field) OVER window function
func (w *WindowBuilder) Min(field string, alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "MIN",
		Field:    field,
		Alias:    alias,
		Window:   window,
	})
	return w
}

// Lag adds LAG(field, offset, default) OVER window function
func (w *WindowBuilder) Lag(field string, offset int, defaultValue interface{}, alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "LAG",
		Field:    field,
		Alias:    alias,
		Window:   window,
	})
	return w
}

// Lead adds LEAD(field, offset, default) OVER window function
func (w *WindowBuilder) Lead(field string, offset int, defaultValue interface{}, alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "LEAD",
		Field:    field,
		Alias:    alias,
		Window:   window,
	})
	return w
}

// FirstValue adds FIRST_VALUE(field) OVER window function
func (w *WindowBuilder) FirstValue(field string, alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "FIRST_VALUE",
		Field:    field,
		Alias:    alias,
		Window:   window,
	})
	return w
}

// LastValue adds LAST_VALUE(field) OVER window function
func (w *WindowBuilder) LastValue(field string, alias string, window *WindowDefinition) *WindowBuilder {
	w.functions = append(w.functions, WindowFunction{
		Function: "LAST_VALUE",
		Field:    field,
		Alias:    alias,
		Window:   window,
	})
	return w
}

// Build returns the list of window functions
func (w *WindowBuilder) Build() []WindowFunction {
	return w.functions
}

// NewWindowDefinition creates a new window definition
func NewWindowDefinition() *WindowDefinition {
	return &WindowDefinition{
		PartitionByFields: []string{},
		OrderByFields:     []sqlgen.OrderBy{},
		FrameSpec:         nil,
	}
}

// PartitionBy sets the PARTITION BY clause
func (w *WindowDefinition) PartitionBy(fields ...string) *WindowDefinition {
	w.PartitionByFields = fields
	return w
}

// OrderBy adds an ORDER BY clause to the window
func (w *WindowDefinition) OrderBy(field string, direction string) *WindowDefinition {
	w.OrderByFields = append(w.OrderByFields, sqlgen.OrderBy{
		Field:     field,
		Direction: direction,
	})
	return w
}

// Frame sets the window frame
func (w *WindowDefinition) Frame(frame *WindowFrame) *WindowDefinition {
	w.FrameSpec = frame
	return w
}

// NewWindowFrame creates a new window frame
func NewWindowFrame(frameType string) *WindowFrame {
	return &WindowFrame{
		Type:          frameType,
		Start:         nil,
		End:           nil,
		ExclusionType: "",
	}
}

// Between sets the frame boundaries
func (f *WindowFrame) Between(start, end *FrameBound) *WindowFrame {
	f.Start = start
	f.End = end
	return f
}

// Exclusion sets the frame exclusion
func (f *WindowFrame) Exclusion(exclusion string) *WindowFrame {
	f.ExclusionType = exclusion
	return f
}

// NewFrameBound creates a new frame boundary
func NewFrameBound(boundType string, offset *int) *FrameBound {
	return &FrameBound{
		Type:   boundType,
		Offset: offset,
	}
}

// UnboundedPreceding creates UNBOUNDED PRECEDING frame bound
func UnboundedPreceding() *FrameBound {
	return NewFrameBound("UNBOUNDED PRECEDING", nil)
}

// Preceding creates N PRECEDING frame bound
func Preceding(offset int) *FrameBound {
	return NewFrameBound("PRECEDING", &offset)
}

// CurrentRow creates CURRENT ROW frame bound
func CurrentRow() *FrameBound {
	return NewFrameBound("CURRENT ROW", nil)
}

// Following creates N FOLLOWING frame bound
func Following(offset int) *FrameBound {
	return NewFrameBound("FOLLOWING", &offset)
}

// UnboundedFollowing creates UNBOUNDED FOLLOWING frame bound
func UnboundedFollowing() *FrameBound {
	return NewFrameBound("UNBOUNDED FOLLOWING", nil)
}
