package sender

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"path/filepath"
	"strings"
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

// SendEmail constructs a multipart message with plain text, HTML, and embedded images.
func SendEmail(cfg *config.Config, to []string, subject, plainBody, htmlBody string, images map[string][]byte) error {
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

	// Main message buffer
	var msg bytes.Buffer
	mainWriter := multipart.NewWriter(&msg)

	// Set top-level headers for multipart/related
	headers := map[string]string{
		"From":         fromHeader,
		"To":           to[0],
		"Subject":      subject,
		"Date":         time.Now().Format(time.RFC1123Z),
		"Message-ID":   generateMessageID(cfg.Email),
		"Content-Type": "multipart/related; boundary=" + mainWriter.Boundary(),
	}
	for k, v := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", k, v)
	}
	fmt.Fprintf(&msg, "\r\n") // End of headers

	// Create the multipart/alternative part as a nested part
	altHeader := textproto.MIMEHeader{}
	altBoundary := "alt-" + mainWriter.Boundary()
	altHeader.Set("Content-Type", "multipart/alternative; boundary="+altBoundary)
	altPartWriter, err := mainWriter.CreatePart(altHeader)
	if err != nil {
		return err
	}

	altWriter := multipart.NewWriter(altPartWriter)
	altWriter.SetBoundary(altBoundary)

	// Create plain text part inside multipart/alternative
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
	textPart, err := altWriter.CreatePart(textHeader)
	if err != nil {
		return err
	}
	fmt.Fprint(textPart, plainBody)

	// Create HTML part inside multipart/alternative
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlPart, err := altWriter.CreatePart(htmlHeader)
	if err != nil {
		return err
	}
	fmt.Fprint(htmlPart, htmlBody)

	altWriter.Close()

	// Attach images to the main multipart/related part
	for cid, data := range images {
		ext := filepath.Ext(strings.Split(cid, "@")[0]) // Extract extension from CID
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		imgHeader := textproto.MIMEHeader{}
		imgHeader.Set("Content-Type", mimeType)
		imgHeader.Set("Content-Transfer-Encoding", "base64")
		imgHeader.Set("Content-ID", "<"+cid+">")
		imgHeader.Set("Content-Disposition", "inline; filename=\""+cid+"\"")

		imgPart, err := mainWriter.CreatePart(imgHeader)
		if err != nil {
			return err
		}
		decodedData, err := base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return err
		}
		imgPart.Write(decodedData)
	}

	mainWriter.Close()

	addr := fmt.Sprintf("%s:%d", smtpServer, smtpPort)
	return smtp.SendMail(addr, auth, cfg.Email, to, msg.Bytes())
}