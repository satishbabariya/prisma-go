// Package diagnostics provides colored pretty printing for errors and warnings.
package diagnostics

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

// DiagnosticColorer defines the interface for coloring diagnostic output.
type DiagnosticColorer interface {
	Title() string
	PrimaryColor(text string) string
}

// ErrorColorer provides coloring for error diagnostics.
type ErrorColorer struct{}

// Title returns the title for errors.
func (e ErrorColorer) Title() string {
	return "error"
}

// PrimaryColor returns the colored text for errors.
func (e ErrorColorer) PrimaryColor(text string) string {
	return color.New(color.FgRed, color.Bold).Sprint(text)
}

// WarningColorer provides coloring for warning diagnostics.
type WarningColorer struct{}

// Title returns the title for warnings.
func (w WarningColorer) Title() string {
	return "warning"
}

// PrimaryColor returns the colored text for warnings.
func (w WarningColorer) PrimaryColor(text string) string {
	return color.New(color.FgYellow, color.Bold).Sprint(text)
}

// PrettyPrint pretty prints an error or warning, including the offending portion
// of the source code, for human-friendly reading.
func PrettyPrint(
	w io.Writer,
	fileName string,
	text string,
	span Span,
	description string,
	colorer DiagnosticColorer,
) error {
	// Disable colors if NO_COLOR environment variable is set
	if os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
	}

	startLineNumber := strings.Count(text[:span.Start], "\n")
	endLineNumber := strings.Count(text[:span.End], "\n")
	fileLines := strings.Split(text, "\n")

	// Calculate bytes before the start line
	bytesInLineBefore := 0
	for i := 0; i < startLineNumber; i++ {
		bytesInLineBefore += len(fileLines[i]) + 1 // +1 for newline
	}

	line := fileLines[startLineNumber]
	startInLine := span.Start - bytesInLineBefore
	endInLine := startInLine + (span.End - span.Start)
	if endInLine > len(line) {
		endInLine = len(line)
	}

	prefix := line[:startInLine]
	offending := line[startInLine:endInLine]
	suffix := line[endInLine:]

	// Color functions
	titleColor := color.New(color.Bold)
	arrowColor := color.New(color.FgCyan, color.Bold)
	filePathColor := color.New(color.Underline)
	lineNumColor := color.New(color.FgCyan, color.Bold)

	// Title and description
	titleColor.Fprintf(w, "%s: ", colorer.Title())
	titleColor.Fprintf(w, "%s\n", description)

	// Arrow and file path
	arrowColor.Fprintf(w, "  --> ")
	filePathColor.Fprintf(w, "%s:%d\n", fileName, startLineNumber+1)

	// Empty line number
	lineNumColor.Fprintf(w, "   | \n")

	// Previous line with content (if available)
	if startLineNumber > 0 && startLineNumber <= len(fileLines) {
		lineNumColor.Fprintf(w, "%2d | ", startLineNumber)
		fmt.Fprintf(w, "%s\n", fileLines[startLineNumber-1])
	}

	// Line with offending content
	lineNumColor.Fprintf(w, "%2d | ", startLineNumber+1)
	fmt.Fprintf(w, "%s", prefix)
	fmt.Fprintf(w, "%s", colorer.PrimaryColor(offending))
	fmt.Fprintf(w, "%s\n", suffix)

	// Pointer line for empty offending text
	if len(offending) == 0 {
		lineNumColor.Fprintf(w, "   | ")
		fmt.Fprintf(w, "%s", strings.Repeat(" ", startInLine))
		fmt.Fprintf(w, "%s\n", colorer.PrimaryColor("^ Unexpected token."))
	}

	// Additional lines if span spans multiple lines
	for lineNumber := startLineNumber + 2; lineNumber <= endLineNumber+2 && lineNumber <= len(fileLines); lineNumber++ {
		if lineNumber-1 < len(fileLines) {
			lineNumColor.Fprintf(w, "%2d | ", lineNumber)
			fmt.Fprintf(w, "%s\n", fileLines[lineNumber-1])
		}
	}

	// Empty line number at end
	lineNumColor.Fprintf(w, "   | \n")

	return nil
}

