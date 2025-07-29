package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/andrinoff/email-cli/config"
	"github.com/andrinoff/email-cli/fetcher"
	"github.com/andrinoff/email-cli/sender"
	"github.com/andrinoff/email-cli/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

const (
	initialEmailLimit = 20
	paginationLimit   = 20
)

type mainModel struct {
	current       tea.Model
	previousModel tea.Model
	config        *config.Config
	emails        []fetcher.Email
	inbox         *tui.Inbox
	width         int
	height        int
	err           error
}

func newInitialModel(cfg *config.Config) *mainModel {
	if cfg == nil {
		return &mainModel{
			current: tui.NewLogin(),
		}
	}
	return &mainModel{
		current: tui.NewChoice(),
		config:  cfg,
	}
}

func (m *mainModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m *mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.current, cmd = m.current.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "esc" {
			switch m.current.(type) {
			case *tui.EmailView:
				m.current = m.inbox
				return m, nil
			case *tui.FilePicker:
				return m, func() tea.Msg { return tui.CancelFilePickerMsg{} }
			case *tui.Inbox, *tui.Composer, *tui.Login:
				m.current = tui.NewChoice()
				return m, m.current.Init()
			}
		}

	case tui.Credentials:
		cfg := &config.Config{
			ServiceProvider: msg.Provider,
			Name:            msg.Name,
			Email:           msg.Email,
			Password:        msg.Password,
		}
		if err := config.SaveConfig(cfg); err != nil {
			log.Printf("could not save config: %v", err)
			return m, tea.Quit
		}
		m.config = cfg
		m.current = tui.NewChoice()
		cmds = append(cmds, m.current.Init())

	case tui.GoToInboxMsg:
		m.current = tui.NewStatus("Fetching emails...")
		return m, tea.Batch(m.current.Init(), fetchEmails(m.config, initialEmailLimit, 0))

	case tui.EmailsFetchedMsg:
		m.emails = msg.Emails
		m.inbox = tui.NewInbox(m.emails)
		m.current = m.inbox
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, m.current.Init())

	case tui.FetchMoreEmailsMsg:
		cmds = append(cmds, func() tea.Msg { return tui.FetchingMoreEmailsMsg{} })
		cmds = append(cmds, fetchEmails(m.config, paginationLimit, msg.Offset))
		return m, tea.Batch(cmds...)

	case tui.EmailsAppendedMsg:
		m.emails = append(m.emails, msg.Emails...)
		m.current, cmd = m.current.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case tui.GoToSendMsg:
		m.current = tui.NewComposer(m.config.Email, msg.To, msg.Subject, msg.Body)
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, m.current.Init())

	case tui.GoToSettingsMsg:
		m.current = tui.NewLogin()
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, m.current.Init())

	case tui.ViewEmailMsg:
		emailView := tui.NewEmailView(m.emails[msg.Index], m.width, m.height)
		m.current = emailView
		cmds = append(cmds, m.current.Init())

	case tui.ReplyToEmailMsg:
		to := msg.Email.From
		subject := "Re: " + msg.Email.Subject
		body := fmt.Sprintf("\n\nOn %s, %s wrote:\n> %s", msg.Email.Date.Format("Jan 2, 2006 at 3:04 PM"), msg.Email.From, strings.ReplaceAll(msg.Email.Body, "\n", "\n> "))
		m.current = tui.NewComposer(m.config.Email, to, subject, body)
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, m.current.Init())

	case tui.GoToFilePickerMsg:
		m.previousModel = m.current
		wd, _ := os.Getwd()
		m.current = tui.NewFilePicker(wd)
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.FileSelectedMsg:
		if m.previousModel != nil {
			m.previousModel, cmd = m.previousModel.Update(msg)
			cmds = append(cmds, cmd)
			m.current = m.previousModel
			m.previousModel = nil
		}
		return m, tea.Batch(cmds...)

	case tui.CancelFilePickerMsg:
		if m.previousModel != nil {
			m.current = m.previousModel
			m.previousModel = nil
		}
		return m, nil

	case tui.SendEmailMsg:
		m.current = tui.NewStatus("Sending email...")
		cmds = append(cmds, m.current.Init(), sendEmail(m.config, msg))

	case tui.EmailResultMsg:
		m.current = tui.NewChoice()
		cmds = append(cmds, m.current.Init())

	case tui.DeleteEmailMsg:
		m.current = tui.NewStatus("Deleting email...")
		cmds = append(cmds, m.current.Init(), deleteEmailCmd(m.config, msg.UID))

	case tui.ArchiveEmailMsg:
		m.current = tui.NewStatus("Archiving email...")
		cmds = append(cmds, m.current.Init(), archiveEmailCmd(m.config, msg.UID))

	case tui.EmailActionDoneMsg:
		if msg.Err != nil {
			// In a real app, you might show an error message.
			// For now, we'll just go back to the inbox.
			log.Printf("Action failed: %v", msg.Err)
			m.current = m.inbox
			return m, nil
		}
		// Remove the email from the local cache
		var updatedEmails []fetcher.Email
		for _, email := range m.emails {
			if email.UID != msg.UID {
				updatedEmails = append(updatedEmails, email)
			}
		}
		m.emails = updatedEmails
		// Refresh the inbox view
		m.inbox = tui.NewInbox(m.emails)
		m.current = m.inbox
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()
	}

	m.current, cmd = m.current.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *mainModel) View() string {
	return m.current.View()
}

func markdownToHTML(md []byte) []byte {
	var buf bytes.Buffer
	p := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	if err := p.Convert(md, &buf); err != nil {
		return md
	}
	return buf.Bytes()
}

func sendEmail(cfg *config.Config, msg tui.SendEmailMsg) tea.Cmd {
	return func() tea.Msg {
		recipients := []string{msg.To}
		body := msg.Body
		images := make(map[string][]byte)
		attachments := make(map[string][]byte)

		re := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
		matches := re.FindAllStringSubmatch(body, -1)

		for _, match := range matches {
			imgPath := match[1]
			imgData, err := os.ReadFile(imgPath)
			if err != nil {
				log.Printf("Could not read image file %s: %v", imgPath, err)
				continue
			}
			cid := fmt.Sprintf("%s%s@%s", uuid.NewString(), filepath.Ext(imgPath), "email-cli")
			images[cid] = []byte(base64.StdEncoding.EncodeToString(imgData))
			body = strings.Replace(body, imgPath, "cid:"+cid, 1)
		}

		htmlBody := markdownToHTML([]byte(body))

		if msg.AttachmentPath != "" {
			fileData, err := os.ReadFile(msg.AttachmentPath)
			if err != nil {
				log.Printf("Could not read attachment file %s: %v", msg.AttachmentPath, err)
			} else {
				_, filename := filepath.Split(msg.AttachmentPath)
				attachments[filename] = fileData
			}
		}

		err := sender.SendEmail(cfg, recipients, msg.Subject, msg.Body, string(htmlBody), images, attachments, msg.InReplyTo, msg.References)
		if err != nil {
			log.Printf("Failed to send email: %v", err)
			return tui.EmailResultMsg{Err: err}
		}
		time.Sleep(1 * time.Second)
		return tui.EmailResultMsg{}
	}
}

func fetchEmails(cfg *config.Config, limit, offset uint32) tea.Cmd {
	return func() tea.Msg {
		emails, err := fetcher.FetchEmails(cfg, limit, offset)
		if err != nil {
			return tui.FetchErr(err)
		}
		if offset == 0 {
			return tui.EmailsFetchedMsg{Emails: emails}
		}
		return tui.EmailsAppendedMsg{Emails: emails}
	}
}

func deleteEmailCmd(cfg *config.Config, uid uint32) tea.Cmd {
	return func() tea.Msg {
		err := fetcher.DeleteEmail(cfg, uid)
		return tui.EmailActionDoneMsg{UID: uid, Err: err}
	}
}

func archiveEmailCmd(cfg *config.Config, uid uint32) tea.Cmd {
	return func() tea.Msg {
		err := fetcher.ArchiveEmail(cfg, uid)
		return tui.EmailActionDoneMsg{UID: uid, Err: err}
	}
}

func main() {
	cfg, err := config.LoadConfig()
	var initialModel *mainModel
	if err != nil {
		initialModel = newInitialModel(nil)
	} else {
		initialModel = newInitialModel(cfg)
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}