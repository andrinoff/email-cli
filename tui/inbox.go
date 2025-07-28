

package tui

import (
	"fmt"
	"strings"

	"github.com/andrinoff/email-cli/config"
	"github.com/andrinoff/email-cli/fetcher"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type inboxModel struct {
	spinner  spinner.Model
	loading  bool
	emails   []fetcher.Email
	list     list.Model
	err      error
	config   *config.Config
	selected bool
}

func (m inboxModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m inboxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			m.selected = true
			return m, nil
		}
	case tea.WindowSizeMsg:
		h, v := DialogBoxStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case []fetcher.Email:
		m.loading = false
		m.emails = msg
		items := make([]list.Item, len(m.emails))
		for i, email := range m.emails {
			items[i] = item{
				title: email.Subject,
				desc:  fmt.Sprintf("From: %s", email.From),
			}
		}
		m.list.SetItems(items)

	case error:
		m.err = msg
		return m, nil
	}

	var cmd tea.Cmd
	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
	} else {
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m inboxModel) View() string {
	if m.err != nil {
		return InfoStyle.Render(fmt.Sprintf("\n   Error: %v \n\n Press any key to exit.", m.err))
	}
	if m.loading {
		return InfoStyle.Render(fmt.Sprintf("\n   %s Fetching emails... \n\n", m.spinner.View()))
	}
	if m.selected {
		i, ok := m.list.SelectedItem().(item)
		if !ok {
			return "error"
		}
		var selectedEmail fetcher.Email
		for _, email := range m.emails {
			if email.Subject == i.Title() {
				selectedEmail = email
				break
			}
		}
		var s strings.Builder
		s.WriteString(fmt.Sprintf("From: %s\n", selectedEmail.From))
		s.WriteString(fmt.Sprintf("Subject: %s\n", selectedEmail.Subject))
		s.WriteString(fmt.Sprintf("Date: %s\n\n", selectedEmail.Date.Format("2006-01-02 15:04:05")))
		s.WriteString(selectedEmail.Body)
		s.WriteString(HelpStyle.Render("\n\n(q to quit)"))
		return DialogBoxStyle.Render(s.String())

	}
	return DialogBoxStyle.Render(m.list.View())
}

func initialInboxModel(cfg *config.Config) inboxModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Inbox"
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	return inboxModel{
		spinner: s,
		loading: true,
		config:  cfg,
		list:    l,
	}
}

func RunInbox(cfg *config.Config) error {
	m := initialInboxModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		emails, err := fetcher.FetchEmails(cfg)
		if err != nil {
			p.Send(err)
		}
		p.Send(emails)
	}()

	_, err := p.Run()
	return err
}