package fetcher

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/quotedprintable"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/floatpane/matcha/config"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

// Attachment holds data for an email attachment.
type Attachment struct {
	Filename string
	PartID   string // Keep PartID to fetch on demand
	Data     []byte
	Encoding string // Store encoding for proper decoding
}

type Email struct {
	UID         uint32
	From        string
	To          []string
	Subject     string
	Body        string
	Date        time.Time
	MessageID   string
	References  []string
	Attachments []Attachment
	AccountID   string // ID of the account this email belongs to
}

func decodePart(reader io.Reader, header mail.PartHeader) (string, error) {
	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		body, _ := ioutil.ReadAll(reader)
		return string(body), nil
	}

	charset := "utf-8"
	if params["charset"] != "" {
		charset = strings.ToLower(params["charset"])
	}

	encoding, err := ianaindex.IANA.Encoding(charset)
	if err != nil || encoding == nil {
		encoding, _ = ianaindex.IANA.Encoding("utf-8")
	}

	transformReader := transform.NewReader(reader, encoding.NewDecoder())
	decodedBody, err := ioutil.ReadAll(transformReader)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		return "[This is a multipart message]", nil
	}

	return string(decodedBody), nil
}

func decodeHeader(header string) string {
	dec := new(mime.WordDecoder)
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		encoding, err := ianaindex.IANA.Encoding(charset)
		if err != nil {
			return nil, err
		}
		return transform.NewReader(input, encoding.NewDecoder()), nil
	}
	decoded, err := dec.DecodeHeader(header)
	if err != nil {
		return header
	}
	return decoded
}

func connect(account *config.Account) (*client.Client, error) {
	imapServer := account.GetIMAPServer()
	imapPort := account.GetIMAPPort()

	if imapServer == "" {
		return nil, fmt.Errorf("unsupported service_provider: %s", account.ServiceProvider)
	}

	addr := fmt.Sprintf("%s:%d", imapServer, imapPort)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}

	if err := c.Login(account.Email, account.Password); err != nil {
		return nil, err
	}

	return c, nil
}

func getSentMailbox(account *config.Account) string {
	switch account.ServiceProvider {
	case "gmail":
		return "[Gmail]/Sent Mail"
	case "icloud":
		return "Sent Messages"
	default:
		return "Sent"
	}
}

func FetchMailboxEmails(account *config.Account, mailbox string, limit, offset uint32) ([]Email, error) {
	c, err := connect(account)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	mbox, err := c.Select(mailbox, false)
	if err != nil {
		return nil, err
	}

	if mbox.Messages == 0 {
		return []Email{}, nil
	}

	to := mbox.Messages - offset
	from := uint32(1)
	if to > limit {
		from = to - limit + 1
	}

	if to < 1 {
		return []Email{}, nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, limit)
	done := make(chan error, 1)
	fetchItems := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid}
	go func() {
		done <- c.Fetch(seqset, fetchItems, messages)
	}()

	var msgs []*imap.Message
	for msg := range messages {
		msgs = append(msgs, msg)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	var emails []Email
	for _, msg := range msgs {
		if msg == nil || msg.Envelope == nil {
			continue
		}

		var fromAddr string
		if len(msg.Envelope.From) > 0 {
			fromAddr = msg.Envelope.From[0].Address()
		}

		var toAddrList []string
		// Build recipient list from To and Cc for matching and display
		for _, addr := range msg.Envelope.To {
			toAddrList = append(toAddrList, addr.Address())
		}
		for _, addr := range msg.Envelope.Cc {
			toAddrList = append(toAddrList, addr.Address())
		}

		// Determine which email to filter on: prefer Account.FetchEmail, fallback to Account.Email
		fetchEmail := strings.ToLower(strings.TrimSpace(account.FetchEmail))
		if fetchEmail == "" {
			fetchEmail = strings.ToLower(strings.TrimSpace(account.Email))
		}

		// Check if any recipient matches the fetchEmail
		matched := false
		for _, r := range toAddrList {
			if strings.EqualFold(strings.TrimSpace(r), fetchEmail) {
				matched = true
				break
			}
		}

		if !matched {
			// Skip messages not addressed to the configured fetch email
			continue
		}

		emails = append(emails, Email{
			UID:       msg.Uid,
			From:      fromAddr,
			To:        toAddrList,
			Subject:   decodeHeader(msg.Envelope.Subject),
			Date:      msg.Envelope.Date,
			AccountID: account.ID,
		})
	}

	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}

func FetchEmailBodyFromMailbox(account *config.Account, mailbox string, uid uint32) (string, []Attachment, error) {
	c, err := connect(account)
	if err != nil {
		return "", nil, err
	}
	defer c.Logout()

	if _, err := c.Select(mailbox, false); err != nil {
		return "", nil, err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	fetchItems := []imap.FetchItem{imap.FetchBodyStructure}
	go func() {
		done <- c.UidFetch(seqset, fetchItems, messages)
	}()

	if err := <-done; err != nil {
		return "", nil, err
	}

	msg := <-messages
	if msg == nil || msg.BodyStructure == nil {
		return "", nil, fmt.Errorf("no message or body structure found with UID %d", uid)
	}

	var textPartID string
	var attachments []Attachment
	var checkPart func(part *imap.BodyStructure, partID string)
	checkPart = func(part *imap.BodyStructure, partID string) {
		// Check for text content
		if part.MIMEType == "text" && (part.MIMESubType == "plain" || part.MIMESubType == "html") && textPartID == "" {
			textPartID = partID
		}

		// Check for attachments using multiple methods
		filename := ""
		// First try the Filename() method which handles various cases
		if fn, err := part.Filename(); err == nil && fn != "" {
			filename = fn
		}
		// Fallback: check DispositionParams
		if filename == "" {
			if fn, ok := part.DispositionParams["filename"]; ok && fn != "" {
				filename = fn
			}
		}
		// Fallback: check Params (for name parameter)
		if filename == "" {
			if fn, ok := part.Params["name"]; ok && fn != "" {
				filename = fn
			}
		}
		// Fallback: check Params for filename
		if filename == "" {
			if fn, ok := part.Params["filename"]; ok && fn != "" {
				filename = fn
			}
		}

		// Add as attachment if it has a disposition or a filename (and not just plain text)
		if filename != "" && (part.Disposition == "attachment" || part.Disposition == "inline" || part.MIMEType != "text") {
			attachments = append(attachments, Attachment{
				Filename: filename,
				PartID:   partID,
				Encoding: part.Encoding, // Store encoding for proper decoding
			})
		}
	}

	var findParts func(*imap.BodyStructure, string)
	findParts = func(bs *imap.BodyStructure, prefix string) {
		// If this is a non-multipart message, check the body structure itself
		if len(bs.Parts) == 0 {
			partID := prefix
			if partID == "" {
				partID = "1"
			}
			checkPart(bs, partID)
			return
		}

		// Iterate through parts
		for i, part := range bs.Parts {
			partID := fmt.Sprintf("%d", i+1)
			if prefix != "" {
				partID = fmt.Sprintf("%s.%d", prefix, i+1)
			}

			checkPart(part, partID)

			if len(part.Parts) > 0 {
				findParts(part, partID)
			}
		}
	}
	findParts(msg.BodyStructure, "")

	var body string
	if textPartID != "" {
		partMessages := make(chan *imap.Message, 1)
		partDone := make(chan error, 1)

		fetchItem := imap.FetchItem(fmt.Sprintf("BODY.PEEK[%s]", textPartID))
		section, err := imap.ParseBodySectionName(fetchItem)
		if err != nil {
			return "", nil, err
		}

		go func() {
			partDone <- c.UidFetch(seqset, []imap.FetchItem{fetchItem}, partMessages)
		}()

		if err := <-partDone; err != nil {
			return "", nil, err
		}

		partMsg := <-partMessages
		if partMsg != nil {
			literal := partMsg.GetBody(section)
			if literal != nil {
				// The new decoding logic starts here
				buf, _ := ioutil.ReadAll(literal)
				mr, err := mail.CreateReader(bytes.NewReader(buf))
				if err != nil {
					body = string(buf)
				} else {
					p, err := mr.NextPart()
					if err != nil {
						body = string(buf)
					} else {
						encoding := p.Header.Get("Content-Transfer-Encoding")
						bodyBytes, _ := ioutil.ReadAll(p.Body)

						switch strings.ToLower(encoding) {
						case "base64":
							decoded, err := base64.StdEncoding.DecodeString(string(bodyBytes))
							if err == nil {
								body = string(decoded)
							} else {
								body = string(bodyBytes)
							}
						case "quoted-printable":
							decoded, err := ioutil.ReadAll(quotedprintable.NewReader(strings.NewReader(string(bodyBytes))))
							if err == nil {
								body = string(decoded)
							} else {
								body = string(bodyBytes)
							}
						default:
							body = string(bodyBytes)
						}
					}
				}
			}
		}
	}

	return body, attachments, nil
}

func FetchAttachmentFromMailbox(account *config.Account, mailbox string, uid uint32, partID string, encoding string) ([]byte, error) {
	c, err := connect(account)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	if _, err := c.Select(mailbox, false); err != nil {
		return nil, err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	fetchItem := imap.FetchItem(fmt.Sprintf("BODY.PEEK[%s]", partID))
	section, err := imap.ParseBodySectionName(fetchItem)
	if err != nil {
		return nil, err
	}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqset, []imap.FetchItem{fetchItem}, messages)
	}()

	if err := <-done; err != nil {
		return nil, err
	}

	msg := <-messages
	if msg == nil {
		return nil, fmt.Errorf("could not fetch attachment")
	}

	literal := msg.GetBody(section)
	if literal == nil {
		return nil, fmt.Errorf("could not get attachment body")
	}

	rawBytes, err := ioutil.ReadAll(literal)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(encoding) {
	case "base64":
		decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(rawBytes))
		decoded, err := ioutil.ReadAll(decoder)
		if err == nil {
			return decoded, nil
		}
		return rawBytes, nil
	case "quoted-printable":
		decoded, err := ioutil.ReadAll(quotedprintable.NewReader(bytes.NewReader(rawBytes)))
		if err == nil {
			return decoded, nil
		}
		return rawBytes, nil
	default:
		return rawBytes, nil
	}
}

func moveEmail(account *config.Account, uid uint32, sourceMailbox, destMailbox string) error {
	c, err := connect(account)
	if err != nil {
		return err
	}
	defer c.Logout()

	if _, err := c.Select(sourceMailbox, false); err != nil {
		return err
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	return c.UidMove(seqSet, destMailbox)
}

func DeleteEmailFromMailbox(account *config.Account, mailbox string, uid uint32) error {
	c, err := connect(account)
	if err != nil {
		return err
	}
	defer c.Logout()

	if _, err := c.Select(mailbox, false); err != nil {
		return err
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}

	if err := c.UidStore(seqSet, item, flags, nil); err != nil {
		return err
	}

	return c.Expunge(nil)
}

func ArchiveEmailFromMailbox(account *config.Account, mailbox string, uid uint32) error {
	var archiveMailbox string
	switch account.ServiceProvider {
	case "gmail":
		archiveMailbox = "[Gmail]/All Mail"
	default:
		archiveMailbox = "Archive"
	}
	return moveEmail(account, uid, mailbox, archiveMailbox)
}

// Convenience wrappers defaulting to INBOX for existing call sites.

func FetchEmails(account *config.Account, limit, offset uint32) ([]Email, error) {
	return FetchMailboxEmails(account, "INBOX", limit, offset)
}

func FetchSentEmails(account *config.Account, limit, offset uint32) ([]Email, error) {
	return FetchMailboxEmails(account, getSentMailbox(account), limit, offset)
}

func FetchEmailBody(account *config.Account, uid uint32) (string, []Attachment, error) {
	return FetchEmailBodyFromMailbox(account, "INBOX", uid)
}

func FetchSentEmailBody(account *config.Account, uid uint32) (string, []Attachment, error) {
	return FetchEmailBodyFromMailbox(account, getSentMailbox(account), uid)
}

func FetchAttachment(account *config.Account, uid uint32, partID string, encoding string) ([]byte, error) {
	return FetchAttachmentFromMailbox(account, "INBOX", uid, partID, encoding)
}

func FetchSentAttachment(account *config.Account, uid uint32, partID string, encoding string) ([]byte, error) {
	return FetchAttachmentFromMailbox(account, getSentMailbox(account), uid, partID, encoding)
}

func DeleteEmail(account *config.Account, uid uint32) error {
	return DeleteEmailFromMailbox(account, "INBOX", uid)
}

func DeleteSentEmail(account *config.Account, uid uint32) error {
	return DeleteEmailFromMailbox(account, getSentMailbox(account), uid)
}

func ArchiveEmail(account *config.Account, uid uint32) error {
	return ArchiveEmailFromMailbox(account, "INBOX", uid)
}

func ArchiveSentEmail(account *config.Account, uid uint32) error {
	return ArchiveEmailFromMailbox(account, getSentMailbox(account), uid)
}
