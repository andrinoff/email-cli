package sender

import (
	"crypto/rand"
	"fmt"
	"net/smtp"
	"time"

	"github.com/andrinoff/email-cli/config"
)

// generateMessageID creates a unique Message-ID header.
// This is a crucial header for deliverability, as its absence is a spam indicator.
func generateMessageID(from string) string {
	// Generate a random part to ensure uniqueness
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		// Fallback to a less random but still unique value if crypto/rand fails
		return fmt.Sprintf("<%d.%s>", time.Now().UnixNano(), from)
	}
	return fmt.Sprintf("<%x@%s>", buf, from)
}

func SendEmail(cfg *config.Config, to []string, subject, body string) error {
	var smtpServer string
	var smtpPort int

	// Determine the SMTP server based on the service provider in the config.
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

	// Set up authentication information.
	auth := smtp.PlainAuth("", cfg.Email, cfg.Password, smtpServer)

	// Construct the full email message with proper headers.
	headers := map[string]string{
		"From":         cfg.Email,
		"To":           to[0], // Assuming one recipient for the header display
		"Subject":      subject,
		"Message-ID":   generateMessageID(cfg.Email),
		"Content-Type": "text/plain; charset=UTF-8", // Explicitly set content type
	}

	var msg string
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body

	// SMTP server address.
	addr := fmt.Sprintf("%s:%d", smtpServer, smtpPort)

	// Send the email.
	err := smtp.SendMail(addr, auth, cfg.Email, to, []byte(msg))
	return err
}
