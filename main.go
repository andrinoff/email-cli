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
	"sync"
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
	cachedComposer *tui.Composer
	config         *config.Config
	emails         []fetcher.Email
	emailsByAcct   map[string][]fetcher.Email
	inbox          *tui.Inbox
	width          int
	height         int
	err            error
}

func newInitialModel(cfg *config.Config) *mainModel {
	hasCache := config.HasEmailCache()
	initialModel := &mainModel{
		emailsByAcct: make(map[string][]fetcher.Email),
	}

	if cfg == nil || !cfg.HasAccounts() {
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
		// Save draft to disk
		if msg.ComposerState != nil {
			draft := msg.ComposerState.ToDraft()
			go func() {
				if err := config.SaveDraft(draft); err != nil {
					log.Printf("Error saving draft: %v", err)
				}
			}()
		}
		m.current = tui.NewChoice(true)
		return m, m.current.Init()

	case tui.RestoreDraftMsg:
		if m.cachedComposer != nil {
			m.current = m.cachedComposer
			m.cachedComposer.ResetConfirmation()
			m.cachedComposer = nil
			return m, m.current.Init()
		}

	case tui.Credentials:
		// Add new account or update existing
		account := config.Account{
			ID:              uuid.New().String(),
			Name:            msg.Name,
			Email:           msg.Email,
			Password:        msg.Password,
			ServiceProvider: msg.Provider,
		}

		if msg.Provider == "custom" {
			account.IMAPServer = msg.IMAPServer
			account.IMAPPort = msg.IMAPPort
			account.SMTPServer = msg.SMTPServer
			account.SMTPPort = msg.SMTPPort
		}

		if m.config == nil {
			m.config = &config.Config{}
		}

		// Check if we're editing an existing account
		if login, ok := m.current.(*tui.Login); ok && login.IsEditMode() {
			// Find and update the existing account
			existingID := login.GetAccountID()
			for i, acc := range m.config.Accounts {
				if acc.ID == existingID {
					account.ID = existingID
					m.config.Accounts[i] = account
					break
				}
			}
		} else {
			m.config.AddAccount(account)
		}

		if err := config.SaveConfig(m.config); err != nil {
			log.Printf("could not save config: %v", err)
			return m, tea.Quit
		}

		m.current = tui.NewChoice(m.cachedComposer != nil)
		return m, m.current.Init()

	case tui.GoToInboxMsg:
		if m.config == nil || !m.config.HasAccounts() {
			m.current = tui.NewLogin()
			return m, m.current.Init()
		}
		// Try to load from cache first for instant display
		if config.HasEmailCache() {
			return m, loadCachedEmails()
		}
		// No cache, fetch normally
		m.current = tui.NewStatus("Fetching emails from all accounts...")
		return m, tea.Batch(m.current.Init(), fetchAllAccountsEmails(m.config))

	case tui.CachedEmailsLoadedMsg:
		if msg.Cache == nil {
			// Cache load failed, fetch normally
			m.current = tui.NewStatus("Fetching emails from all accounts...")
			return m, tea.Batch(m.current.Init(), fetchAllAccountsEmails(m.config))
		}

		// Convert cached emails to fetcher.Email
		var cachedEmails []fetcher.Email
		emailsByAcct := make(map[string][]fetcher.Email)
		for _, cached := range msg.Cache.Emails {
			email := fetcher.Email{
				UID:       cached.UID,
				From:      cached.From,
				To:        cached.To,
				Subject:   cached.Subject,
				Date:      cached.Date,
				MessageID: cached.MessageID,
				AccountID: cached.AccountID,
			}
			cachedEmails = append(cachedEmails, email)
			emailsByAcct[cached.AccountID] = append(emailsByAcct[cached.AccountID], email)
		}

		m.emails = cachedEmails
		m.emailsByAcct = emailsByAcct
		m.inbox = tui.NewInbox(m.emails, m.config.Accounts)
		m.current = m.inbox
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})

		// Start background refresh
		return m, tea.Batch(
			m.current.Init(),
			func() tea.Msg { return tui.RefreshingEmailsMsg{} },
			refreshEmails(m.config),
		)

	case tui.EmailsRefreshedMsg:
		m.emailsByAcct = msg.EmailsByAccount

		// Flatten all emails
		var allEmails []fetcher.Email
		for _, emails := range msg.EmailsByAccount {
			allEmails = append(allEmails, emails...)
		}

		// Sort by date (newest first)
		for i := 0; i < len(allEmails); i++ {
			for j := i + 1; j < len(allEmails); j++ {
				if allEmails[j].Date.After(allEmails[i].Date) {
					allEmails[i], allEmails[j] = allEmails[j], allEmails[i]
				}
			}
		}

		m.emails = allEmails

		// Save to cache
		go saveEmailsToCache(m.emails)

		// Update inbox if it exists
		if m.inbox != nil {
			m.inbox.SetEmails(m.emails, m.config.Accounts)
			// Forward the message to inbox to clear refreshing state
			m.current, _ = m.current.Update(msg)
		}
		return m, nil

	case tui.AllEmailsFetchedMsg:
		m.emailsByAcct = msg.EmailsByAccount

		// Flatten all emails
		var allEmails []fetcher.Email
		for _, emails := range msg.EmailsByAccount {
			allEmails = append(allEmails, emails...)
		}

		// Sort by date (newest first)
		for i := 0; i < len(allEmails); i++ {
			for j := i + 1; j < len(allEmails); j++ {
				if allEmails[j].Date.After(allEmails[i].Date) {
					allEmails[i], allEmails[j] = allEmails[j], allEmails[i]
				}
			}
		}

		m.emails = allEmails

		// Save to cache
		go saveEmailsToCache(m.emails)

		m.inbox = tui.NewInbox(m.emails, m.config.Accounts)
		m.current = m.inbox
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.EmailsFetchedMsg:
		// Single account fetch result
		if m.emailsByAcct == nil {
			m.emailsByAcct = make(map[string][]fetcher.Email)
		}
		m.emailsByAcct[msg.AccountID] = msg.Emails

		// Rebuild all emails
		var allEmails []fetcher.Email
		for _, emails := range m.emailsByAcct {
			allEmails = append(allEmails, emails...)
		}

		// Sort by date
		for i := 0; i < len(allEmails); i++ {
			for j := i + 1; j < len(allEmails); j++ {
				if allEmails[j].Date.After(allEmails[i].Date) {
					allEmails[i], allEmails[j] = allEmails[j], allEmails[i]
				}
			}
		}

		m.emails = allEmails
		if m.inbox == nil {
			m.inbox = tui.NewInbox(m.emails, m.config.Accounts)
		} else {
			m.inbox.SetEmails(m.emails, m.config.Accounts)
		}
		m.current = m.inbox
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.FetchMoreEmailsMsg:
		if msg.AccountID == "" {
			return m, nil // Don't fetch more for "ALL" view
		}
		account := m.config.GetAccountByID(msg.AccountID)
		if account == nil {
			return m, nil
		}
		return m, tea.Batch(
			func() tea.Msg { return tui.FetchingMoreEmailsMsg{} },
			fetchEmails(account, paginationLimit, msg.Offset),
		)

	case tui.EmailsAppendedMsg:
		// Add new emails to the appropriate account
		if m.emailsByAcct == nil {
			m.emailsByAcct = make(map[string][]fetcher.Email)
		}
		m.emailsByAcct[msg.AccountID] = append(m.emailsByAcct[msg.AccountID], msg.Emails...)
		m.emails = append(m.emails, msg.Emails...)
		return m, nil

	case tui.GoToSendMsg:
		m.cachedComposer = nil
		if m.config != nil && len(m.config.Accounts) > 0 {
			firstAccount := m.config.GetFirstAccount()
			composer := tui.NewComposerWithAccounts(m.config.Accounts, firstAccount.ID, msg.To, msg.Subject, msg.Body)
			m.current = composer
		} else {
			m.current = tui.NewComposer("", msg.To, msg.Subject, msg.Body)
		}
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.GoToDraftsMsg:
		drafts := config.GetAllDrafts()
		m.current = tui.NewDrafts(drafts)
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.OpenDraftMsg:
		m.cachedComposer = nil
		var accounts []config.Account
		if m.config != nil {
			accounts = m.config.Accounts
		}
		composer := tui.NewComposerFromDraft(msg.Draft, accounts)
		m.current = composer
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.DeleteSavedDraftMsg:
		go func() {
			if err := config.DeleteDraft(msg.DraftID); err != nil {
				log.Printf("Error deleting draft: %v", err)
			}
		}()
		// Send message back to drafts view
		m.current, cmd = m.current.Update(tui.DraftDeletedMsg{DraftID: msg.DraftID})
		return m, cmd

	case tui.GoToSettingsMsg:
		if m.config != nil {
			m.current = tui.NewSettings(m.config.Accounts)
		} else {
			m.current = tui.NewSettings(nil)
		}
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.GoToAddAccountMsg:
		m.current = tui.NewLogin()
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.GoToChoiceMenuMsg:
		m.current = tui.NewChoice(m.cachedComposer != nil)
		return m, m.current.Init()

	case tui.DeleteAccountMsg:
		if m.config != nil {
			m.config.RemoveAccount(msg.AccountID)
			if err := config.SaveConfig(m.config); err != nil {
				log.Printf("could not save config: %v", err)
			}
			// Remove emails for this account
			delete(m.emailsByAcct, msg.AccountID)

			// Rebuild all emails
			var allEmails []fetcher.Email
			for _, emails := range m.emailsByAcct {
				allEmails = append(allEmails, emails...)
			}
			m.emails = allEmails

			// Go back to settings
			m.current = tui.NewSettings(m.config.Accounts)
			m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, m.current.Init()

	case tui.ViewEmailMsg:
		email := m.getEmailByUIDAndAccount(msg.UID, msg.AccountID)
		if email == nil {
			return m, nil
		}
		m.current = tui.NewStatus("Fetching email content...")
		return m, tea.Batch(m.current.Init(), fetchEmailBodyCmd(m.config, *email, msg.UID, msg.AccountID))

	case tui.EmailBodyFetchedMsg:
		if msg.Err != nil {
			log.Printf("could not fetch email body: %v", msg.Err)
			m.current = m.inbox
			return m, nil
		}

		// Update the email in our stores
		m.updateEmailBodyByUID(msg.UID, msg.AccountID, msg.Body, msg.Attachments)

		email := m.getEmailByUIDAndAccount(msg.UID, msg.AccountID)
		if email == nil {
			m.current = m.inbox
			return m, nil
		}

		// Find the index for the email view (used for display purposes)
		emailIndex := m.getEmailIndex(msg.UID, msg.AccountID)
		emailView := tui.NewEmailView(*email, emailIndex, m.width, m.height)
		m.current = emailView
		return m, m.current.Init()

	case tui.ReplyToEmailMsg:
		to := msg.Email.From
		subject := "Re: " + msg.Email.Subject
		body := fmt.Sprintf("\n\nOn %s, %s wrote:\n> %s", msg.Email.Date.Format("Jan 2, 2006 at 3:04 PM"), msg.Email.From, strings.ReplaceAll(msg.Email.Body, "\n", "\n> "))

		if m.config != nil && len(m.config.Accounts) > 0 {
			// Use the account that received the email
			accountID := msg.Email.AccountID
			if accountID == "" {
				accountID = m.config.GetFirstAccount().ID
			}
			composer := tui.NewComposerWithAccounts(m.config.Accounts, accountID, to, subject, body)
			m.current = composer
		} else {
			m.current = tui.NewComposer("", to, subject, body)
		}
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
		// Get draft ID before clearing composer (if it's a composer)
		var draftID string
		if composer, ok := m.current.(*tui.Composer); ok {
			draftID = composer.GetDraftID()
		}
		m.cachedComposer = nil
		m.current = tui.NewStatus("Sending email...")

		// Get the account to send from
		var account *config.Account
		if msg.AccountID != "" && m.config != nil {
			account = m.config.GetAccountByID(msg.AccountID)
		}
		if account == nil && m.config != nil {
			account = m.config.GetFirstAccount()
		}

		// Save contact and delete draft in background
		go func() {
			// Save the recipient as a contact
			if msg.To != "" {
				// Parse "Name <email>" format
				name, email := parseEmailAddress(msg.To)
				if err := config.AddContact(name, email); err != nil {
					log.Printf("Error saving contact: %v", err)
				}
			}
			// Delete the draft since email is being sent
			if draftID != "" {
				if err := config.DeleteDraft(draftID); err != nil {
					log.Printf("Error deleting draft after send: %v", err)
				}
			}
		}()

		return m, tea.Batch(m.current.Init(), sendEmail(account, msg))

	case tui.EmailResultMsg:
		m.current = tui.NewChoice(m.cachedComposer != nil)
		return m, m.current.Init()

	case tui.DeleteEmailMsg:
		m.previousModel = m.current
		m.current = tui.NewStatus("Deleting email...")

		account := m.config.GetAccountByID(msg.AccountID)
		if account == nil {
			m.current = m.inbox
			return m, nil
		}

		return m, tea.Batch(m.current.Init(), deleteEmailCmd(account, msg.UID, msg.AccountID))

	case tui.ArchiveEmailMsg:
		m.previousModel = m.current
		m.current = tui.NewStatus("Archiving email...")

		account := m.config.GetAccountByID(msg.AccountID)
		if account == nil {
			m.current = m.inbox
			return m, nil
		}

		return m, tea.Batch(m.current.Init(), archiveEmailCmd(account, msg.UID, msg.AccountID))

	case tui.EmailActionDoneMsg:
		if msg.Err != nil {
			log.Printf("Action failed: %v", msg.Err)
			m.current = m.inbox
			return m, nil
		}

		// Remove email from stores
		m.removeEmail(msg.UID, msg.AccountID)

		if m.inbox != nil {
			m.inbox.RemoveEmail(msg.UID, msg.AccountID)
		}
		m.current = m.inbox
		m.current, _ = m.current.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.current.Init()

	case tui.DownloadAttachmentMsg:
		m.previousModel = m.current
		m.current = tui.NewStatus(fmt.Sprintf("Downloading %s...", msg.Filename))

		account := m.config.GetAccountByID(msg.AccountID)
		if account == nil {
			m.current = m.previousModel
			return m, nil
		}

		email := m.getEmailByIndex(msg.Index)
		if email == nil {
			m.current = m.previousModel
			return m, nil
		}

		return m, tea.Batch(m.current.Init(), downloadAttachmentCmd(account, email.UID, msg))

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

func (m *mainModel) getEmailByIndex(index int) *fetcher.Email {
	if index >= 0 && index < len(m.emails) {
		return &m.emails[index]
	}
	return nil
}

func (m *mainModel) getEmailByUIDAndAccount(uid uint32, accountID string) *fetcher.Email {
	for i := range m.emails {
		if m.emails[i].UID == uid && m.emails[i].AccountID == accountID {
			return &m.emails[i]
		}
	}
	return nil
}

func (m *mainModel) getEmailIndex(uid uint32, accountID string) int {
	for i := range m.emails {
		if m.emails[i].UID == uid && m.emails[i].AccountID == accountID {
			return i
		}
	}
	return -1
}

func (m *mainModel) updateEmailBodyByUID(uid uint32, accountID string, body string, attachments []fetcher.Attachment) {
	// Update in all emails list
	for i := range m.emails {
		if m.emails[i].UID == uid && m.emails[i].AccountID == accountID {
			m.emails[i].Body = body
			m.emails[i].Attachments = attachments
			break
		}
	}

	// Also update in account-specific store
	if emails, ok := m.emailsByAcct[accountID]; ok {
		for i := range emails {
			if emails[i].UID == uid {
				emails[i].Body = body
				emails[i].Attachments = attachments
				break
			}
		}
	}
}

func (m *mainModel) removeEmail(uid uint32, accountID string) {
	// Remove from all emails
	var filtered []fetcher.Email
	for _, e := range m.emails {
		if !(e.UID == uid && e.AccountID == accountID) {
			filtered = append(filtered, e)
		}
	}
	m.emails = filtered

	// Remove from account-specific store
	if emails, ok := m.emailsByAcct[accountID]; ok {
		var filteredAcct []fetcher.Email
		for _, e := range emails {
			if e.UID != uid {
				filteredAcct = append(filteredAcct, e)
			}
		}
		m.emailsByAcct[accountID] = filteredAcct
	}
}

func (m *mainModel) View() string {
	return m.current.View()
}

func fetchAllAccountsEmails(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		emailsByAccount := make(map[string][]fetcher.Email)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, account := range cfg.Accounts {
			wg.Add(1)
			go func(acc config.Account) {
				defer wg.Done()
				emails, err := fetcher.FetchEmails(&acc, initialEmailLimit, 0)
				if err != nil {
					log.Printf("Error fetching from %s: %v", acc.Email, err)
					return
				}
				mu.Lock()
				emailsByAccount[acc.ID] = emails
				mu.Unlock()
			}(account)
		}

		wg.Wait()
		return tui.AllEmailsFetchedMsg{EmailsByAccount: emailsByAccount}
	}
}

func fetchEmails(account *config.Account, limit, offset uint32) tea.Cmd {
	return func() tea.Msg {
		emails, err := fetcher.FetchEmails(account, limit, offset)
		if err != nil {
			return tui.FetchErr(err)
		}
		if offset == 0 {
			return tui.EmailsFetchedMsg{Emails: emails, AccountID: account.ID}
		}
		return tui.EmailsAppendedMsg{Emails: emails, AccountID: account.ID}
	}
}

func loadCachedEmails() tea.Cmd {
	return func() tea.Msg {
		cache, err := config.LoadEmailCache()
		if err != nil {
			return tui.CachedEmailsLoadedMsg{Cache: nil}
		}
		return tui.CachedEmailsLoadedMsg{Cache: cache}
	}
}

func refreshEmails(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		emailsByAccount := make(map[string][]fetcher.Email)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, account := range cfg.Accounts {
			wg.Add(1)
			go func(acc config.Account) {
				defer wg.Done()
				emails, err := fetcher.FetchEmails(&acc, initialEmailLimit, 0)
				if err != nil {
					log.Printf("Error fetching from %s: %v", acc.Email, err)
					return
				}
				mu.Lock()
				emailsByAccount[acc.ID] = emails
				mu.Unlock()
			}(account)
		}

		wg.Wait()
		return tui.EmailsRefreshedMsg{EmailsByAccount: emailsByAccount}
	}
}

func saveEmailsToCache(emails []fetcher.Email) {
	var cachedEmails []config.CachedEmail
	for _, email := range emails {
		cachedEmails = append(cachedEmails, config.CachedEmail{
			UID:       email.UID,
			From:      email.From,
			To:        email.To,
			Subject:   email.Subject,
			Date:      email.Date,
			MessageID: email.MessageID,
			AccountID: email.AccountID,
		})

		// Save sender as a contact
		if email.From != "" {
			name, emailAddr := parseEmailAddress(email.From)
			if err := config.AddContact(name, emailAddr); err != nil {
				log.Printf("Error saving contact from email: %v", err)
			}
		}
	}
	cache := &config.EmailCache{Emails: cachedEmails}
	if err := config.SaveEmailCache(cache); err != nil {
		log.Printf("Error saving email cache: %v", err)
	}
}

// parseEmailAddress parses "Name <email>" or just "email" format
func parseEmailAddress(addr string) (name, email string) {
	addr = strings.TrimSpace(addr)
	if idx := strings.Index(addr, "<"); idx != -1 {
		name = strings.TrimSpace(addr[:idx])
		endIdx := strings.Index(addr, ">")
		if endIdx > idx {
			email = strings.TrimSpace(addr[idx+1 : endIdx])
		} else {
			email = strings.TrimSpace(addr[idx+1:])
		}
	} else {
		email = addr
	}
	return name, email
}

func fetchEmailBodyCmd(cfg *config.Config, email fetcher.Email, uid uint32, accountID string) tea.Cmd {
	return func() tea.Msg {
		account := cfg.GetAccountByID(accountID)
		if account == nil {
			return tui.EmailBodyFetchedMsg{UID: uid, AccountID: accountID, Err: fmt.Errorf("account not found")}
		}

		body, attachments, err := fetcher.FetchEmailBody(account, uid)
		if err != nil {
			return tui.EmailBodyFetchedMsg{UID: uid, AccountID: accountID, Err: err}
		}

		return tui.EmailBodyFetchedMsg{
			UID:         uid,
			Body:        body,
			Attachments: attachments,
			AccountID:   accountID,
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

func sendEmail(account *config.Account, msg tui.SendEmailMsg) tea.Cmd {
	return func() tea.Msg {
		if account == nil {
			return tui.EmailResultMsg{Err: fmt.Errorf("no account configured")}
		}

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

		err := sender.SendEmail(account, recipients, msg.Subject, msg.Body, string(htmlBody), images, attachments, msg.InReplyTo, msg.References)
		if err != nil {
			log.Printf("Failed to send email: %v", err)
			return tui.EmailResultMsg{Err: err}
		}
		return tui.EmailResultMsg{}
	}
}

func deleteEmailCmd(account *config.Account, uid uint32, accountID string) tea.Cmd {
	return func() tea.Msg {
		err := fetcher.DeleteEmail(account, uid)
		return tui.EmailActionDoneMsg{UID: uid, AccountID: accountID, Err: err}
	}
}

func archiveEmailCmd(account *config.Account, uid uint32, accountID string) tea.Cmd {
	return func() tea.Msg {
		err := fetcher.ArchiveEmail(account, uid)
		return tui.EmailActionDoneMsg{UID: uid, AccountID: accountID, Err: err}
	}
}

func downloadAttachmentCmd(account *config.Account, uid uint32, msg tui.DownloadAttachmentMsg) tea.Cmd {
	return func() tea.Msg {
		data, err := fetcher.FetchAttachment(account, uid, msg.PartID)
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
