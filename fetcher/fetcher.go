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
	fetchItems := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, imap.FetchItem("BODY[]")}
	go func() {
		done <- c.Fetch(seqset, fetchItems, messages)
	}()

	var emails []Email
	for msg := range messages {
		if msg == nil {
			continue
		}

		bodyLiteral := msg.GetBody(&imap.BodySectionName{})
		if bodyLiteral == nil {
			log.Println("Could not get message body")
			continue
		}

		mr, err := mail.CreateReader(bodyLiteral)
		if err != nil {
			log.Printf("Error creating mail reader: %v", err)
			continue
		}

		header := mr.Header
		fromAddrs, _ := header.AddressList("From")
		toAddrs, _ := header.AddressList("To")
		subject := decodeHeader(header.Get("Subject"))
		date, _ := header.Date()
		messageID := header.Get("Message-ID")
		references := header.Get("References")

		var fromAddr string
		if len(fromAddrs) > 0 {
			fromAddr = fromAddrs[0].Address
		}

		var toAddrList []string
		for _, addr := range toAddrs {
			toAddrList = append(toAddrList, addr.Address)
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

			// Correctly parse Content-Disposition
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
						continue // Skip to next part
					}
				}
			}

			// Process body part if not an attachment
			mediaType, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
			if (mediaType == "text/plain" || mediaType == "text/html") && body == "" {
				decodedPart, decodeErr := decodePart(p.Body, p.Header)
				if decodeErr == nil {
					body = decodedPart
				}
			}
		}

		emails = append(emails, Email{
			UID:         msg.Uid,
			From:        fromAddr,
			To:          toAddrList,
			Subject:     subject,
			Body:        body,
			Date:        date,
			MessageID:   messageID,
			References:  strings.Fields(references),
			Attachments: attachments,
		})
	}

	if err := <-done; err != nil {
		return nil, err
	}

	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
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
