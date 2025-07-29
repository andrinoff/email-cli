package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/andrinoff/email-cli/config"
	"github.com/andrinoff/email-cli/fetcher"
	"github.com/andrinoff/email-cli/sender"
	"github.com/andrinoff/email-cli/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

// mainModel holds the state for the entire application.
type mainModel struct {
	current tea.Model
	config  *config.Config
	emails  []fetcher.Email
	inbox   *tui.Inbox
	width   int
	height  int
	err     error
}

// newInitialModel returns a pointer to the initial model.
func newInitialModel(cfg *config.Config) *mainModel {
	// If config is nil, start with the login screen.
	if cfg == nil {
		return &mainModel{
			current: tui.NewLogin(),
		}
	}
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
		// Pass the window size message to the current view.
		m.current, cmd = m.current.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// Allow ESC to navigate back.
		if msg.String() == "esc" {
			switch m.current.(type) {
			case *tui.EmailView:
				m.current = m.inbox // Go back to the cached inbox.
				return m, nil
			case *tui.Inbox, *tui.Composer, *tui.Login:
				m.current = tui.NewChoice() // Go back to the main menu.
				return m, m.current.Init()
			}
		}

	// --- Custom Messages for Switching Views ---
	case tui.Credentials:
		cfg := &config.Config{
			ServiceProvider: msg.Provider,
			Name:            msg.Name,
			Email:           msg.Email,
			Password:        msg.Password,
		}
		if err := config.SaveConfig(cfg); err != nil {
			log.Printf("could not save config: %v", err)
			return m, tea.Quit
		}
		m.config = cfg
		m.current = tui.NewChoice()
		cmds = append(cmds, m.current.Init())

	case tui.GoToInboxMsg:
		m.current = tui.NewStatus("Fetching emails...")
		return m, tea.Batch(m.current.Init(), fetchEmails(m.config))

	case tui.EmailsFetchedMsg:
		m.emails = msg.Emails
		m.inbox = tui.NewInbox(m.emails)
		m.current = m.inbox
		// Manually set the size of the new view.
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, m.current.Init())

	case tui.GoToSendMsg:
		m.current = tui.NewComposer(m.config.Email)
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, m.current.Init())

	case tui.GoToSettingsMsg:
		m.current = tui.NewLogin()
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, m.current.Init())

	case tui.ViewEmailMsg:
		emailView := tui.NewEmailView(m.emails[msg.Index], m.width, m.height)
		m.current = emailView
		cmds = append(cmds, m.current.Init())

	case tui.SendEmailMsg:
		m.current = tui.NewStatus("Sending email...")
		cmds = append(cmds, m.current.Init(), sendEmail(m.config, msg))

	case tui.EmailResultMsg:
		m.current = tui.NewChoice()
		cmds = append(cmds, m.current.Init())
	}

	// Pass all other messages to the current view.
	m.current, cmd = m.current.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *mainModel) View() string {
	return m.current.View()
}

// markdownToHTML converts a Markdown string to an HTML string.
func markdownToHTML(md []byte) []byte {
	var buf bytes.Buffer
	p := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML in email.
		),
	)
	if err := p.Convert(md, &buf); err != nil {
		return md // Fallback to original markdown.
	}
	return buf.Bytes()
}

// sendEmail finds local image paths, embeds them, and sends the email.
func sendEmail(cfg *config.Config, msg tui.SendEmailMsg) tea.Cmd {
	return func() tea.Msg {
		recipients := []string{msg.To}
		body := msg.Body
		images := make(map[string][]byte)

		// Find all markdown image tags.
		re := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
		matches := re.FindAllStringSubmatch(body, -1)

		for _, match := range matches {
			imgPath := match[1]
			imgData, err := os.ReadFile(imgPath) // Use os.ReadFile.
			if err != nil {
				log.Printf("Could not read image file %s: %v", imgPath, err)
				continue
			}

			// Create a unique CID that includes the file extension.
			cid := fmt.Sprintf("%s%s@%s", uuid.NewString(), filepath.Ext(imgPath), "email-cli")
			images[cid] = []byte(base64.StdEncoding.EncodeToString(imgData))
			body = strings.Replace(body, imgPath, "cid:"+cid, 1)
		}

		htmlBody := markdownToHTML([]byte(body))

		err := sender.SendEmail(cfg, recipients, msg.Subject, msg.Body, string(htmlBody), images)
		if err != nil {
			log.Printf("Failed to send email: %v", err)
			return tui.EmailResultMsg{Err: err}
		}
		time.Sleep(1 * time.Second)
		return tui.EmailResultMsg{}
	}
}

// fetchEmails retrieves emails in the background.
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
	var initialModel *mainModel
	if err != nil {
		initialModel = newInitialModel(nil)
	} else {
		initialModel = newInitialModel(cfg)
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}