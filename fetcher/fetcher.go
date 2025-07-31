package fetcher

import (
	"fmt"
	"io"
	"io/ioutil"
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
	PartID   string // Keep PartID to fetch on demand
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
	var findParts func(*imap.BodyStructure, string)
	findParts = func(bs *imap.BodyStructure, prefix string) {
		for i, part := range bs.Parts {
			partID := fmt.Sprintf("%d", i+1)
			if prefix != "" {
				partID = fmt.Sprintf("%s.%d", prefix, i+1)
			}

			if part.MIMEType == "text" && (part.MIMESubType == "plain" || part.MIMESubType == "html") && textPartID == "" {
				textPartID = partID
			}
			if part.Disposition == "attachment" || part.Disposition == "inline" {
				if filename, ok := part.Params["filename"]; ok {
					attachments = append(attachments, Attachment{Filename: filename, PartID: partID})
				}
			}
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
		
		// Correctly create the fetch item as a string for your library version
		fetchItem := imap.FetchItem(fmt.Sprintf("BODY.PEEK[%s]", textPartID))
		section, err := imap.ParseBodySectionName(fetchItem)
		if err != nil {
			return "", nil, err
		}

		go func() {
			// Pass the string-based fetch item
			partDone <- c.UidFetch(seqset, []imap.FetchItem{fetchItem}, partMessages)
		}()

		if err := <-partDone; err != nil {
			return "", nil, err
		}

		partMsg := <-partMessages
		if partMsg != nil {
			// Use the parsed section to get the body from the response
			literal := partMsg.GetBody(section)
			if literal != nil {
				bodyBytes, _ := ioutil.ReadAll(literal)
				body = string(bodyBytes)
			}
		}
	}

	return body, attachments, nil
}

func FetchAttachment(cfg *config.Config, uid uint32, partID string) ([]byte, error) {
	c, err := connect(cfg)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	if _, err := c.Select("INBOX", false); err != nil {
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

	return ioutil.ReadAll(literal)
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