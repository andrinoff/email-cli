package fetcher

import (
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

type Email struct {
	From      string
	To        []string
	Subject   string
	Body      string
	Date      time.Time
	MessageID string
	References []string
}

func decodePart(reader io.Reader, header mail.PartHeader) (string, error) {
	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		body, _ := ioutil.ReadAll(reader)
		return string(body), nil
	}

	charset := "utf-8" // Default charset
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

// FetchEmails now supports pagination with a limit and offset.
func FetchEmails(cfg *config.Config, limit, offset uint32) ([]Email, error) {
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
		return nil, fmt.Errorf("unsupported or missing service_provider in config.json: %s", cfg.ServiceProvider)
	}

	addr := fmt.Sprintf("%s:%d", imapServer, imapPort)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}
	log.Println("Connected")
	defer c.Logout()

	if err := c.Login(cfg.Email, cfg.Password); err != nil {
		return nil, err
	}
	log.Println("Logged in")

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return nil, err
	}

	// Handle pagination logic
	if mbox.Messages == 0 {
		return []Email{}, nil
	}

	to := mbox.Messages - offset
	from := uint32(1)
	if to > limit {
		from = to - limit + 1
	}

	if to < 1 {
		return []Email{}, nil // No more messages to fetch
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, limit)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchItem("BODY[]")}, messages)
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
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Printf("Error getting next part: %v", err)
				break
			}

			mediaType, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
			if mediaType == "text/plain" || mediaType == "text/html" {
				decodedPart, decodeErr := decodePart(p.Body, p.Header)
				if decodeErr == nil {
					body = decodedPart
					break
				}
			}
		}

		emails = append(emails, Email{
			From:       fromAddr,
			To:         toAddrList,
			Subject:    subject,
			Body:       body,
			Date:       date,
			MessageID:  messageID,
			References: strings.Fields(references),
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