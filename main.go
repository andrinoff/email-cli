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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/floatpane/matcha/config"
	"github.com/floatpane/matcha/fetcher"
	"github.com/floatpane/matcha/sender"
	"github.com/floatpane/matcha/tui"
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

const (
	initialEmailLimit = 20
	paginationLimit   = 20
)

type mainModel struct {
	current        tea.Model
	previousModel  tea.Model
	cachedComposer *tui.Composer // To cache a discarded draft
	config         *config.Config
	emails         []fetcher.Email
	inbox          *tui.Inbox
	width          int
	height         int
	err            error
}

func newInitialModel(cfg *config.Config) *mainModel {
	// Determine if there is a cached composer to pass to the initial choice view
	hasCache := false
	initialModel := &mainModel{}
	if cfg == nil {
		initialModel.current = tui.NewLogin()
	} else {
		initialModel.current = tui.NewChoice(hasCache)
		initialModel.config = cfg
	}
	return initialModel
}

func (m *mainModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m *mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	m.current, cmd = m.current.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "esc" {
			switch m.current.(type) {
			case *tui.FilePicker:
				return m, func() tea.Msg { return tui.CancelFilePickerMsg{} }
			case *tui.Inbox, *tui.Login:
				m.current = tui.NewChoice(m.cachedComposer != nil)
				return m, m.current.Init()
			}
		}

	case tui.BackToInboxMsg:
		if m.inbox != nil {
			m.current = m.inbox
		} else {
			m.current = tui.NewChoice(m.cachedComposer != nil)
		}
		return m, nil

	case tui.DiscardDraftMsg:
		m.cachedComposer = msg.ComposerState
		m.current = tui.NewChoice(true) // Now there is a cached draft
		return m, m.current.Init()

	case tui.RestoreDraftMsg:
		if m.cachedComposer != nil {
			m.current = m.cachedComposer
			m.cachedComposer.ResetConfirmation()
			m.cachedComposer = nil // Clear cache after restoring
			return m, m.current.Init()
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
		m.current = tui.NewChoice(m.cachedComposer != nil)
		return m, m.current.Init()

	case tui.GoToInboxMsg:
		m.current = tui.NewStatus("Fetching emails...")
		return m, tea.Batch(m.current.Init(), fetchEmails(m.config, initialEmailLimit, 0))

	case tui.EmailsFetchedMsg:
		m.emails = msg.Emails
		m.inbox = tui.NewInbox(m.emails)
		m.current = m.inbox
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.FetchMoreEmailsMsg:
		return m, tea.Batch(
			func() tea.Msg { return tui.FetchingMoreEmailsMsg{} },
			fetchEmails(m.config, paginationLimit, msg.Offset),
		)

	case tui.EmailsAppendedMsg:
		m.emails = append(m.emails, msg.Emails...)
		return m, nil

	case tui.GoToSendMsg:
		// When composing a new email, we discard any previously cached draft.
		m.cachedComposer = nil
		m.current = tui.NewComposer(m.config.Email, msg.To, msg.Subject, msg.Body)
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.GoToSettingsMsg:
		m.current = tui.NewLogin()
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.ViewEmailMsg:
		// Show a status message while fetching the email body
		m.current = tui.NewStatus("Fetching email content...")
		// Pass the index directly to the command
		return m, tea.Batch(m.current.Init(), fetchEmailBodyCmd(m.config, m.emails[msg.Index], msg.Index))

	case tui.EmailBodyFetchedMsg:
		if msg.Err != nil {
			log.Printf("could not fetch email body: %v", msg.Err)
			m.current = m.inbox
			return m, nil
		}
		// Use the index from the message to update the correct email
		m.emails[msg.Index].Body = msg.Body
		m.emails[msg.Index].Attachments = msg.Attachments

		emailView := tui.NewEmailView(m.emails[msg.Index], msg.Index, m.width, m.height)
		m.current = emailView
		return m, m.current.Init()

	case tui.ReplyToEmailMsg:
		to := msg.Email.From
		subject := "Re: " + msg.Email.Subject
		body := fmt.Sprintf("\n\nOn %s, %s wrote:\n> %s", msg.Email.Date.Format("Jan 2, 2006 at 3:04 PM"), msg.Email.From, strings.ReplaceAll(msg.Email.Body, "\n", "\n> "))
		m.current = tui.NewComposer(m.config.Email, to, subject, body)
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.GoToFilePickerMsg:
		m.previousModel = m.current
		wd, _ := os.Getwd()
		m.current = tui.NewFilePicker(wd)
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.FileSelectedMsg, tui.CancelFilePickerMsg:
		if m.previousModel != nil {
			m.current = m.previousModel
			m.previousModel = nil
		}
		m.current, cmd = m.current.Update(msg)
		cmds = append(cmds, cmd)

	case tui.SendEmailMsg:
		m.cachedComposer = nil // Clear cache on successful send
		m.current = tui.NewStatus("Sending email...")
		return m, tea.Batch(m.current.Init(), sendEmail(m.config, msg))

	case tui.EmailResultMsg:
		m.current = tui.NewChoice(m.cachedComposer != nil)
		return m, m.current.Init()

	case tui.DeleteEmailMsg:
		m.previousModel = m.current
		m.current = tui.NewStatus("Deleting email...")
		return m, tea.Batch(m.current.Init(), deleteEmailCmd(m.config, msg.UID))

	case tui.ArchiveEmailMsg:
		m.previousModel = m.current
		m.current = tui.NewStatus("Archiving email...")
		return m, tea.Batch(m.current.Init(), archiveEmailCmd(m.config, msg.UID))

	case tui.EmailActionDoneMsg:
		if msg.Err != nil {
			log.Printf("Action failed: %v", msg.Err)
			m.current = m.inbox
			return m, nil
		}
		var updatedEmails []fetcher.Email
		for _, email := range m.emails {
			if email.UID != msg.UID {
				updatedEmails = append(updatedEmails, email)
			}
		}
		m.emails = updatedEmails
		m.inbox = tui.NewInbox(m.emails)
		m.current = m.inbox
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.DownloadAttachmentMsg:
		m.previousModel = m.current
		m.current = tui.NewStatus(fmt.Sprintf("Downloading %s...", msg.Filename))
		// Use the new FetchAttachment function
		return m, tea.Batch(m.current.Init(), downloadAttachmentCmd(m.config, m.emails[msg.Index].UID, msg))

	case tui.AttachmentDownloadedMsg:
		var statusMsg string
		if msg.Err != nil {
			statusMsg = fmt.Sprintf("Error downloading: %v", msg.Err)
		} else {
			statusMsg = fmt.Sprintf("Saved to %s", msg.Path)
		}
		m.current = tui.NewStatus(statusMsg)
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return tui.RestoreViewMsg{}
		})

	case tui.RestoreViewMsg:
		if m.previousModel != nil {
			m.current = m.previousModel
			m.previousModel = nil
		}
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m *mainModel) View() string {
	return m.current.View()
}

func fetchEmailBodyCmd(cfg *config.Config, email fetcher.Email, index int) tea.Cmd {
	return func() tea.Msg {
		body, attachments, err := fetcher.FetchEmailBody(cfg, email.UID)
		if err != nil {
			return tui.EmailBodyFetchedMsg{Index: index, Err: err}
		}

		// Return the fetched data along with the original index
		return tui.EmailBodyFetchedMsg{
			Index:       index,
			Body:        body,
			Attachments: attachments,
		}
	}
}

func markdownToHTML(md []byte) []byte {
	var buf bytes.Buffer
	p := goldmark.New(goldmark.WithRendererOptions(html.WithUnsafe()))
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
			cid := fmt.Sprintf("%s%s@%s", uuid.NewString(), filepath.Ext(imgPath), "matcha")
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

func downloadAttachmentCmd(cfg *config.Config, uid uint32, msg tui.DownloadAttachmentMsg) tea.Cmd {
	return func() tea.Msg {
		data, err := fetcher.FetchAttachment(cfg, uid, msg.PartID)
		if err != nil {
			return tui.AttachmentDownloadedMsg{Err: err}
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return tui.AttachmentDownloadedMsg{Err: err}
		}
		downloadsPath := filepath.Join(homeDir, "Downloads")
		if _, err := os.Stat(downloadsPath); os.IsNotExist(err) {
			if mkErr := os.MkdirAll(downloadsPath, 0755); mkErr != nil {
				return tui.AttachmentDownloadedMsg{Err: mkErr}
			}
		}
		filePath := filepath.Join(downloadsPath, msg.Filename)
		err = os.WriteFile(filePath, data, 0644)
		return tui.AttachmentDownloadedMsg{Path: filePath, Err: err}
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
