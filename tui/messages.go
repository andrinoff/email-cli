package tui

import "github.com/andrinoff/email-cli/fetcher"

// A message to tell the main model to switch to the inbox view.
type GoToInboxMsg struct{}

// A message to tell the main model to switch to the composer view.
type GoToSendMsg struct{}

// A message containing a fetched email to be viewed.
type ViewEmailMsg struct {
	Email fetcher.Email
}

// A message with the content of an email to be sent.
type SendEmailMsg struct {
	To      string
	Subject string
	Body    string
}

// A message to indicate that an email has been successfully sent.
type EmailSentMsg struct{}