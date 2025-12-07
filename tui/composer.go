package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/floatpane/matcha/config"
)

// Styles for the UI
var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	helpStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	focusedButton       = focusedStyle.Copy().Render("[ Send ]")
	blurredButton       = blurredStyle.Copy().Render("[ Send ]")
	emailRecipientStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	attachmentStyle     = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("240"))
	fromSelectorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

const (
	focusFrom = iota
	focusTo
	focusSubject
	focusBody
	focusAttachment
	focusSend
)

// Composer model holds the state of the email composition UI.
type Composer struct {
	focusIndex     int
	toInput        textinput.Model
	subjectInput   textinput.Model
	bodyInput      textarea.Model
	attachmentPath string
	width          int
	height         int
	confirmingExit bool

	// Multi-account support
	accounts           []config.Account
	selectedAccountIdx int
	showAccountPicker  bool
}

// NewComposer initializes a new composer model.
func NewComposer(from, to, subject, body string) *Composer {
	m := &Composer{}

	m.toInput = textinput.New()
	m.toInput.Cursor.Style = cursorStyle
	m.toInput.Placeholder = "To"
	m.toInput.SetValue(to)
	m.toInput.Prompt = "> "
	m.toInput.CharLimit = 256

	m.subjectInput = textinput.New()
	m.subjectInput.Cursor.Style = cursorStyle
	m.subjectInput.Placeholder = "Subject"
	m.subjectInput.SetValue(subject)
	m.subjectInput.Prompt = "> "
	m.subjectInput.CharLimit = 256

	m.bodyInput = textarea.New()
	m.bodyInput.Cursor.Style = cursorStyle
	m.bodyInput.Placeholder = "Body (Markdown supported)..."
	m.bodyInput.SetValue(body)
	m.bodyInput.Prompt = "> "
	m.bodyInput.SetHeight(10)
	m.bodyInput.SetCursor(0)

	// Start focus on To field (From is selectable but not a text input)
	m.focusIndex = focusTo
	m.toInput.Focus()

	return m
}

// NewComposerWithAccounts initializes a composer with multiple account support.
func NewComposerWithAccounts(accounts []config.Account, selectedAccountID string, to, subject, body string) *Composer {
	m := NewComposer("", to, subject, body)
	m.accounts = accounts

	// Find the selected account index
	for i, acc := range accounts {
		if acc.ID == selectedAccountID {
			m.selectedAccountIdx = i
			break
		}
	}

	return m
}

// ResetConfirmation ensures a restored draft isn't stuck in the exit prompt.
func (m *Composer) ResetConfirmation() {
	m.confirmingExit = false
}

func (m *Composer) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Composer) getFromAddress() string {
	if len(m.accounts) > 0 && m.selectedAccountIdx < len(m.accounts) {
		acc := m.accounts[m.selectedAccountIdx]
		if acc.Name != "" {
			return fmt.Sprintf("%s <%s>", acc.Name, acc.Email)
		}
		return acc.Email
	}
	return ""
}

func (m *Composer) getSelectedAccount() *config.Account {
	if len(m.accounts) > 0 && m.selectedAccountIdx < len(m.accounts) {
		return &m.accounts[m.selectedAccountIdx]
	}
	return nil
}

func (m *Composer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		inputWidth := msg.Width - 6
		m.toInput.Width = inputWidth
		m.subjectInput.Width = inputWidth
		m.bodyInput.SetWidth(inputWidth)

	case SetComposerCursorToStartMsg:
		m.bodyInput.SetCursor(0)
		return m, nil

	case FileSelectedMsg:
		m.attachmentPath = msg.Path
		return m, nil

	case tea.KeyMsg:
		// Handle account picker mode
		if m.showAccountPicker {
			switch msg.String() {
			case "up", "k":
				if m.selectedAccountIdx > 0 {
					m.selectedAccountIdx--
				}
			case "down", "j":
				if m.selectedAccountIdx < len(m.accounts)-1 {
					m.selectedAccountIdx++
				}
			case "enter":
				m.showAccountPicker = false
			case "esc":
				m.showAccountPicker = false
			}
			return m, nil
		}

		if m.confirmingExit {
			switch msg.String() {
			case "y", "Y":
				return m, func() tea.Msg { return DiscardDraftMsg{ComposerState: m} }
			case "n", "N", "esc":
				m.confirmingExit = false
				return m, nil
			default:
				return m, nil
			}
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			m.confirmingExit = true
			return m, nil

		case tea.KeyTab, tea.KeyShiftTab:
			if msg.Type == tea.KeyShiftTab {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			maxFocus := focusSend
			if m.focusIndex > maxFocus {
				m.focusIndex = focusFrom
			} else if m.focusIndex < focusFrom {
				m.focusIndex = maxFocus
			}

			m.toInput.Blur()
			m.subjectInput.Blur()
			m.bodyInput.Blur()

			switch m.focusIndex {
			case focusTo:
				cmds = append(cmds, m.toInput.Focus())
			case focusSubject:
				cmds = append(cmds, m.subjectInput.Focus())
			case focusBody:
				cmds = append(cmds, m.bodyInput.Focus())
				cmds = append(cmds, func() tea.Msg { return SetComposerCursorToStartMsg{} })
			}
			return m, tea.Batch(cmds...)

		case tea.KeyEnter:
			switch m.focusIndex {
			case focusFrom:
				if len(m.accounts) > 1 {
					m.showAccountPicker = true
				}
				return m, nil
			case focusAttachment:
				return m, func() tea.Msg { return GoToFilePickerMsg{} }
			case focusSend:
				acc := m.getSelectedAccount()
				accountID := ""
				if acc != nil {
					accountID = acc.ID
				}
				return m, func() tea.Msg {
					return SendEmailMsg{
						To:             m.toInput.Value(),
						Subject:        m.subjectInput.Value(),
						Body:           m.bodyInput.Value(),
						AttachmentPath: m.attachmentPath,
						AccountID:      accountID,
					}
				}
			}
		}
	}

	switch m.focusIndex {
	case focusTo:
		m.toInput, cmd = m.toInput.Update(msg)
		cmds = append(cmds, cmd)
	case focusSubject:
		m.subjectInput, cmd = m.subjectInput.Update(msg)
		cmds = append(cmds, cmd)
	case focusBody:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Composer) View() string {
	var composerView strings.Builder
	var button string

	if m.focusIndex == focusSend {
		button = focusedButton
	} else {
		button = blurredButton
	}

	// From field with account selector
	fromAddr := m.getFromAddress()
	var fromField string
	if len(m.accounts) > 1 {
		if m.focusIndex == focusFrom {
			fromField = focusedStyle.Render(fmt.Sprintf("> From: %s (press Enter to change)", fromAddr))
		} else {
			fromField = blurredStyle.Render(fmt.Sprintf("  From: %s", fromAddr))
		}
	} else {
		fromField = "From: " + emailRecipientStyle.Render(fromAddr)
	}

	var attachmentField string
	attachmentText := "None (Press Enter to select)"
	if m.attachmentPath != "" {
		attachmentText = m.attachmentPath
	}

	if m.focusIndex == focusAttachment {
		attachmentField = focusedStyle.Render(fmt.Sprintf("> Attachment: %s", attachmentText))
	} else {
		attachmentField = blurredStyle.Render(fmt.Sprintf("  Attachment: %s", attachmentText))
	}

	composerView.WriteString(lipgloss.JoinVertical(lipgloss.Left,
		"Compose New Email",
		fromField,
		m.toInput.View(),
		m.subjectInput.View(),
		m.bodyInput.View(),
		attachmentStyle.Render(attachmentField),
		button,
		helpStyle.Render("Markdown/HTML • tab: next field • esc: back to menu"),
	))

	// Account picker overlay
	if m.showAccountPicker {
		var accountList strings.Builder
		accountList.WriteString("Select Account:\n\n")
		for i, acc := range m.accounts {
			display := acc.Email
			if acc.Name != "" {
				display = fmt.Sprintf("%s (%s)", acc.Name, acc.Email)
			}
			if i == m.selectedAccountIdx {
				accountList.WriteString(selectedItemStyle.Render(fmt.Sprintf("> %s", display)))
			} else {
				accountList.WriteString(itemStyle.Render(fmt.Sprintf("  %s", display)))
			}
			accountList.WriteString("\n")
		}
		accountList.WriteString("\n")
		accountList.WriteString(HelpStyle.Render("↑/↓: navigate • enter: select • esc: cancel"))

		dialog := DialogBoxStyle.Render(accountList.String())
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.confirmingExit {
		dialog := DialogBoxStyle.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				"Discard draft?",
				HelpStyle.Render("\n(y/n)"),
			),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	return composerView.String()
}

// SetAccounts sets the available accounts for sending.
func (m *Composer) SetAccounts(accounts []config.Account) {
	m.accounts = accounts
	if m.selectedAccountIdx >= len(accounts) {
		m.selectedAccountIdx = 0
	}
}

// SetSelectedAccount sets the selected account by ID.
func (m *Composer) SetSelectedAccount(accountID string) {
	for i, acc := range m.accounts {
		if acc.ID == accountID {
			m.selectedAccountIdx = i
			return
		}
	}
}

// GetSelectedAccountID returns the ID of the currently selected account.
func (m *Composer) GetSelectedAccountID() string {
	if len(m.accounts) > 0 && m.selectedAccountIdx < len(m.accounts) {
		return m.accounts[m.selectedAccountIdx].ID
	}
	return ""
}
