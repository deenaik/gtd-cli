package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	DimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	BoldStyle   = lipgloss.NewStyle().Bold(true)
	WarnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	ErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	AccentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
)

type Table struct {
	Headers []string
	Rows    [][]string
	Out     io.Writer
}

func NewTable(headers ...string) *Table {
	return &Table{
		Headers: headers,
		Out:     os.Stdout,
	}
}

func (t *Table) AddRow(cols ...string) {
	t.Rows = append(t.Rows, cols)
}

func (t *Table) Render() {
	if len(t.Headers) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = len(h)
	}
	for _, row := range t.Rows {
		for i, col := range row {
			if i < len(widths) && len(col) > widths[i] {
				widths[i] = len(col)
			}
		}
	}

	// Print header
	headerParts := make([]string, len(t.Headers))
	for i, h := range t.Headers {
		headerParts[i] = HeaderStyle.Render(padRight(h, widths[i]))
	}
	fmt.Fprintln(t.Out, strings.Join(headerParts, "  "))

	// Separator
	sepParts := make([]string, len(t.Headers))
	for i := range t.Headers {
		sepParts[i] = DimStyle.Render(strings.Repeat("─", widths[i]))
	}
	fmt.Fprintln(t.Out, strings.Join(sepParts, "  "))

	// Rows
	for _, row := range t.Rows {
		parts := make([]string, len(t.Headers))
		for i := range t.Headers {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			parts[i] = padRight(val, widths[i])
		}
		fmt.Fprintln(t.Out, strings.Join(parts, "  "))
	}
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func PrintSuccess(msg string) {
	fmt.Println(SuccessStyle.Render("✓ " + msg))
}

func PrintError(msg string) {
	fmt.Fprintln(os.Stderr, ErrorStyle.Render("✗ " + msg))
}

func PrintWarn(msg string) {
	fmt.Println(WarnStyle.Render("! " + msg))
}

func PrintInfo(msg string) {
	fmt.Println(AccentStyle.Render("• " + msg))
}

func PrintSection(title string) {
	fmt.Println()
	fmt.Println(BoldStyle.Render("── " + title + " ──"))
}
