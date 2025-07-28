package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Login holds the state for the login form.
type Login struct {
	focusIndex int
	inputs     []textinput.Model
}

// NewLogin creates a new login model. This function was missing.
func NewLogin() tea.Model {
	m := Login{
		inputs: make([]textinput.Model, 2),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = focusedStyle
		t.CharLimit = 64

		switch i {
		case 0:
			t.Placeholder = "Email"
			t.Focus()
			t.Prompt = "✉️ > "
		case 1:
			t.Placeholder = "Password"
			t.EchoMode = textinput.EchoPassword
			t.Prompt = "🔑 > "
		}
		m.inputs[i] = t
	}

	return m
}

// Init initializes the login model.
func (m Login) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the login model.
func (m Login) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		// On Enter, if we are on the last field, submit the credentials.
		case tea.KeyEnter:
			if m.focusIndex == len(m.inputs)-1 {
				return m, func() tea.Msg {
					return Credentials{
						Email:    m.inputs[0].Value(),
						Password: m.inputs[1].Value(),
					}
				}
			}
			fallthrough
		// Cycle focus between inputs.
		case tea.KeyTab, tea.KeyShiftTab, tea.KeyUp, tea.KeyDown:
			s := msg.String()
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)
		}
	}

	// Update the focused input field.
	var cmds = make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

// View renders the login form.
func (m Login) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Welcome to Email CLI"),
		"Please enter your credentials.",
		m.inputs[0].View(),
		m.inputs[1].View(),
		helpStyle.Render("\nenter: submit • tab: next field • esc: quit"),
	)
}