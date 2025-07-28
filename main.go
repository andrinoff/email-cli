package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- STYLES ---
// Define styles for different parts of the UI using lipgloss.
var (
	// Style for the main container/dialog box.
	dialogBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 0).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	// Style for the text labels (e.g., "To:", "Subject:").
	labelStyle = lipgloss.NewStyle().Padding(0, 1)

	// Style for the help text at the bottom.
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Style for success messages.
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)

	// Style for error/sending messages.
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)

// --- MODEL ---
// The model represents the state of our TUI application.
type model struct {
	// Inputs for the email fields.
	recipientInput textinput.Model
	subjectInput   textinput.Model
	bodyArea       textarea.Model

	// index tracks which input is currently focused.
	// 0: recipient, 1: subject, 2: body
	index int

	// State flags
	sending bool // Is the email currently being "sent"?
	sent    bool // Has the email been "sent"?
	err     error // Any error that occurred.
}

// initialModel creates the starting state of the application.
func initialModel() model {
	// --- Recipient Input ---
	ti := textinput.New()
	ti.Placeholder = "recipient@example.com"
	ti.Focus() // Start with the recipient field focused.
	ti.CharLimit = 156
	ti.Width = 50
	ti.Prompt = "To:      "

	// --- Subject Input ---
	si := textinput.New()
	si.Placeholder = "Hello there!"
	si.CharLimit = 156
	si.Width = 50
	si.Prompt = "Subject: "

	// --- Body Text Area ---
	ta := textarea.New()
	ta.Placeholder = "Dear friend, ..."
	ta.SetWidth(50)
	ta.SetHeight(10)

	return model{
		recipientInput: ti,
		subjectInput:   si,
		bodyArea:       ta,
		index:          0,
		err:            nil,
	}
}

// --- BUBBLETEA METHODS ---

// Init is the first function that will be called. It returns a command.
// We'll use it to make the cursor blink.
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all user input and updates the model accordingly.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	// Handle key presses
	case tea.KeyMsg:
		switch msg.Type {
		// Quit the application
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		// Trigger the "send" action
		case tea.KeyCtrlS:
			// If we are not already sending, start the sending process.
			if !m.sending {
				m.sending = true
				// In a real app, this is where you'd send the email.
				// We'll return a command that simulates a network operation.
				// For now, we'll just pretend it was successful after a short delay.
				return m, func() tea.Msg {
					// time.Sleep(time.Second * 2) // Simulate network delay
					return "sent" // Send a message back on success
				}
			}

		// Handle Tab and Shift+Tab to navigate between fields
		case tea.KeyTab, tea.KeyShiftTab:
			if msg.Type == tea.KeyTab {
				m.index = (m.index + 1) % 3
			} else {
				m.index--
				if m.index < 0 {
					m.index = 2
				}
			}

			// Blur all inputs
			m.recipientInput.Blur()
			m.subjectInput.Blur()
			m.bodyArea.Blur()

			// Focus the correct input based on the index
			switch m.index {
			case 0:
				m.recipientInput.Focus()
			case 1:
				m.subjectInput.Focus()
			case 2:
				m.bodyArea.Focus()
			}
			return m, nil
		}

	// Handle the "sent" message from our simulated send command
	case string:
		if msg == "sent" {
			m.sending = false
			m.sent = true
			// Quit after a short delay to show the success message.
			return m, tea.Sequence(
				func() tea.Msg {
					// time.Sleep(time.Second)
					return nil
				},
				tea.Quit,
			)
		}
	}

	// Pass the message to the focused input field for handling.
	switch m.index {
	case 0:
		m.recipientInput, cmd = m.recipientInput.Update(msg)
	case 1:
		m.subjectInput, cmd = m.subjectInput.Update(msg)
	case 2:
		m.bodyArea, cmd = m.bodyArea.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the UI based on the current model state.
func (m model) View() string {
	// If the email was sent, just show a success message.
	if m.sent {
		return successStyle.Render("\n   ✓ Email sent successfully! \n\n")
	}

	// If we are "sending", show an info message.
	if m.sending {
		return infoStyle.Render("\n   Sending email... \n\n")
	}

	// Otherwise, render the form.
	var s string

	s += "\n"
	s += m.recipientInput.View() + "\n"
	s += m.subjectInput.View() + "\n"
	s += m.bodyArea.View()

	help := fmt.Sprintf("\n\n %s | %s | %s | %s \n",
		"Tab: Next Field", "Esc: Quit", "Ctrl+S: Send", helpStyle.Render("Focus: "+m.focusString()))
	s += helpStyle.Render(help)

	return dialogBoxStyle.Render(s)
}

// Helper function to show which field is focused.
func (m model) focusString() string {
	switch m.index {
	case 0:
		return "Recipient"
	case 1:
		return "Subject"
	case 2:
		return "Body"
	default:
		return ""
	}
}

// --- MAIN FUNCTION ---
func main() {
	// Initialize the Bubble Tea program.
	p := tea.NewProgram(initialModel())

	// Run the program and handle any errors.
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
