package sender

import (
	"fmt"
	"net/smtp"

	"github.com/andrinoff/email-cli/config" // Import the local config package
)

// SendEmail sends an email using the provided configuration and content.
func SendEmail(cfg *config.Config, to []string, subject, body string) error {
	var smtpHost, smtpPort string

	switch cfg.ServiceProvider {
	case "gmail":
		smtpHost = "smtp.gmail.com"
		smtpPort = "587"
	case "icloud":
		smtpHost = "smtp.mail.me.com"
		smtpPort = "587"
	default:
		return fmt.Errorf("unsupported service provider: %s", cfg.ServiceProvider)
	}

	auth := smtp.PlainAuth("", cfg.Email, cfg.Password, smtpHost)
	msg := []byte("To: " + to[0] + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body)
	addr := smtpHost + ":" + smtpPort
	return smtp.SendMail(addr, auth, cfg.Email, to, msg)
}