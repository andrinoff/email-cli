package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the UI
var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	helpStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	focusedButton       = focusedStyle.Copy().Render("[ Send ]")
	blurredButton       = blurredStyle.Copy().Render("[ Send ]")
	emailRecipientStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	attachmentStyle     = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("240")) // This was the missing style
)

// Composer model holds the state of the email composition UI.
type Composer struct {
	focusIndex     int
	toInput        textinput.Model
	subjectInput   textinput.Model
	bodyInput      textarea.Model
	attachmentPath string
	fromAddr       string
	width          int
	height         int
	confirmingExit bool
}

// NewComposer initializes a new composer model.
func NewComposer(from, to, subject, body string) *Composer {
	m := &Composer{fromAddr: from}

	m.toInput = textinput.New()
	m.toInput.Cursor.Style = cursorStyle
	m.toInput.Placeholder = "To"
	m.toInput.SetValue(to)
	m.toInput.Focus()
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

	return m
}

// ResetConfirmation ensures a restored draft isn't stuck in the exit prompt.
func (m *Composer) ResetConfirmation() {
	m.confirmingExit = false
}

func (m *Composer) Init() tea.Cmd {
	return textinput.Blink
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

			if m.focusIndex > 4 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 4
			}

			m.toInput.Blur()
			m.subjectInput.Blur()
			m.bodyInput.Blur()

			switch m.focusIndex {
			case 0:
				cmds = append(cmds, m.toInput.Focus())
			case 1:
				cmds = append(cmds, m.subjectInput.Focus())
			case 2:
				cmds = append(cmds, m.bodyInput.Focus())
				cmds = append(cmds, func() tea.Msg { return SetComposerCursorToStartMsg{} })
			}
			return m, tea.Batch(cmds...)

		case tea.KeyEnter:
			if m.focusIndex == 3 {
				return m, func() tea.Msg { return GoToFilePickerMsg{} }
			}
			if m.focusIndex == 4 {
				return m, func() tea.Msg {
					return SendEmailMsg{
						To:             m.toInput.Value(),
						Subject:        m.subjectInput.Value(),
						Body:           m.bodyInput.Value(),
						AttachmentPath: m.attachmentPath,
					}
				}
			}
		}
	}

	switch m.focusIndex {
	case 0:
		m.toInput, cmd = m.toInput.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		m.subjectInput, cmd = m.subjectInput.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Composer) View() string {
	var composerView strings.Builder
	var button string

	if m.focusIndex == 4 {
		button = focusedButton
	} else {
		button = blurredButton
	}

	var attachmentField string
	attachmentText := "None (Press Enter to select)"
	if m.attachmentPath != "" {
		attachmentText = m.attachmentPath
	}

	if m.focusIndex == 3 {
		attachmentField = focusedStyle.Render(fmt.Sprintf("> Attachment: %s", attachmentText))
	} else {
		attachmentField = blurredStyle.Render(fmt.Sprintf("  Attachment: %s", attachmentText))
	}

	composerView.WriteString(lipgloss.JoinVertical(lipgloss.Left,
		"Compose New Email",
		"From: "+emailRecipientStyle.Render(m.fromAddr),
		m.toInput.View(),
		m.subjectInput.View(),
		m.bodyInput.View(),
		attachmentStyle.Render(attachmentField),
		button,
		helpStyle.Render("Markdown/HTML • tab: next field • esc: back to menu"),
	))

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
