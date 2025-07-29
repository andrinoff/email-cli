package tui

import "github.com/andrinoff/email-cli/fetcher"

// A message to view an email.
type ViewEmailMsg struct {
	Index int
}

// A message to indicate that an email has been sent.
type SendEmailMsg struct {
	To         string
	Subject    string
	Body       string
	InReplyTo  string
	References []string
}

// A message to indicate that the user has entered their credentials.
type Credentials struct {
	Provider string
	Name     string
	Email    string
	Password string
}

// A message to indicate that the user has chosen a service.
type ChooseServiceMsg struct {
	Service string
}

// EmailResultMsg is sent after an email sending attempt.
// If Err is not nil, the email failed to send.
type EmailResultMsg struct {
	Err error
}

// ClearStatusMsg is sent to clear the status message from the view.
type ClearStatusMsg struct{}

// A message containing the fetched emails.
type EmailsFetchedMsg struct {
	Emails []fetcher.Email
}

// A message to indicate that an error occurred while fetching emails.
type FetchErr error

// A message to navigate to the inbox view.
type GoToInboxMsg struct{}

// A message to navigate to the composer view.
type GoToSendMsg struct {
	To      string
	Subject string
	Body    string
}

// A message to navigate to the settings view.
type GoToSettingsMsg struct{}

// A message to fetch more emails with a given offset.
type FetchMoreEmailsMsg struct {
	Offset uint32
}

// A message to indicate that the app is fetching more emails.
type FetchingMoreEmailsMsg struct{}

// A message to indicate that new emails have been fetched and should be appended.
type EmailsAppendedMsg struct {
	Emails []fetcher.Email
}

// A message to reply to an email.
type ReplyToEmailMsg struct {
	Email fetcher.Email
}

// A message to set the composer cursor to the start.
type SetComposerCursorToStartMsg struct{}