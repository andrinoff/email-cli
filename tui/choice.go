package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Choice struct {
	list list.Model
}

func NewChoice() *Choice {
	items := []list.Item{
		choiceItem("View Inbox"),
		choiceItem("Send Email"),
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "What would you like to do?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return &Choice{list: l}
}

type choiceItem string

func (i choiceItem) FilterValue() string { return "" }
func (i choiceItem) Title() string       { return string(i) }
func (i choiceItem) Description() string { return "" }

func (m *Choice) Init() tea.Cmd {
	return nil
}

func (m *Choice) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		if msg.String() == "enter" {
			switch m.list.SelectedItem().(choiceItem) {
			case "View Inbox":
				return m, func() tea.Msg { return GoToInboxMsg{} }
			case "Send Email":
				return m, func() tea.Msg { return GoToSendMsg{} }
			}
		}
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Choice) View() string {
	return DocStyle.Render(m.list.View())
}