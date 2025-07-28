package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type choiceModel struct {
	list   list.Model
	choice string
}

func (m choiceModel) Init() tea.Cmd {
	return nil
}

func (m choiceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i.title)
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := DialogBoxStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m choiceModel) View() string {
	return DialogBoxStyle.Render(m.list.View())
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func RunChoice() (string, error) {
	items := []list.Item{
		item{title: "send", desc: "Send a new email"},
		item{title: "inbox", desc: "View your inbox"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "What would you like to do?"
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	m := choiceModel{list: l}

	p := tea.NewProgram(m, tea.WithAltScreen())

	mod, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
	if m, ok := mod.(choiceModel); ok && m.choice != "" {
		return m.choice, nil
	}
	return "", fmt.Errorf("no choice made")
}
