package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/andrinoff/email-cli/config"
	"github.com/andrinoff/email-cli/fetcher"
	"github.com/andrinoff/email-cli/sender"
	"github.com/andrinoff/email-cli/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// mainModel now holds the state for the entire application.
type mainModel struct {
	current tea.Model
	config  *config.Config
	emails  []fetcher.Email
	width   int
	height  int
	err     error
}

// newInitialModel now returns a pointer, which is crucial for state management.
func newInitialModel(cfg *config.Config) *mainModel {
	return &mainModel{
		current: tui.NewChoice(),
		config:  cfg,
	}
}

func (m *mainModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m *mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Pass the window size message to the current view
		m.current, cmd = m.current.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// Allow ESC to go back to the main menu
		if msg.String() == "esc" {
			if _, ok := m.current.(*tui.Choice); !ok {
				m.current = tui.NewChoice()
				return m, m.current.Init()
			}
		}

	// --- Custom Messages for switching views ---
	case tui.GoToInboxMsg:
		m.current = tui.NewStatus("Fetching emails...")
		return m, tea.Batch(m.current.Init(), fetchEmails(m.config))

	case tui.EmailsFetchedMsg:
		m.emails = msg.Emails
		m.current = tui.NewInbox(m.emails)
		// Manually set the size of the new view
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, m.current.Init())

	case tui.GoToSendMsg:
		m.current = tui.NewComposer(m.config.Email)
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, m.current.Init())

	case tui.ViewEmailMsg:
		m.current = tui.NewEmailView(m.emails[msg.Index], m.width, m.height)
		cmds = append(cmds, m.current.Init())

	case tui.SendEmailMsg:
		m.current = tui.NewStatus("Sending email...")
		cmds = append(cmds, m.current.Init(), sendEmail(m.config, msg))

	case tui.EmailResultMsg:
		m.current = tui.NewChoice()
		cmds = append(cmds, m.current.Init())
	}

	// Pass all other messages to the current view
	m.current, cmd = m.current.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *mainModel) View() string {
	return m.current.View()
}

// sendEmail is a command that sends an email in the background.
func sendEmail(cfg *config.Config, msg tui.SendEmailMsg) tea.Cmd {
	return func() tea.Msg {
		recipients := []string{msg.To}
		err := sender.SendEmail(cfg, recipients, msg.Subject, msg.Body)
		if err != nil {
			log.Printf("Failed to send email: %v", err) // Log error
			return tui.EmailResultMsg{Err: err}
		}
		time.Sleep(1 * time.Second) // Give user time to see the "Sending" message
		return tui.EmailResultMsg{}
	}
}

func fetchEmails(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		emails, err := fetcher.FetchEmails(cfg)
		if err != nil {
			return tui.FetchErr(err)
		}
		return tui.EmailsFetchedMsg{Emails: emails}
	}
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		// If the config doesn't exist, we can guide the user to create one.
		// For now, we'll just log a fatal error.
		log.Fatalf("could not load config: %v", err)
	}

	p := tea.NewProgram(newInitialModel(cfg), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}