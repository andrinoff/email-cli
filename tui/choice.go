package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/floatpane/matcha/config"
)

// Styles defined locally to avoid import issues.
var (
	docStyle          = lipgloss.NewStyle().Margin(1, 2)
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Background(lipgloss.Color("#25A065")).Padding(0, 1)
	logoStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	listHeader        = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingBottom(1)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("42"))
)

// ASCII logo for the start screen
const choiceLogo = `
                    __       __
   ____ ___  ____ _/ /______/ /_  ____ _
  / __ '__ \/ __ '/ __/ ___/ __ \/ __ '/
 / / / / / / /_/ / /_/ /__/ / / / /_/ /
/_/ /_/ /_/\__,_/\__/\___/_/ /_/\__,_/
`

type Choice struct {
	cursor         int
	choices        []string
	hasSavedDrafts bool
}

func NewChoice() Choice {
	hasSavedDrafts := config.HasDrafts()
	choices := []string{"View Inbox", "Compose Email"}
	if hasSavedDrafts {
		choices = append(choices, "Drafts")
	}
	choices = append(choices, "Settings")
	return Choice{
		choices:        choices,
		hasSavedDrafts: hasSavedDrafts,
	}
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
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			selectedChoice := m.choices[m.cursor]
			switch selectedChoice {
			case "View Inbox":
				return m, func() tea.Msg { return GoToInboxMsg{} }
			case "Compose Email":
				return m, func() tea.Msg { return GoToSendMsg{} }
			case "Drafts":
				return m, func() tea.Msg { return GoToDraftsMsg{} }
			case "Settings":
				return m, func() tea.Msg { return GoToSettingsMsg{} }
			}
		}
	}
	return m, nil
}

func (m Choice) View() string {
	var b strings.Builder

	b.WriteString(logoStyle.Render(choiceLogo))
	b.WriteString("\n")
	b.WriteString(listHeader.Render("What would you like to do?"))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		if m.cursor == i {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("> %s", choice)))
		} else {
			b.WriteString(itemStyle.Render(fmt.Sprintf("  %s", choice)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Use ↑/↓ to navigate, enter to select, and ctrl+c to quit."))

	return docStyle.Render(b.String())
}
