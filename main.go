package main

import (
	"fmt"
	"os"

	"github.com/andrinoff/email-cli/config"
	"github.com/andrinoff/email-cli/fetcher"
	"github.com/andrinoff/email-cli/view" // Updated package
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport" // Import viewport
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)
	headerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true)
)

// emailItem wraps fetcher.Email to satisfy the list.Item interface.
type emailItem struct {
	fetcher.Email
}

func (e emailItem) Title() string       { return e.Subject }
func (e emailItem) Description() string { return fmt.Sprintf("From: %s", e.From) }
func (e emailItem) FilterValue() string { return e.Subject }

type model struct {
	list         list.Model
	spinner      spinner.Model
	viewport     viewport.Model
	state        viewState
	selectedItem emailItem
	err          error
}

type viewState int

const (
	choosingState viewState = iota
	loadingState
	inboxState
	emailViewState
)

// Messages for tea.Cmd
type (
	emailsFetchedMsg []fetcher.Email
	errMsg           struct{ err error }
)

func (e errMsg) Error() string { return e.err.Error() }

func newModel() *model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	initialItems := []list.Item{
		emailItem{fetcher.Email{Subject: "View Inbox"}},
		emailItem{fetcher.Email{Subject: "Send Email (coming soon...)"}},
	}

	m := &model{
		spinner:  s,
		list:     list.New(initialItems, list.NewDefaultDelegate(), 0, 0),
		state:    choosingState,
		viewport: viewport.New(10, 10), // Initial size
	}
	m.list.Title = "Email CLI"
	m.list.SetShowStatusBar(false)
	return m
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		listHeight := msg.Height - 2
		// Adjust viewport size, leaving space for a header
		viewportHeaderHeight := 4
		m.list.SetSize(msg.Width-2, listHeight)
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - viewportHeaderHeight

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state == emailViewState {
				m.state = inboxState
			} else if m.state == inboxState {
				// Reset to initial choice list
				initialItems := []list.Item{
					emailItem{fetcher.Email{Subject: "View Inbox"}},
					emailItem{fetcher.Email{Subject: "Send Email (coming soon...)"}},
				}
				m.list.SetItems(initialItems)
				m.list.Title = "Email CLI"
				m.state = choosingState
			}
			return m, nil
		case "enter":
			if m.state == choosingState {
				selected := m.list.SelectedItem().(emailItem)
				if selected.Subject == "View Inbox" {
					m.state = loadingState
					cmds = append(cmds, m.spinner.Tick, fetchEmails)
				}
			} else if m.state == inboxState {
				m.selectedItem = m.list.SelectedItem().(emailItem)
				m.state = emailViewState

				// Process body and set viewport content
				body, err := view.ProcessBody(m.selectedItem.Body)
				if err != nil {
					body = fmt.Sprintf("Error rendering body: %v", err)
				}
				m.viewport.SetContent(body)
				m.viewport.GotoTop() // Scroll to top on new email
			}
		}

	case emailsFetchedMsg:
		items := make([]list.Item, len(msg))
		for i, email := range msg {
			items[i] = emailItem{fetcher.Email(email)}
		}
		m.list.SetItems(items)
		m.list.Title = "Inbox"
		m.state = inboxState

	case errMsg:
		m.err = msg
		return m, tea.Quit
	}

	// Handle updates for the current state
	switch m.state {
	case loadingState:
		m.spinner, cmd = m.spinner.Update(msg)
	case inboxState, choosingState:
		m.list, cmd = m.list.Update(msg)
	case emailViewState:
		m.viewport, cmd = m.viewport.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	switch m.state {
	case loadingState:
		return fmt.Sprintf("\n\n   %s Fetching emails...\n\n", m.spinner.View())
	case emailViewState:
		header := fmt.Sprintf("From: %s\nSubject: %s", m.selectedItem.From, m.selectedItem.Subject)
		styledHeader := headerStyle.Width(m.viewport.Width).Render(header)
		return fmt.Sprintf("%s\n%s", styledHeader, m.viewport.View())
	default: // choosingState and inboxState
		return appStyle.Render(m.list.View())
	}
}

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

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}