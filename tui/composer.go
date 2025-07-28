package tui

import (
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
	focusedButton       = focusedStyle.Copy().Render("[ Send ]")
	blurredButton       = blurredStyle.Copy().Render("[ Send ]")
	emailRecipientStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)

// Composer model holds the state of the email composition UI.
type Composer struct {
	focusIndex int
	toInput    textinput.Model
	subjectInput textinput.Model
	bodyInput  textarea.Model
	fromAddr   string
}

// NewComposer initializes a new composer model.
func NewComposer(from string) Composer {
	m := Composer{fromAddr: from}

	m.toInput = textinput.New()
	m.toInput.Cursor.Style = cursorStyle
	m.toInput.Placeholder = "To"
	m.toInput.Focus()
	m.toInput.Prompt = "> "
	m.toInput.CharLimit = 256

	m.subjectInput = textinput.New()
	m.subjectInput.Cursor.Style = cursorStyle
	m.subjectInput.Placeholder = "Subject"
	m.subjectInput.Prompt = "> "
	m.subjectInput.CharLimit = 256

	m.bodyInput = textarea.New()
	m.bodyInput.Cursor.Style = cursorStyle
	m.bodyInput.Placeholder = "Body..."
	m.bodyInput.Prompt = "> "
	m.bodyInput.SetHeight(10)
	m.bodyInput.SetWidth(60)

	return m
}

func (m Composer) Init() tea.Cmd {
	return textinput.Blink
}

func (m Composer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		// Handle Tab and Shift+Tab to cycle focus between inputs.
		case tea.KeyTab, tea.KeyShiftTab:
			if msg.Type == tea.KeyShiftTab {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			// Wrap around
			if m.focusIndex > 3 { // 3 is the Send button
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 3
			}
			
			// Blur all inputs
			m.toInput.Blur()
			m.subjectInput.Blur()
			m.bodyInput.Blur()

			// Focus the correct input
			switch m.focusIndex {
			case 0:
				m.toInput.Focus()
			case 1:
				m.subjectInput.Focus()
			case 2:
				m.bodyInput.Focus()
			}
			return m, tea.Batch(cmds...)

		// Handle Enter key.
		case tea.KeyEnter:
			// If on the Send button, send the email.
			if m.focusIndex == 3 {
				return m, func() tea.Msg {
					return SendEmailMsg{
						To:      m.toInput.Value(),
						Subject: m.subjectInput.Value(),
						Body:    m.bodyInput.Value(),
					}
				}
			}
		}
	}

	// Update the focused input.
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

// View renders the UI.
func (m Composer) View() string {
	button := &blurredButton
	if m.focusIndex == 3 {
		button = &focusedButton
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		"Compose New Email",
		"From: "+emailRecipientStyle.Render(m.fromAddr),
		m.toInput.View(),
		m.subjectInput.View(),
		m.bodyInput.View(),
		*button,
		helpStyle.Render("tab: next field • esc: quit"),
	)
}