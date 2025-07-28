package fetcher

import (
	"crypto/tls"
	"fmt"
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

	options := imapclient.Options{
		TLSConfig: &tls.Config{ServerName: serverName},
	}
	c, err := imapclient.DialTLS(imapHost, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to dial IMAP server: %w", err)
	}
	defer c.Close()

	if err := c.Login(cfg.Email, cfg.Password).Wait(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	selectData, err := c.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}
	numMessages := selectData.NumMessages

	if numMessages == 0 {
		return []Email{}, nil // No messages
	}

	start := uint32(1)
	if numMessages > 10 {
		start = numMessages - 9
	}
	seqSet := imap.SeqSetNum(start, numMessages)

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText},
		},
	}

	cmd := c.Fetch(seqSet, fetchOptions)

	var emails []Email
	for {
		msg := cmd.Next()
		if msg == nil {
			break
		}

		buf, err := msg.Collect()
		if err != nil {
			log.Printf("failed to collect message data: %v", err)
			continue
		}
		
		var from, subject, body string
		var date time.Time

		if buf.Envelope != nil {
			if len(buf.Envelope.From) > 0 {
				from = buf.Envelope.From[0].Addr()
			}
			subject = buf.Envelope.Subject
			date = buf.Envelope.Date
		}
		
		// Corrected: FindBodySection returns []byte, not io.Reader
		if bodySection := buf.FindBodySection(&imap.FetchItemBodySection{Specifier: imap.PartSpecifierText}); bodySection != nil {
			body = string(bodySection) // Directly convert []byte to string
		}

		emails = append(emails, Email{
			From:    from,
			Subject: subject,
			Body:    body,
			Date:    date,
		})
	}

	if err := cmd.Close(); err != nil {
		return nil, fmt.Errorf("FETCH command failed: %w", err)
	}

	// Reverse the slice to show the newest emails first
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}