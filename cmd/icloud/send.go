package icloud

import (
	"fmt"
	"log"
	"net/smtp"
)

func main(to_email []string, password string, subject string, body string, from_email string) {
	// --- Configuration ---
	// Your full iCloud email address.
	

	// Requires an app-specific password for your iCloud account.
	// 
	// How to generate an app-specific password:
	// 1. Go to https://appleid.apple.com
	// 2. Sign in with your Apple ID and password.
	// 3. Under "Sign-In and Security", click on "App-Specific Passwords".
	// 4. Click "Generate an app-specific password" or the "+" button.
	// 5. Enter a name for the password (e.g., "Go Email Script") and click "Create".
	// FIXME: 6. Copy the generated password and paste it in configs. 


	// iCloud SMTP server settings.
	smtpHost := "smtp.mail.me.com"
	smtpPort := "587"

	// --- Email Content ---
	message := []byte(subject + body)

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
