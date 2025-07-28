package main

import (
	"log"
	"os"
	"time"

	// These imports were missing, causing all the "undefined" errors.
	"github.com/andrinoff/email-cli/config"
	"github.com/andrinoff/email-cli/fetcher"
	"github.com/andrinoff/email-cli/sender"
	"github.com/andrinoff/email-cli/tui"

	tea "github.com/charmbracelet/bubbletea"
)

type modelState int

const (
	loginView modelState = iota
	mainMenu
	inboxView
	emailView
	composeView
)

// Model holds the application's state.
type Model struct {
	state         modelState
	login         tea.Model
	mainMenu      tea.Model
	inbox         tea.Model
	emailView     tea.Model
	composer      tea.Model
	fetcher       *fetcher.Fetcher
	sender        *sender.Sender
	emails        []fetcher.Email
	statusMessage string
	err           error
}

// New creates the initial model.
func New() Model {
	// We start with the login view.
	return Model{
		state:    loginView,
		login:    tui.NewLogin(),
		mainMenu: tui.NewChoice(),
	}
}

// Init runs any initial commands.
func (m Model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// Update handles all messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Global messages that can be handled regardless of state.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tui.EmailResultMsg:
		if msg.Err != nil {
			m.statusMessage = "❌ Error: " + msg.Err.Error()
		} else {
			m.statusMessage = "✅ Email sent successfully!"
		}
		m.state = mainMenu
		return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return tui.ClearStatusMsg{}
		})
	case tui.ClearStatusMsg:
		m.statusMessage = ""
		return m, nil
	}

	// State-specific message handling.
	switch m.state {
	case loginView:
		// When the user submits their credentials...
		if msg, ok := msg.(tui.Credentials); ok {
			// For now, we'll default to Gmail.
			service := "gmail"
			cfg := &config.Config{Email: msg.Email, Password: msg.Password, Service: service}
			config.Save(cfg)
			m.fetcher = fetcher.New(cfg)
			m.sender = sender.New(msg.Email, msg.Password, service)
			m.state = mainMenu
			return m, nil
		}

	case mainMenu:
		// Handle user's choice from the main menu.
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "1", "j", "down": // View Inbox
				m.state = inboxView
				m.inbox = tui.NewInbox(m.emails)
				return m, func() tea.Msg {
					emails, err := m.fetcher.FetchEmails("INBOX")
					if err != nil {
						return tui.FetchErr(err)
					}
					return tui.EmailsFetchedMsg{Emails: emails}
				}
			case "2", "k", "up": // Compose Email
				m.state = composeView
				m.composer = tui.NewComposer(m.fetcher.Config.Email)
				return m, nil
			}
		}

	case inboxView:
		switch msg := msg.(type) {
		case tui.EmailsFetchedMsg:
			m.emails = msg.Emails
			m.inbox = tui.NewInbox(m.emails)
			return m, nil
		case tui.ViewEmailMsg:
			m.state = emailView
			m.emailView = tui.NewEmailView(m.emails[msg.Index])
			return m, nil
		case tea.KeyMsg:
			if msg.String() == "esc" {
				m.state = mainMenu
			}
		}

	case emailView:
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
			m.state = inboxView // Go back to the inbox list.
		}

	case composeView:
		switch msg := msg.(type) {
		case tui.SendEmailMsg:
			m.statusMessage = "⏳ Sending email..."
			return m, func() tea.Msg {
				err := m.sender.Send(msg.To, msg.Subject, msg.Body)
				return tui.EmailResultMsg{Err: err}
			}
		case tea.KeyMsg:
			if msg.String() == "esc" {
				m.state = mainMenu
			}
		}
	}

	// Pass the message to the current view's Update function.
	switch m.state {
	case loginView:
		m.login, cmd = m.login.Update(msg)
	case mainMenu:
		m.mainMenu, cmd = m.mainMenu.Update(msg)
	case inboxView:
		m.inbox, cmd = m.inbox.Update(msg)
	case emailView:
		m.emailView, cmd = m.emailView.Update(msg)
	case composeView:
		m.composer, cmd = m.composer.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// View renders the current UI.
func (m Model) View() string {
	var s string
	switch m.state {
	case loginView:
		s = m.login.View()
	case mainMenu:
		s = m.mainMenu.View()
	case inboxView:
		s = m.inbox.View()
	case emailView:
		s = m.emailView.View()
	case composeView:
		s = m.composer.View()
	default:
		s = "Unknown state."
	}

	if m.statusMessage != "" {
		return s + "\n\n" + m.statusMessage
	}
	return s
}

func main() {
	// Load config first to see if we can skip login.
	cfg := config.Load()
	m := New()
	if cfg.Email != "" && cfg.Password != "" && cfg.Service != "" {
		m.fetcher = fetcher.New(cfg)
		m.sender = sender.New(cfg.Email, cfg.Password, cfg.Service)
		m.state = mainMenu
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}