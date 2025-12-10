package diagnostics

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Diagnostics represents a list of validation or parser errors and warnings.
// This is used to accumulate multiple errors and warnings during validation.
// It is used to not error out early and instead show multiple errors at once.
type Diagnostics struct {
	errors   []DatamodelError
	warnings []DatamodelWarning
}

// NewDiagnostics creates a new empty Diagnostics collection.
func NewDiagnostics() Diagnostics {
	return Diagnostics{
		errors:   make([]DatamodelError, 0),
		warnings: make([]DatamodelWarning, 0),
	}
}

// Warnings returns all warnings in the collection.
func (d *Diagnostics) Warnings() []DatamodelWarning {
	return d.warnings
}

// IntoWarnings consumes the diagnostics and returns all warnings.
func (d *Diagnostics) IntoWarnings() []DatamodelWarning {
	warnings := make([]DatamodelWarning, len(d.warnings))
	copy(warnings, d.warnings)
	return warnings
}

// Errors returns all errors in the collection.
func (d *Diagnostics) Errors() []DatamodelError {
	return d.errors
}

// PushError adds an error to the collection.
func (d *Diagnostics) PushError(err DatamodelError) {
	d.errors = append(d.errors, err)
}

// PushWarning adds a warning to the collection.
func (d *Diagnostics) PushWarning(warning DatamodelWarning) {
	d.warnings = append(d.warnings, warning)
}

// HasErrors returns true if there is at least one error in this collection.
func (d *Diagnostics) HasErrors() bool {
	return len(d.errors) > 0
}

// ToResult returns an error if there are errors, otherwise returns nil.
func (d *Diagnostics) ToResult() error {
	if d.HasErrors() {
		return fmt.Errorf("validation failed with %d errors", len(d.errors))
	}
	return nil
}

// ToPrettyString formats all errors as a pretty-printed string.
func (d *Diagnostics) ToPrettyString(fileName, datamodelString string) string {
	var buf bytes.Buffer
	for _, err := range d.errors {
		d.writePrettyError(&buf, fileName, datamodelString, err)
	}
	return buf.String()
}

// WarningsToPrettyString formats all warnings as a pretty-printed string.
func (d *Diagnostics) WarningsToPrettyString(fileName, datamodelString string) string {
	var buf bytes.Buffer
	for _, warn := range d.warnings {
		d.writePrettyWarning(&buf, fileName, datamodelString, warn)
	}
	return buf.String()
}

// writePrettyError writes a pretty-printed error to the buffer with colors.
func (d *Diagnostics) writePrettyError(buf *bytes.Buffer, fileName, text string, err DatamodelError) {
	startLineNum := d.getLineNumber(text, err.Span().Start)
	endLineNum := d.getLineNumber(text, err.Span().End)
	lines := d.getLines(text)

	bytesInLineBefore := d.getLineStart(text, startLineNum)
	line := lines[startLineNum]
	startInLine := err.Span().Start - bytesInLineBefore
	endInLine := startInLine + (err.Span().End - err.Span().Start)
	if endInLine > len(line) {
		endInLine = len(line)
	}

	prefix := line[:startInLine]
	offending := line[startInLine:endInLine]
	suffix := line[endInLine:]

	// Color functions
	errorTitle := color.New(color.FgRed, color.Bold)
	errorDesc := color.New(color.Bold)
	arrowColor := color.New(color.FgCyan, color.Bold)
	filePathColor := color.New(color.Underline)
	lineNumColor := color.New(color.FgCyan, color.Bold)
	offendingColor := color.New(color.FgRed, color.Bold)

	// Title and description
	errorTitle.Fprintf(buf, "error")
	fmt.Fprintf(buf, ": ")
	errorDesc.Fprintf(buf, "%s\n", err.Message())

	// Arrow and file path
	arrowColor.Fprintf(buf, "  --> ")
	filePathColor.Fprintf(buf, "%s:%d\n", fileName, startLineNum+1)

	// Empty line number
	lineNumColor.Fprintf(buf, "   | \n")

	// Line with content (if available)
	if startLineNum < len(lines) {
		lineNumColor.Fprintf(buf, "%2d | ", startLineNum)
		fmt.Fprintf(buf, "%s", prefix)
		offendingColor.Fprintf(buf, "%s", offending)
		fmt.Fprintf(buf, "%s\n", suffix)
	}

	// Pointer line
	if len(offending) == 0 {
		lineNumColor.Fprintf(buf, "   | ")
		fmt.Fprintf(buf, "%s", strings.Repeat(" ", startInLine))
		offendingColor.Fprintf(buf, "^ Unexpected token.\n")
	} else {
		lineNumColor.Fprintf(buf, "   | ")
		fmt.Fprintf(buf, "%s", strings.Repeat(" ", startInLine))
		offendingColor.Fprintf(buf, "%s\n", strings.Repeat("^", len(offending)))
	}

	// Additional lines if span spans multiple lines
	for lineNum := startLineNum + 1; lineNum <= endLineNum && lineNum < len(lines); lineNum++ {
		lineNumColor.Fprintf(buf, "%2d | ", lineNum+1)
		fmt.Fprintf(buf, "%s\n", lines[lineNum])
	}

	// Empty line number at end
	lineNumColor.Fprintf(buf, "   | \n")
}

// writePrettyWarning writes a pretty-printed warning to the buffer with colors.
func (d *Diagnostics) writePrettyWarning(buf *bytes.Buffer, fileName, text string, warn DatamodelWarning) {
	startLineNum := d.getLineNumber(text, warn.Span().Start)
	endLineNum := d.getLineNumber(text, warn.Span().End)
	lines := d.getLines(text)

	bytesInLineBefore := d.getLineStart(text, startLineNum)
	line := lines[startLineNum]
	startInLine := warn.Span().Start - bytesInLineBefore
	endInLine := startInLine + (warn.Span().End - warn.Span().Start)
	if endInLine > len(line) {
		endInLine = len(line)
	}

	prefix := line[:startInLine]
	offending := line[startInLine:endInLine]
	suffix := line[endInLine:]

	// Color functions
	warningTitle := color.New(color.FgYellow, color.Bold)
	warningDesc := color.New(color.Bold)
	arrowColor := color.New(color.FgCyan, color.Bold)
	filePathColor := color.New(color.Underline)
	lineNumColor := color.New(color.FgCyan, color.Bold)
	offendingColor := color.New(color.FgYellow, color.Bold)

	// Title and description
	warningTitle.Fprintf(buf, "warning")
	fmt.Fprintf(buf, ": ")
	warningDesc.Fprintf(buf, "%s\n", warn.Message())

	// Arrow and file path
	arrowColor.Fprintf(buf, "  --> ")
	filePathColor.Fprintf(buf, "%s:%d\n", fileName, startLineNum+1)

	// Empty line number
	lineNumColor.Fprintf(buf, "   | \n")

	// Line with content
	if startLineNum < len(lines) {
		lineNumColor.Fprintf(buf, "%2d | ", startLineNum+1)
		fmt.Fprintf(buf, "%s", prefix)
		offendingColor.Fprintf(buf, "%s", offending)
		fmt.Fprintf(buf, "%s\n", suffix)
	}

	// Pointer line
	if len(offending) > 0 {
		lineNumColor.Fprintf(buf, "   | ")
		fmt.Fprintf(buf, "%s", strings.Repeat(" ", startInLine))
		offendingColor.Fprintf(buf, "%s\n", strings.Repeat("^", len(offending)))
	}

	// Additional lines if span spans multiple lines
	for lineNum := startLineNum + 1; lineNum <= endLineNum && lineNum < len(lines); lineNum++ {
		lineNumColor.Fprintf(buf, "%2d | ", lineNum+1)
		fmt.Fprintf(buf, "%s\n", lines[lineNum])
	}

	// Empty line number at end
	lineNumColor.Fprintf(buf, "   | \n")
}

// getLineNumber returns the line number (0-based) for a given position.
func (d *Diagnostics) getLineNumber(text string, pos int) int {
	return strings.Count(text[:pos], "\n")
}

// getLineStart returns the start position of a line.
func (d *Diagnostics) getLineStart(text string, lineNum int) int {
	pos := 0
	for i := 0; i < lineNum; i++ {
		if idx := strings.Index(text[pos:], "\n"); idx >= 0 {
			pos += idx + 1
		} else {
			break
		}
	}
	return pos
}

// getLines splits text into lines.
func (d *Diagnostics) getLines(text string) []string {
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// FromError creates a Diagnostics from a single error.
func FromError(err DatamodelError) Diagnostics {
	d := NewDiagnostics()
	d.PushError(err)
	return d
}

// FromWarning creates a Diagnostics from a single warning.
func FromWarning(warning DatamodelWarning) Diagnostics {
	d := NewDiagnostics()
	d.PushWarning(warning)
	return d
}
