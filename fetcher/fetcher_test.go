package fetcher

import (
	"testing"

	"github.com/floatpane/matcha/config"
)

// TestFetchEmails is an integration test that requires a live IMAP server and valid credentials.
// NOTE: This test will be skipped if it cannot load a configuration file,
// making it safe to run in a CI environment without credentials.
// To run this test locally, ensure you have a valid `config.json` file.
func TestFetchEmails(t *testing.T) {
	// Attempt to load the configuration.
	cfg, err := config.LoadConfig()
	if err != nil {
		// If config doesn't exist, skip the test. This is useful for CI environments.
		t.Skipf("Skipping TestFetchEmails: could not load config: %v", err)
	}

	// If the password is a placeholder, skip the test to avoid failed auth attempts.
	if cfg.Password == "" || cfg.Password == "supersecret" {
		t.Skip("Skipping TestFetchEmails: placeholder or empty password found in config.")
	}

	emails, err := FetchEmails(cfg, 10, 10)
	if err != nil {
		t.Fatalf("FetchEmails() failed with error: %v", err)
	}

	if len(emails) == 0 {
		// This is not necessarily a failure, but we can log it.
		t.Log("FetchEmails() returned 0 emails. This might be expected.")
	}

	// Check that the emails are sorted from newest to oldest.
	// Skip emails with zero/invalid dates when checking sort order.
	if len(emails) > 1 {
		var validEmails []Email
		for _, e := range emails {
			if !e.Date.IsZero() {
				validEmails = append(validEmails, e)
			}
		}
		if len(validEmails) > 1 {
			if validEmails[0].Date.Before(validEmails[len(validEmails)-1].Date) {
				t.Error("Emails do not appear to be sorted from newest to oldest.")
			}
		}
	}

	// Check a sample email for expected content.
	for _, email := range emails {
		if email.Subject == "" && email.From == "" {
			t.Errorf("Fetched email has empty subject and from fields: %+v", email)
		}
	}
}
