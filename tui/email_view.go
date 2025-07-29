package tui

import (
	"fmt"

	"github.com/andrinoff/email-cli/fetcher"
	"github.com/andrinoff/email-cli/view"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	emailHeaderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				Padding(0, 1)
)

type EmailView struct {
	viewport viewport.Model
	email    fetcher.Email
}

func NewEmailView(email fetcher.Email, width, height int) *EmailView {
	body, err := view.ProcessBody(email.Body)
	if err != nil {
		body = fmt.Sprintf("Error rendering body: %v", err)
	}

	header := fmt.Sprintf("From: %s\nSubject: %s", email.From, email.Subject)
	headerHeight := lipgloss.Height(header) + 2 // Account for padding and border

	vp := viewport.New(width, height-headerHeight)
	vp.SetContent(body)

	return &EmailView{
		viewport: vp,
		email:    email,
	}
}

func (m *EmailView) Init() tea.Cmd {
	return nil
}

func (m *EmailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return m, func() tea.Msg {
				return ReplyToEmailMsg{Email: m.email}
			}
		}
	case tea.WindowSizeMsg:
		header := fmt.Sprintf("From: %s\nSubject: %s", m.email.From, m.email.Subject)
		headerHeight := lipgloss.Height(header) + 2
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *EmailView) View() string {
	header := fmt.Sprintf("From: %s\nSubject: %s", m.email.From, m.email.Subject)
	styledHeader := emailHeaderStyle.Width(m.viewport.Width).Render(header)
	help := helpStyle.Render("r: reply • esc: back to inbox")
	return fmt.Sprintf("%s\n%s\n%s", styledHeader, m.viewport.View(), help)
}