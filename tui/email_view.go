package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/floatpane/matcha/fetcher"
	"github.com/floatpane/matcha/view"
)

var (
	emailHeaderStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Padding(0, 1)
	attachmentBoxStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).PaddingLeft(2).MarginTop(1)
)

type EmailView struct {
	viewport           viewport.Model
	email              fetcher.Email
	emailIndex         int
	attachmentCursor   int
	focusOnAttachments bool
	accountID          string
}

func NewEmailView(email fetcher.Email, emailIndex, width, height int) *EmailView {
	// Pass the styles from the tui package to the view package
	body, err := view.ProcessBody(email.Body, H1Style, H2Style, BodyStyle)
	if err != nil {
		body = fmt.Sprintf("Error rendering body: %v", err)
	}

	header := fmt.Sprintf("From: %s\nSubject: %s", email.From, email.Subject)
	headerHeight := lipgloss.Height(header) + 2

	attachmentHeight := 0
	if len(email.Attachments) > 0 {
		attachmentHeight = len(email.Attachments) + 2
	}

	vp := viewport.New(width, height-headerHeight-attachmentHeight)
	vp.SetContent(body)

	return &EmailView{
		viewport:   vp,
		email:      email,
		emailIndex: emailIndex,
		accountID:  email.AccountID,
	}
}

func (m *EmailView) Init() tea.Cmd {
	return nil
}

func (m *EmailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle 'esc' key locally
		if msg.Type == tea.KeyEsc {
			if m.focusOnAttachments {
				m.focusOnAttachments = false
				return m, nil
			}
			return m, func() tea.Msg { return BackToInboxMsg{} }
		}

		if m.focusOnAttachments {
			switch msg.String() {
			case "up", "k":
				if m.attachmentCursor > 0 {
					m.attachmentCursor--
				}
			case "down", "j":
				if m.attachmentCursor < len(m.email.Attachments)-1 {
					m.attachmentCursor++
				}
			case "enter":
				if len(m.email.Attachments) > 0 {
					selected := m.email.Attachments[m.attachmentCursor]
					idx := m.emailIndex
					accountID := m.accountID
					return m, func() tea.Msg {
						return DownloadAttachmentMsg{
							Index:     idx,
							Filename:  selected.Filename,
							PartID:    selected.PartID,
							Data:      selected.Data,
							AccountID: accountID,
						}
					}
				}
			case "tab":
				m.focusOnAttachments = false
			}
		} else {
			switch msg.String() {
			case "r":
				return m, func() tea.Msg { return ReplyToEmailMsg{Email: m.email} }
			case "d":
				accountID := m.accountID
				uid := m.email.UID
				return m, func() tea.Msg {
					return DeleteEmailMsg{UID: uid, AccountID: accountID}
				}
			case "a":
				accountID := m.accountID
				uid := m.email.UID
				return m, func() tea.Msg {
					return ArchiveEmailMsg{UID: uid, AccountID: accountID}
				}
			case "tab":
				if len(m.email.Attachments) > 0 {
					m.focusOnAttachments = true
				}
			}
		}
	case tea.WindowSizeMsg:
		header := fmt.Sprintf("From: %s\nSubject: %s", m.email.From, m.email.Subject)
		headerHeight := lipgloss.Height(header) + 2
		attachmentHeight := 0
		if len(m.email.Attachments) > 0 {
			attachmentHeight = len(m.email.Attachments) + 2
		}
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - attachmentHeight
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *EmailView) View() string {
	header := fmt.Sprintf("From: %s | Subject: %s", m.email.From, m.email.Subject)
	styledHeader := emailHeaderStyle.Width(m.viewport.Width).Render(header)

	var help string
	if m.focusOnAttachments {
		help = helpStyle.Render("↑/↓: navigate • enter: download • esc/tab: back to email body")
	} else {
		help = helpStyle.Render("r: reply • d: delete • a: archive • tab: focus attachments • esc: back to inbox")
	}

	var attachmentView string
	if len(m.email.Attachments) > 0 {
		var b strings.Builder
		b.WriteString("Attachments:\n")
		for i, attachment := range m.email.Attachments {
			cursor := "  "
			style := itemStyle
			if m.focusOnAttachments && i == m.attachmentCursor {
				cursor = "> "
				style = selectedItemStyle
			}
			b.WriteString(style.Render(fmt.Sprintf("%s%s", cursor, attachment.Filename)))
			b.WriteString("\n")
		}
		attachmentView = attachmentBoxStyle.Render(b.String())
	}

	return fmt.Sprintf("%s\n%s\n%s\n%s", styledHeader, m.viewport.View(), attachmentView, help)
}

// GetAccountID returns the account ID for this email
func (m *EmailView) GetAccountID() string {
	return m.accountID
}

// GetEmail returns the email being viewed
func (m *EmailView) GetEmail() fetcher.Email {
	return m.email
}
