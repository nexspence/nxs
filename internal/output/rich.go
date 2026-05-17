package output

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type RichPrinter struct{}

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func (p *RichPrinter) Table(headers []string, rows [][]string) {
	t := table.New().
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}
			return lipgloss.NewStyle()
		})
	fmt.Println(t)
}

func (p *RichPrinter) Success(msg string) {
	fmt.Println(successStyle.Render("✓ " + msg))
}

func (p *RichPrinter) Error(msg string) {
	fmt.Fprintln(os.Stderr, errorStyle.Render("✗ "+msg))
}

func (p *RichPrinter) JSON(v any) {
	(&JSONPrinter{}).JSON(v)
}
