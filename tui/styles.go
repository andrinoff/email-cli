package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	DialogBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 0).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	HelpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)

var DocStyle = lipgloss.NewStyle().Margin(1, 2)

// A simple model for showing a status message
type Status struct {
    spinner spinner.Model
    message string
}

func NewStatus(msg string) Status {
    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
    return Status{spinner: s, message: msg}
}

func (m Status) Init() tea.Cmd { return m.spinner.Tick }

func (m Status) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Status) View() string {
	return fmt.Sprintf("\n\n   %s %s\n\n", m.spinner.View(), m.message)
}