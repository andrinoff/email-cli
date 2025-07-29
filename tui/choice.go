package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// A more beautiful style for the main menu
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// Styling for the choice list
	listHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingBottom(1)

	// Custom item styles
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("205"))
)

type Choice struct {
	cursor int
}

func NewChoice() Choice {
	return Choice{cursor: 0}
}

func (m Choice) Init() tea.Cmd {
	return nil
}

func (m Choice) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < 2 { // We have three choices now
				m.cursor++
			}
		case "enter":
			switch m.cursor {
			case 0:
				return m, func() tea.Msg { return GoToInboxMsg{} }
			case 1:
				return m, func() tea.Msg { return GoToSendMsg{} }
			case 2:
				return m, func() tea.Msg { return GoToSettingsMsg{} }
			}
		}
	}
	return m, nil
}

func (m Choice) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Email CLI") + "\n\n")

	// Header
	b.WriteString(listHeader.Render("What would you like to do?"))
	b.WriteString("\n\n")

	// Choices
	choices := []string{"View Inbox", "Compose Email", "Settings"}
	for i, choice := range choices {
		if m.cursor == i {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("> %s", choice)))
		} else {
			b.WriteString(itemStyle.Render(fmt.Sprintf("  %s", choice)))
		}
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Use ↑/↓ to navigate, enter to select, and q to quit."))

	return docStyle.Render(b.String())
}