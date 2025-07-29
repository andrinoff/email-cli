package sender

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"time"

	"github.com/andrinoff/email-cli/config"
)

// generateMessageID creates a unique Message-ID header.
func generateMessageID(from string) string {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		return fmt.Sprintf("<%d.%s>", time.Now().UnixNano(), from)
	}
	return fmt.Sprintf("<%x@%s>", buf, from)
}

// SendEmail now constructs a multipart message with both plain text and HTML parts.
func SendEmail(cfg *config.Config, to []string, subject, plainBody, htmlBody string) error {
	var smtpServer string
	var smtpPort int

	switch cfg.ServiceProvider {
	case "gmail":
		smtpServer = "smtp.gmail.com"
		smtpPort = 587
	case "icloud":
		smtpServer = "smtp.mail.me.com"
		smtpPort = 587
	default:
		return fmt.Errorf("unsupported or missing service_provider in config.json: %s", cfg.ServiceProvider)
	}

	auth := smtp.PlainAuth("", cfg.Email, cfg.Password, smtpServer)

	fromHeader := cfg.Email
	if cfg.Name != "" {
		fromHeader = fmt.Sprintf("%s <%s>", cfg.Name, cfg.Email)
	}

	// Create a new multipart message.
	var msg bytes.Buffer
	mw := multipart.NewWriter(&msg)

	// Set top-level headers.
	headers := map[string]string{
		"From":         fromHeader,
		"To":           to[0],
		"Subject":      subject,
		"Date":         time.Now().Format(time.RFC1123Z),
		"Message-ID":   generateMessageID(cfg.Email),
		"Content-Type": "multipart/alternative; boundary=" + mw.Boundary(),
	}
	for k, v := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", k, v)
	}
	fmt.Fprintf(&msg, "\r\n") // End of headers

	// Create plain text part.
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := mw.CreatePart(textHeader)
	if err != nil {
		return err
	}
	fmt.Fprint(part, plainBody)

	// Create HTML part.
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err = mw.CreatePart(htmlHeader)
	if err != nil {
		return err
	}
	fmt.Fprint(part, htmlBody)

	// Close the multipart writer to write the final boundary.
	if err := mw.Close(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", smtpServer, smtpPort)
	return smtp.SendMail(addr, auth, cfg.Email, to, msg.Bytes())
}