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
	mailbox            MailboxKind
}

func NewEmailView(email fetcher.Email, emailIndex, width, height int, mailbox MailboxKind) *EmailView {
	// Pass the styles from the tui package to the view package
	body, err := view.ProcessBody(email.Body, H1Style, H2Style, BodyStyle)
	if err != nil {
		body = fmt.Sprintf("Error rendering body: %v", err)
	}

	// Create header and compute heights that reduce viewport space.
	header := fmt.Sprintf("From: %s\nSubject: %s", email.From, email.Subject)
	headerHeight := lipgloss.Height(header) + 2

	attachmentHeight := 0
	if len(email.Attachments) > 0 {
		attachmentHeight = len(email.Attachments) + 2
	}

	// Build viewport with initial size and set wrapped content to fit the width.
	vp := viewport.New(width, height-headerHeight-attachmentHeight)

	// Wrap the body to the viewport width so lines don't run off-screen.
	wrapped := wrapBodyToWidth(body, vp.Width)
	vp.SetContent(wrapped)

	return &EmailView{
		viewport:   vp,
		email:      email,
		emailIndex: emailIndex,
		accountID:  email.AccountID,
		mailbox:    mailbox,
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
			return m, func() tea.Msg { return BackToMailboxMsg{Mailbox: m.mailbox} }
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
							Mailbox:   m.mailbox,
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
					return DeleteEmailMsg{UID: uid, AccountID: accountID, Mailbox: m.mailbox}
				}
			case "a":
				accountID := m.accountID
				uid := m.email.UID
				return m, func() tea.Msg {
					return ArchiveEmailMsg{UID: uid, AccountID: accountID, Mailbox: m.mailbox}
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
		// Update viewport dimensions
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - attachmentHeight

		// When the window size changes, rewrap the body to the new width
		body, err := view.ProcessBody(m.email.Body, H1Style, H2Style, BodyStyle)
		if err != nil {
			body = fmt.Sprintf("Error rendering body: %v", err)
		}
		wrapped := wrapBodyToWidth(body, m.viewport.Width)
		m.viewport.SetContent(wrapped)
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

func wrapBodyToWidth(body string, width int) string {
	return BodyStyle.Width(width).Render(body)
}

// GetEmail returns the email being viewed
func (m *EmailView) GetEmail() fetcher.Email {
	return m.email
}
