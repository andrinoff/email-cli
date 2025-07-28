package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/andrinoff/email-cli/config"
	"github.com/andrinoff/email-cli/sender"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type composerModel struct {
	recipientInput textinput.Model
	subjectInput   textinput.Model
	bodyArea       textarea.Model
	index          int
	sending        bool
	sent           bool
	err            error
	config         *config.Config
}

func initialComposerModel(cfg *config.Config) composerModel {
	ti := textinput.New()
	ti.Placeholder = "recipient@example.com"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50
	ti.Prompt = "To:      "
	si := textinput.New()
	si.Placeholder = "Hello there!"
	si.CharLimit = 156
	si.Width = 50
	si.Prompt = "Subject: "
	ta := textarea.New()
	ta.Placeholder = "Dear friend, ..."
	ta.SetWidth(50)
	ta.SetHeight(10)
	return composerModel{
		recipientInput: ti,
		subjectInput:   si,
		bodyArea:       ta,
		index:          0,
		config:         cfg,
		err:            nil,
	}
}

func (m composerModel) Init() tea.Cmd { return textinput.Blink }

func (m composerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlS:
			if !m.sending {
				m.sending = true
				to := []string{m.recipientInput.Value()}
				subject := m.subjectInput.Value()
				body := m.bodyArea.Value()
				return m, func() tea.Msg {
					err := sender.SendEmail(m.config, to, subject, body)
					if err != nil {
						return err
					}
					return "sent"
				}
			}
		case tea.KeyTab, tea.KeyShiftTab:
			if msg.Type == tea.KeyTab {
				m.index = (m.index + 1) % 3
			} else {
				m.index--
				if m.index < 0 {
					m.index = 2
				}
			}
			m.recipientInput.Blur()
			m.subjectInput.Blur()
			m.bodyArea.Blur()
			switch m.index {
			case 0:
				m.recipientInput.Focus()
			case 1:
				m.subjectInput.Focus()
			case 2:
				m.bodyArea.Focus()
			}
			return m, tea.Batch(cmds...)
		}
	case string:
		if msg == "sent" {
			m.sending = false
			m.sent = true
			return m, tea.Sequence(func() tea.Msg { time.Sleep(time.Second); return nil }, tea.Quit)
		}
	case error:
		m.sending = false
		m.err = msg
		return m, nil
	}
	var cmd tea.Cmd
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

func (m composerModel) View() string {
	if m.err != nil {
		return InfoStyle.Render(fmt.Sprintf("\n   Error: %v \n\n Press any key to exit.", m.err))
	}
	if m.sent {
		return SuccessStyle.Render("\n   ✓ Email sent successfully! \n\n")
	}
	if m.sending {
		return InfoStyle.Render("\n   Sending email... \n\n")
	}
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(m.recipientInput.View() + "\n")
	s.WriteString(m.subjectInput.View() + "\n")
	s.WriteString(m.bodyArea.View())
	help := fmt.Sprintf("\n\n %s | %s | %s \n", "Tab: Next Field", "Esc: Quit", "Ctrl+S: Send")
	s.WriteString(HelpStyle.Render(help))
	return DialogBoxStyle.Render(s.String())
}

// RunComposer starts the email composer UI.
func RunComposer(cfg *config.Config) error {
	p := tea.NewProgram(initialComposerModel(cfg))
	_, err := p.Run()
	return err
}