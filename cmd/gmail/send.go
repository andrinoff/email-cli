package gmail

import (
	"fmt"
	"log"
	"net/smtp"
)

func main(from_email string, password string, to_email []string, subject string, body string) {

	// Requires a Google App Password for your Gmail account.
	// You must generate an App Password.
	//
	// How to generate a Google App Password:
	// 1. Go to your Google Account at https://myaccount.google.com/
	// 2. Select "Security" from the left navigation panel.
	// 3. Under "How you sign in to Google," click on "2-Step Verification" and ensure it is turned ON.
	// 4. Go back to the Security page and click on "App passwords." You may need to sign in again.
	// 5. At the bottom, click "Select app" and choose "Mail."
	// 6. Click "Select device" and choose "Other (Custom name)."
	// 7. Give it a name (e.g., "Go Email Script") and click "Generate."
	// FIXME:  8. Copy the 16-character password and paste it into the config.


	// Gmail SMTP server settings.
	// DO NOT EDIT
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	// --- Email Content ---
	// The message must be formatted according to RFC 822.
	// The "To", "From", and "Subject" headers are separated from the body by a blank line.
	message := []byte("To: " + to_email[0] + "\r\n" +
		"Subject: " + subject + "\r\n" +
		body + "\r\n")

	// --- Authentication ---
	// Create an authentication object.
	auth := smtp.PlainAuth("", from_email, password, smtpHost)

	// --- Sending the Email ---
	// Connect to the SMTP server, authenticate, and send the email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from_email, to_email, message)
	if err != nil {
		log.Fatalf("Failed to send email: %s", err)
	}

	fmt.Println("Email sent successfully!")
}
