package tui

import (
	"fmt"
	"io"

	"github.com/andrinoff/email-cli/fetcher"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("205"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	inboxHelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title + " " + i.desc }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.title)

	// **FIX**: Corrected the function literal to accept a variadic string
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + s[0])
		}
	}

	fmt.Fprint(w, fn(str))
}

type Inbox struct {
	list list.Model
}

func NewInbox(emails []fetcher.Email) Inbox {
	items := make([]list.Item, len(emails))
	for i, email := range emails {
		items[i] = item{
			title: email.Subject,
			desc:  email.From,
		}
	}

	l := list.New(items, itemDelegate{}, 20, 14)
	l.Title = "Inbox"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = inboxHelpStyle

	return Inbox{list: l}
}

func (m Inbox) Init() tea.Cmd {
	return nil
}

func (m Inbox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				_ = i
				return m, func() tea.Msg {
					return ViewEmailMsg{Index: m.list.Index()}
				}
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Inbox) View() string {
	return "\n" + m.list.View()
}