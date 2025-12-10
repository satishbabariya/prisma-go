// Package schemaast provides table rendering functionality for Prisma schema formatting.
package formatting

import (


	"strings"
)

const columnSpacing = 1

// Row represents a row in a table format.
type Row interface {
	isRow()
}

// RegularRow represents a row with columns aligned by the table format.
// The suffix is an arbitrary string that does not influence the table layout (used for end of line comments).
type RegularRow struct {
	Columns []string
	Suffix  string
}

func (RegularRow) isRow() {}

// InterleavedRow represents a row without columns (non-table content).
type InterleavedRow struct {
	Text string
}

func (InterleavedRow) isRow() {}

// TableFormat provides table formatting functionality for aligned columns.
type TableFormat struct {
	table []Row
}

// NewTableFormat creates a new table format.
func NewTableFormat() *TableFormat {
	return &TableFormat{
		table: make([]Row, 0),
	}
}

// Reset clears the table format.
func (tf *TableFormat) Reset() {
	tf.table = make([]Row, 0)
}

// ColumnLockedWriterFor returns a writer for a specific column index in the current row.
// The returned function can be used to write to that column.
func (tf *TableFormat) ColumnLockedWriterFor(index int) func(string) {
	if len(tf.table) == 0 {
		tf.StartNewLine()
	}

	lastRow := tf.table[len(tf.table)-1]
	switch row := lastRow.(type) {
	case *InterleavedRow:
		// Convert interleaved row to regular row
		newRow := &RegularRow{
			Columns: make([]string, index+1),
			Suffix:  row.Text,
		}
		tf.table[len(tf.table)-1] = newRow
		colIdx := index
		return func(text string) {
			if regularRow, ok := tf.table[len(tf.table)-1].(*RegularRow); ok {
				for len(regularRow.Columns) <= colIdx {
					regularRow.Columns = append(regularRow.Columns, "")
				}
				regularRow.Columns[colIdx] = text
			}
		}
	case *RegularRow:
		// Ensure columns slice is large enough
		for len(row.Columns) <= index {
			row.Columns = append(row.Columns, "")
		}
		colIdx := index
		return func(text string) {
			if regularRow, ok := tf.table[len(tf.table)-1].(*RegularRow); ok {
				for len(regularRow.Columns) <= colIdx {
					regularRow.Columns = append(regularRow.Columns, "")
				}
				regularRow.Columns[colIdx] = text
			}
		}
	default:
		panic("unexpected row type")
	}
}

// Interleave adds an interleaved row (non-table content).
func (tf *TableFormat) Interleave(text string) {
	tf.table = append(tf.table, &InterleavedRow{Text: text})
}

// AppendSuffixToCurrentRow appends suffix text to the current regular row.
func (tf *TableFormat) AppendSuffixToCurrentRow(text string) {
	if len(tf.table) == 0 {
		panic("State error: Not inside a regular table row.")
	}

	lastRow := tf.table[len(tf.table)-1]
	if regularRow, ok := lastRow.(*RegularRow); ok {
		regularRow.Suffix += text
	} else {
		panic("State error: Not inside a regular table row.")
	}
}

// StartNewLine starts a new regular row.
func (tf *TableFormat) StartNewLine() {
	tf.table = append(tf.table, &RegularRow{
		Columns: make([]string, 0),
		Suffix:  "",
	})
}

// Render renders the table to the target renderer.
func (tf *TableFormat) Render(target *Renderer) {
	// First, measure columns
	maxNumberOfColumns := 0
	for _, row := range tf.table {
		if regularRow, ok := row.(*RegularRow); ok {
			if len(regularRow.Columns) > maxNumberOfColumns {
				maxNumberOfColumns = len(regularRow.Columns)
			}
		}
	}

	maxWidthsForEachColumn := make([]int, maxNumberOfColumns)

	// Calculate max widths
	for _, row := range tf.table {
		if regularRow, ok := row.(*RegularRow); ok {
			// Remove trailing empty columns
			for len(regularRow.Columns) > 0 && regularRow.Columns[len(regularRow.Columns)-1] == "" {
				regularRow.Columns = regularRow.Columns[:len(regularRow.Columns)-1]
			}
			for i, col := range regularRow.Columns {
				if i < len(maxWidthsForEachColumn) {
					if len(col) > maxWidthsForEachColumn[i] {
						maxWidthsForEachColumn[i] = len(col)
					}
				}
			}
		}
	}

	// Then, render
	for _, row := range tf.table {
		switch r := row.(type) {
		case *RegularRow:
			for i, col := range r.Columns {
				spacing := 0
				if i < len(r.Columns)-1 {
					// Calculate spacing for alignment
					colWidth := len(col)
					if i < len(maxWidthsForEachColumn) {
						spacing = maxWidthsForEachColumn[i] - colWidth + columnSpacing
					}
				}
				target.builder.WriteString(col)
				if spacing > 0 {
					target.builder.WriteString(strings.Repeat(" ", spacing))
				}
			}
			if r.Suffix != "" {
				if len(r.Columns) > 0 {
					target.builder.WriteString(" ")
				}
				target.builder.WriteString(r.Suffix)
			}
		case *InterleavedRow:
			target.builder.WriteString(r.Text)
		}
		target.builder.WriteString("\n")
	}

	tf.Reset()
}

// Write writes text to the current regular row (implements LineWriteable interface).
func (tf *TableFormat) Write(text string) {
	trimmed := strings.TrimSpace(text)

	if len(tf.table) == 0 {
		tf.StartNewLine()
	}

	lastRow := tf.table[len(tf.table)-1]
	if regularRow, ok := lastRow.(*RegularRow); ok {
		regularRow.Columns = append(regularRow.Columns, trimmed)
	} else {
		panic("State error: Not inside a regular table row.")
	}
}

// EndLine does nothing for table format (implements LineWriteable interface).
func (tf *TableFormat) EndLine() {
	// Table format handles line endings in Render()
}


