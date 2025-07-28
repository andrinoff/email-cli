package tui

import (
	"fmt"

	"github.com/andrinoff/email-cli/config"
	"github.com/andrinoff/email-cli/fetcher"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// inboxItem wraps fetcher.Email to satisfy the list.Item interface.
type inboxItem struct {
	fetcher.Email
}

func (i inboxItem) Title() string       { return i.Subject }
func (i inboxItem) Description() string { return fmt.Sprintf("From: %s", i.From) }
func (i inboxItem) FilterValue() string { return i.Subject }

type Inbox struct {
	list    list.Model
	spinner spinner.Model
	loading bool
	err     error
}

func NewInbox() *Inbox {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Create an empty list for now. It will be populated later.
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Inbox"

	return &Inbox{
		spinner: s,
		list:    l,
		loading: true,
	}
}

func (m *Inbox) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchEmails)
}

func (m *Inbox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		return m, cmd

	case emailsFetchedMsg:
		m.loading = false
		items := make([]list.Item, len(msg))
		for i, email := range msg {
			items[i] = inboxItem{email}
		}
		m.list.SetItems(items)
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "enter" && !m.loading {
			selected := m.list.SelectedItem().(inboxItem)
			return m, func() tea.Msg {
				return ViewEmailMsg{Email: selected.Email}
			}
		}

	case tea.WindowSizeMsg:
		// When the window size changes, update the list's dimensions.
		m.list.SetSize(msg.Width, msg.Height)
	}

	if !m.loading {
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m *Inbox) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	if m.loading {
		return fmt.Sprintf("\n\n   %s Fetching emails...\n\n", m.spinner.View())
	}
	return DocStyle.Render(m.list.View())
}

// --- Bubble Tea Commands and Messages ---

type emailsFetchedMsg []fetcher.Email
type errMsg struct{ err error }

func fetchEmails() tea.Msg {
	cfg, err := config.LoadConfig()
	if err != nil {
		return errMsg{err}
	}
	emails, err := fetcher.FetchEmails(cfg)
	if err != nil {
		return errMsg{err}
	}
	return emailsFetchedMsg(emails)
}