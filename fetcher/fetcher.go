package fetcher

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/andrinoff/email-cli/config"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type Email struct {
	From    string
	To      []string
	Subject string
	Body    string
	Date    time.Time
}

func FetchEmails(cfg *config.Config) ([]Email, error) {
	var imapServer string
	var imapPort int

	// Determine the IMAP server based on the service provider in the config.
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

	// Connect to the server
	addr := fmt.Sprintf("%s:%d", imapServer, imapPort)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}
	log.Println("Connected")
	defer c.Logout()

	// Login
	if err := c.Login(cfg.Email, cfg.Password); err != nil {
		return nil, err
	}
	log.Println("Logged in")

	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return nil, err
	}

	// Get the last 10 messages
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > 10 {
		from = mbox.Messages - 9
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchItem("BODY[]")}, messages)
	}()

	var emails []Email
	for msg := range messages {
		r := msg.GetBody(&imap.BodySectionName{})
		if r == nil {
			log.Fatal("Server didn't return message body")
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Fatal(err)
		}

		header := mr.Header
		fromAddrs, _ := header.AddressList("From")
		toAddrs, _ := header.AddressList("To")
		subject, _ := header.Subject()
		date, _ := header.Date()

		var fromAddr string
		if len(fromAddrs) > 0 {
			fromAddr = fromAddrs[0].Address
		}

		var toAddrList []string
		for _, addr := range toAddrs {
			toAddrList = append(toAddrList, addr.Address)
		}

		p, err := mr.NextPart()
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		body, _ := io.ReadAll(p.Body)

		emails = append(emails, Email{
			From:    fromAddr,
			To:      toAddrList,
			Subject: subject,
			Body:    string(body),
			Date:    date,
		})
	}

	if err := <-done; err != nil {
		return nil, err
	}

	// Reverse the order of emails to be from newest to oldest.
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}