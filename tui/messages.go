package tui

import "github.com/andrinoff/email-cli/fetcher"

type ViewEmailMsg struct {
	Index int
}

type SendEmailMsg struct {
	To             string
	Subject        string
	Body           string
	AttachmentPath string
	InReplyTo      string
	References     []string
}

type Credentials struct {
	Provider string
	Name     string
	Email    string
	Password string
}

type ChooseServiceMsg struct {
	Service string
}

type EmailResultMsg struct {
	Err error
}

type ClearStatusMsg struct{}

type EmailsFetchedMsg struct {
	Emails []fetcher.Email
}

type FetchErr error

type GoToInboxMsg struct{}

type GoToSendMsg struct {
	To      string
	Subject string
	Body    string
}

type GoToSettingsMsg struct{}

type FetchMoreEmailsMsg struct {
	Offset uint32
}

type FetchingMoreEmailsMsg struct{}

type EmailsAppendedMsg struct {
	Emails []fetcher.Email
}

type ReplyToEmailMsg struct {
	Email fetcher.Email
}

type SetComposerCursorToStartMsg struct{}

type GoToFilePickerMsg struct{}

type FileSelectedMsg struct {
	Path string
}

type CancelFilePickerMsg struct{}

type DeleteEmailMsg struct {
	UID uint32
}

type ArchiveEmailMsg struct {
	UID uint32
}

type EmailActionDoneMsg struct {
	UID uint32
	Err error
}

type GoToChoiceMenuMsg struct{}

type DownloadAttachmentMsg struct {
	Filename string
	Data     []byte
}

type AttachmentDownloadedMsg struct {
	Path string
	Err  error
}

type RestoreViewMsg struct{}

type BackToInboxMsg struct{}

// --- Draft Messages ---

// DiscardDraftMsg signals that a draft should be cached.
type DiscardDraftMsg struct {
	ComposerState *Composer
}

// RestoreDraftMsg signals that the cached draft should be restored.
type RestoreDraftMsg struct{}
