package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/floatpane/matcha/fetcher"
)

var (
	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	inboxHelpStyle  = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type item struct {
	title, desc   string
	originalIndex int
	uid           uint32 // Added UID to item
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title + " " + i.desc }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.title)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + s[0])
		}
	}

	fmt.Fprint(w, fn(str))
}

type Inbox struct {
	list        list.Model
	isFetching  bool
	emailsCount int
}

func NewInbox(emails []fetcher.Email) *Inbox {
	items := make([]list.Item, len(emails))
	for i, email := range emails {
		items[i] = item{
			title:         email.Subject,
			desc:          email.From,
			originalIndex: i,
			uid:           email.UID, // Store UID
		}
	}

	l := list.New(items, itemDelegate{}, 20, 14)
	l.Title = "Inbox"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = inboxHelpStyle
	l.SetStatusBarItemName("email", "emails")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "archive")),
			key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl + c", "quit")),
		}
	}

	l.KeyMap.Quit.SetEnabled(false)

	return &Inbox{
		list:        l,
		isFetching:  false,
		emailsCount: len(emails),
	}
}

func (m *Inbox) Init() tea.Cmd {
	return nil
}

func (m *Inbox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch keypress := msg.String(); keypress {
		case "d":
			selectedItem, ok := m.list.SelectedItem().(item)
			if ok {
				return m, func() tea.Msg {
					return DeleteEmailMsg{UID: selectedItem.uid}
				}
			}
		case "a":
			selectedItem, ok := m.list.SelectedItem().(item)
			if ok {
				return m, func() tea.Msg {
					return ArchiveEmailMsg{UID: selectedItem.uid}
				}
			}
		case "enter":
			selectedItem, ok := m.list.SelectedItem().(item)
			if ok {
				return m, func() tea.Msg {
					return ViewEmailMsg{Index: selectedItem.originalIndex}
				}
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case FetchingMoreEmailsMsg:
		m.isFetching = true
		m.list.Title = "Fetching more emails..."
		return m, nil

	case EmailsAppendedMsg:
		m.isFetching = false
		m.list.Title = "Inbox"
		newItems := make([]list.Item, len(msg.Emails))
		for i, email := range msg.Emails {
			newItems[i] = item{
				title:         email.Subject,
				desc:          email.From,
				originalIndex: m.emailsCount + i,
				uid:           email.UID,
			}
		}
		currentItems := m.list.Items()
		allItems := append(currentItems, newItems...)
		cmd := m.list.SetItems(allItems)
		m.emailsCount += len(msg.Emails)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	if !m.isFetching && len(m.list.Items()) > 0 && m.list.Index() >= len(m.list.Items())-5 {
		cmds = append(cmds, func() tea.Msg {
			return FetchMoreEmailsMsg{Offset: uint32(m.emailsCount)}
		})
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Inbox) View() string {
	return "\n" + m.list.View()
}
