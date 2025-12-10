package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#00D9FF")
	SuccessColor   = lipgloss.Color("#00FF88")
	WarningColor   = lipgloss.Color("#FFB800")
	ErrorColor     = lipgloss.Color("#FF4444")
	InfoColor      = lipgloss.Color("#00D9FF")
	SecondaryColor = lipgloss.Color("#6C757D")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(InfoColor)

	SecondaryStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor)
)

// PrintHeader prints a beautiful header
func PrintHeader(title string, subtitle string) {
	width := 80
	if w := pterm.GetTerminalWidth(); w > 0 {
		width = w
	}

	header := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 2).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				TitleStyle.Render(title),
				SecondaryStyle.Render(subtitle),
			),
		)

	fmt.Println(header)
	fmt.Println()
}

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Println(SuccessStyle.Render("✓ " + message))
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, ErrorStyle.Render("✗ "+message))
}

// PrintWarning prints a warning message
func PrintWarning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Println(WarningStyle.Render("⚠ " + message))
}

// PrintInfo prints an info message
func PrintInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Println(InfoStyle.Render("ℹ " + message))
}

// PrintStep prints a step indicator
func PrintStep(step int, total int, message string) {
	stepStyle := lipgloss.NewStyle().
		Foreground(SecondaryColor).
		Render(fmt.Sprintf("[%d/%d]", step, total))

	fmt.Printf("%s %s\n", stepStyle, message)
}

// PrintTable prints a table using pterm
func PrintTable(headers []string, rows [][]string) {
	tableData := pterm.TableData{headers}
	tableData = append(tableData, rows...)
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// PrintList prints a bulleted list
func PrintList(items []string) {
	for _, item := range items {
		fmt.Printf("  • %s\n", item)
	}
}

// PrintMarkdown renders markdown content
func PrintMarkdown(content string) error {
	// Use glamour for markdown rendering
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return err
	}

	out, err := r.Render(content)
	if err != nil {
		return err
	}

	fmt.Print(out)
	return nil
}

// PrintBox prints content in a box
func PrintBox(title string, content string) {
	width := 80
	if w := pterm.GetTerminalWidth(); w > 0 {
		width = w
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 2).
		Width(width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				TitleStyle.Render(title),
				content,
			),
		)

	fmt.Println(box)
}

// PrintProgressBar creates and updates a progress bar
func PrintProgressBar(total int) *pterm.ProgressbarPrinter {
	return pterm.DefaultProgressbar.WithTotal(total)
}

// PrintSpinner creates a spinner and returns it
func PrintSpinner(message string) (*pterm.SpinnerPrinter, error) {
	spinner := pterm.DefaultSpinner.WithText(message)
	spinner.Start()
	return spinner, nil
}

// PrintSection prints a section header
func PrintSection(title string) {
	width := 80
	if w := pterm.GetTerminalWidth(); w > 0 {
		width = w
	}

	section := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(SecondaryColor).
		Padding(0, 0, 1, 0).
		Render(title)

	fmt.Println(section)
}

// PrintCodeBlock prints code in a styled block
func PrintCodeBlock(code string, language string) {
	codeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(SecondaryColor).
		Padding(1).
		Background(lipgloss.Color("#1E1E1E")).
		Foreground(lipgloss.Color("#D4D4D4")).
		Width(80)

	header := ""
	if language != "" {
		header = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Render(fmt.Sprintf(" %s ", language))
	}

	content := codeStyle.Render(code)
	if header != "" {
		fmt.Println(header)
	}
	fmt.Println(content)
}

// PrintDiff prints a diff-like output
func PrintDiff(old string, new string) {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	for i := 0; i < len(oldLines) || i < len(newLines); i++ {
		if i < len(oldLines) && i < len(newLines) {
			if oldLines[i] != newLines[i] {
				fmt.Println(ErrorStyle.Render("- " + oldLines[i]))
				fmt.Println(SuccessStyle.Render("+ " + newLines[i]))
			} else {
				fmt.Println("  " + oldLines[i])
			}
		} else if i < len(oldLines) {
			fmt.Println(ErrorStyle.Render("- " + oldLines[i]))
		} else {
			fmt.Println(SuccessStyle.Render("+ " + newLines[i]))
		}
	}
}

// ColorPrint uses fatih/color for simple colored output
func ColorPrint(c *color.Color, format string, args ...interface{}) {
	c.Printf(format, args...)
}

// GetColorPrinters returns color printers for common use cases
func GetColorPrinters() map[string]*color.Color {
	return map[string]*color.Color{
		"success": color.New(color.FgGreen, color.Bold),
		"error":   color.New(color.FgRed, color.Bold),
		"warning": color.New(color.FgYellow, color.Bold),
		"info":    color.New(color.FgCyan),
		"primary": color.New(color.FgCyan, color.Bold),
	}
}

