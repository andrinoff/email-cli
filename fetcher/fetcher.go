package fetcher

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"strings"
	"time"

	"github.com/andrinoff/email-cli/config"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

// Attachment holds data for an email attachment.
type Attachment struct {
	Filename string
	Data     []byte
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

func connect(cfg *config.Config) (*client.Client, error) {
	var imapServer string
	var imapPort int

	switch cfg.ServiceProvider {
	case "gmail":
		imapServer = "imap.gmail.com"
		imapPort = 993
	case "icloud":
		imapServer = "imap.mail.me.com"
		imapPort = 993
	default:
		return nil, fmt.Errorf("unsupported service_provider: %s", cfg.ServiceProvider)
	}

	addr := fmt.Sprintf("%s:%d", imapServer, imapPort)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}

	if err := c.Login(cfg.Email, cfg.Password); err != nil {
		return nil, err
	}

	return c, nil
}

func FetchEmails(cfg *config.Config, limit, offset uint32) ([]Email, error) {
	c, err := connect(cfg)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	mbox, err := c.Select("INBOX", false)
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
	// Only fetch Envelope and UID for the inbox view
	fetchItems := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid}
	go func() {
		done <- c.Fetch(seqset, fetchItems, messages)
	}()

	// Collect messages from the channel into a slice
	var msgs []*imap.Message
	for msg := range messages {
		msgs = append(msgs, msg)
	}

	// Wait for the fetch to complete and check for errors
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
		for _, addr := range msg.Envelope.To {
			toAddrList = append(toAddrList, addr.Address())
		}

		emails = append(emails, Email{
			UID:     msg.Uid,
			From:    fromAddr,
			To:      toAddrList,
			Subject: decodeHeader(msg.Envelope.Subject),
			Date:    msg.Envelope.Date,
		})
	}

	// Reverse the order to show newest first
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}

func FetchEmailBody(cfg *config.Config, uid uint32) (string, []Attachment, error) {
	c, err := connect(cfg)
	if err != nil {
		return "", nil, err
	}
	defer c.Logout()

	if _, err := c.Select("INBOX", false); err != nil {
		return "", nil, err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	fetchItems := []imap.FetchItem{imap.FetchItem("BODY[]")}
	go func() {
		done <- c.UidFetch(seqset, fetchItems, messages)
	}()

	// Wait for the fetch operation to complete first.
	if err := <-done; err != nil {
		return "", nil, err
	}

	// Now that the fetch is complete, check for a message.
	var msg *imap.Message
	select {
	case msg = <-messages:
		if msg == nil {
			return "", nil, fmt.Errorf("fetched a nil message")
		}
	default:
		return "", nil, fmt.Errorf("no message found with UID %d", uid)
	}

	bodyLiteral := msg.GetBody(&imap.BodySectionName{})
	if bodyLiteral == nil {
		return "", nil, fmt.Errorf("could not get message body")
	}

	mr, err := mail.CreateReader(bodyLiteral)
	if err != nil {
		return "", nil, fmt.Errorf("error creating mail reader: %v", err)
	}

	var body string
	var attachments []Attachment
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Error getting next part: %v", err)
			break
		}

		cdHeader := p.Header.Get("Content-Disposition")
		if cdHeader != "" {
			disposition, params, err := mime.ParseMediaType(cdHeader)
			if err == nil && (disposition == "attachment" || disposition == "inline") {
				filename := params["filename"]
				if filename != "" {
					partBody, _ := ioutil.ReadAll(p.Body)
					encoding := p.Header.Get("Content-Transfer-Encoding")
					if strings.ToLower(encoding) == "base64" {
						decoded, decodeErr := base64.StdEncoding.DecodeString(string(partBody))
						if decodeErr == nil {
							partBody = decoded
						}
					}
					attachments = append(attachments, Attachment{Filename: filename, Data: partBody})
					continue
				}
			}
		}

		mediaType, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		if (mediaType == "text/plain" || mediaType == "text/html") && body == "" {
			decodedPart, decodeErr := decodePart(p.Body, p.Header)
			if decodeErr == nil {
				body = decodedPart
			}
		}
	}

	return body, attachments, nil
}

func moveEmail(cfg *config.Config, uid uint32, destMailbox string) error {
	c, err := connect(cfg)
	if err != nil {
		return err
	}
	defer c.Logout()

	if _, err := c.Select("INBOX", false); err != nil {
		return err
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	return c.UidMove(seqSet, destMailbox)
}

func DeleteEmail(cfg *config.Config, uid uint32) error {
	c, err := connect(cfg)
	if err != nil {
		return err
	}
	defer c.Logout()

	if _, err := c.Select("INBOX", false); err != nil {
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

func ArchiveEmail(cfg *config.Config, uid uint32) error {
	var archiveMailbox string
	switch cfg.ServiceProvider {
	case "gmail":
		archiveMailbox = "[Gmail]/All Mail"
	default:
		archiveMailbox = "Archive"
	}
	return moveEmail(cfg, uid, archiveMailbox)
}