package fetcher

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/andrinoff/email-cli/config"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Email struct holds the essential information for a single email.
type Email struct {
	From    string
	Subject string
	Body    string
	Date    time.Time
}

// FetchEmails connects to the email provider's IMAP server and retrieves the latest emails.
func FetchEmails(cfg *config.Config) ([]Email, error) {
	var imapHost, serverName string
	switch cfg.ServiceProvider {
	case "gmail":
		imapHost = "imap.gmail.com:993"
		serverName = "imap.gmail.com"
	case "icloud":
		imapHost = "imap.mail.me.com:993"
		serverName = "imap.mail.me.com"
	default:
		return nil, fmt.Errorf("unsupported service provider: %s", cfg.ServiceProvider)
	}

	// Set up client options
	options := imapclient.Options{
		TLSConfig: &tls.Config{ServerName: serverName},
	}
	// Dial the IMAP server
	c, err := imapclient.DialTLS(imapHost, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to dial IMAP server: %w", err)
	}
	defer c.Close()

	// Login
	if err := c.Login(cfg.Email, cfg.Password).Wait(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	// Select INBOX to get message count
	selectData, err := c.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}
	numMessages := selectData.NumMessages

	if numMessages == 0 {
		return []Email{}, nil // No messages
	}

	// Determine the sequence set for the last 10 messages
	start := uint32(1)
	if numMessages > 10 {
		start = numMessages - 9
	}
	seqSet := imap.SeqSetNum(start, numMessages)

	// Define what to fetch for each message
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText},
		},
	}

	// Fetch the messages
	cmd := c.Fetch(seqSet, fetchOptions)
	defer cmd.Close()

	var emails []Email
	for {
		msg := cmd.Next()
		if msg == nil {
			break // All messages have been processed
		}

		var from, subject, body string
		var date time.Time

		// Get header info from the envelope
		if env := msg.Envelope(); env != nil {
			if len(env.From) > 0 {
				from = env.From[0].Address()
			}
			subject = env.Subject
			date = env.Date
		}

		// Read the body content
		if part := msg.BodySection(&imap.FetchItemBodySection{Specifier: imap.PartSpecifierText}); part != nil {
			bodyBytes, err := io.ReadAll(part)
			if err != nil {
				log.Printf("failed to read body part: %v", err)
				continue
			}
			body = string(bodyBytes)
		}

		emails = append(emails, Email{
			From:    from,
			Subject: subject,
			Body:    body,
			Date:    date,
		})
	}
	if err := cmd.Err(); err != nil {
		return nil, fmt.Errorf("FETCH command failed: %w", err)
	}

	// Reverse the slice to show the newest emails first
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}
