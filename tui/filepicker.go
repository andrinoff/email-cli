package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	filePickerItemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	filePickerSelectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("205"))
	directoryStyle              = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
)

type FilePicker struct {
	cursor      int
	currentPath string
	items       []fs.DirEntry
	width       int
	height      int
}

func NewFilePicker(startPath string) *FilePicker {
	fp := &FilePicker{currentPath: startPath}
	fp.readDir()
	return fp
}

func (m *FilePicker) readDir() {
	files, err := os.ReadDir(m.currentPath)
	if err != nil {
		// Handle error, maybe show a message
		m.items = []fs.DirEntry{}
		return
	}
	m.items = files
	m.cursor = 0 // Reset cursor
}

func (m *FilePicker) Init() tea.Cmd {
	return nil
}

func (m *FilePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.items) == 0 {
				return m, nil
			}
			selectedItem := m.items[m.cursor]
			newPath := filepath.Join(m.currentPath, selectedItem.Name())

			if selectedItem.IsDir() {
				m.currentPath = newPath
				m.readDir()
			} else {
				// It's a file, send a message with the path
				return m, func() tea.Msg {
					return FileSelectedMsg{Path: newPath}
				}
			}
		case "backspace":
			// Go up one directory
			parentDir := filepath.Dir(m.currentPath)
			if parentDir != m.currentPath { // Avoid getting stuck at root
				m.currentPath = parentDir
				m.readDir()
			}
		case "esc", "q":
			return m, func() tea.Msg { return CancelFilePickerMsg{} }
		}
	}
	return m, nil
}

func (m *FilePicker) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select a File") + "\n")
	b.WriteString(fmt.Sprintf("Current Path: %s\n\n", m.currentPath))

	for i, item := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		itemName := item.Name()
		if item.IsDir() {
			itemName = directoryStyle.Render(itemName + "/")
		}

		line := fmt.Sprintf("%s%s", cursor, itemName)

		if m.cursor == i {
			b.WriteString(filePickerSelectedItemStyle.Render(line))
		} else {
			b.WriteString(filePickerItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n" + helpStyle.Render("↑/↓: navigate • enter: select • backspace: up • esc: cancel"))

	return docStyle.Render(b.String())
}
