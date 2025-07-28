package tui

import "github.com/charmbracelet/lipgloss"

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