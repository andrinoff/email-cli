package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/floatpane/matcha/config"
)

var (
	accountItemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedAccountItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("42"))
	accountEmailStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	dangerStyle              = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

// Settings displays the account management screen.
type Settings struct {
	accounts         []config.Account
	cursor           int
	confirmingDelete bool
	width            int
	height           int
}

// NewSettings creates a new settings model.
func NewSettings(accounts []config.Account) *Settings {
	return &Settings{
		accounts: accounts,
		cursor:   0,
	}
}

// Init initializes the settings model.
func (m *Settings) Init() tea.Cmd {
	return nil
}

// Update handles messages for the settings model.
func (m *Settings) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.confirmingDelete {
			switch msg.String() {
			case "y", "Y":
				if m.cursor < len(m.accounts) {
					accountID := m.accounts[m.cursor].ID
					m.confirmingDelete = false
					return m, func() tea.Msg {
						return DeleteAccountMsg{AccountID: accountID}
					}
				}
			case "n", "N", "esc":
				m.confirmingDelete = false
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			// +1 for "Add Account" option
			if m.cursor < len(m.accounts) {
				m.cursor++
			}
		case "d":
			// Delete selected account (not the "Add Account" option)
			if m.cursor < len(m.accounts) && len(m.accounts) > 0 {
				m.confirmingDelete = true
			}
		case "enter":
			// If cursor is on "Add Account"
			if m.cursor == len(m.accounts) {
				return m, func() tea.Msg { return GoToAddAccountMsg{} }
			}
		case "esc":
			return m, func() tea.Msg { return GoToChoiceMenuMsg{} }
		}
	}
	return m, nil
}

// View renders the settings screen.
func (m *Settings) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Account Settings") + "\n\n")
	b.WriteString(listHeader.Render("Your email accounts:"))
	b.WriteString("\n\n")

	if len(m.accounts) == 0 {
		b.WriteString(accountEmailStyle.Render("  No accounts configured.\n"))
		b.WriteString("\n")
	}

	for i, account := range m.accounts {
		displayName := account.Email
		if account.Name != "" {
			displayName = fmt.Sprintf("%s (%s)", account.Name, account.Email)
		}

		providerInfo := account.ServiceProvider
		if account.ServiceProvider == "custom" {
			providerInfo = fmt.Sprintf("custom: %s", account.IMAPServer)
		}

		line := fmt.Sprintf("%s - %s", displayName, accountEmailStyle.Render(providerInfo))

		if m.cursor == i {
			b.WriteString(selectedAccountItemStyle.Render(fmt.Sprintf("> %s", line)))
		} else {
			b.WriteString(accountItemStyle.Render(fmt.Sprintf("  %s", line)))
		}
		b.WriteString("\n")
	}

	// Add Account option
	addAccountText := "Add New Account"
	if m.cursor == len(m.accounts) {
		b.WriteString(selectedAccountItemStyle.Render(fmt.Sprintf("> %s", addAccountText)))
	} else {
		b.WriteString(accountItemStyle.Render(fmt.Sprintf("  %s", addAccountText)))
	}
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • d: delete account • esc: back"))

	if m.confirmingDelete {
		accountName := m.accounts[m.cursor].Email
		dialog := DialogBoxStyle.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				dangerStyle.Render("Delete account?"),
				accountEmailStyle.Render(accountName),
				HelpStyle.Render("\n(y/n)"),
			),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	return docStyle.Render(b.String())
}

// UpdateAccounts updates the list of accounts.
func (m *Settings) UpdateAccounts(accounts []config.Account) {
	m.accounts = accounts
	if m.cursor >= len(accounts) {
		m.cursor = len(accounts)
	}
}
