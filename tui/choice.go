package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2)

	choiceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			PaddingLeft(2)
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
			if m.cursor < 1 {
				m.cursor++
			}
		case "enter":
			if m.cursor == 0 {
				return m, func() tea.Msg { return GoToInboxMsg{} }
			} else if m.cursor == 1 {
				return m, func() tea.Msg { return GoToSendMsg{} }
			}
		}
	}
	return m, nil
}

func (m Choice) View() string {
	s := "What would you like to do?\n\n"

	choices := []string{"View Inbox", "Compose Email"}

	for i, choice := range choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			s += choiceStyle.Render(fmt.Sprintf("%s %s\n", cursor, choice))
		} else {
			s += fmt.Sprintf("%s %s\n", cursor, choice)
		}
	}

	s += "\nPress q to quit.\n"

	return s
}