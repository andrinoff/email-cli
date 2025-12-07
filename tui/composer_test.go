package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/floatpane/matcha/config"
)

// TestComposerUpdate verifies the state transitions in the email composer.
func TestComposerUpdate(t *testing.T) {
	// Initialize a new composer with accounts.
	accounts := []config.Account{
		{ID: "account-1", Email: "test@example.com", Name: "Test User"},
	}
	composer := NewComposerWithAccounts(accounts, "account-1", "", "", "")

	t.Run("Focus cycling", func(t *testing.T) {
		// Initial focus is on the 'To' input (index 1, since From is 0).
		// But NewComposer starts focus at focusTo which is 1.
		if composer.focusIndex != focusTo {
			t.Errorf("Initial focusIndex should be %d (focusTo), got %d", focusTo, composer.focusIndex)
		}

		// Simulate pressing Tab to move to the 'Subject' field.
		model, _ := composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != focusSubject {
			t.Errorf("After one Tab, focusIndex should be %d (focusSubject), got %d", focusSubject, composer.focusIndex)
		}

		// Simulate pressing Tab again to move to the 'Body' field.
		model, _ = composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != focusBody {
			t.Errorf("After two Tabs, focusIndex should be %d (focusBody), got %d", focusBody, composer.focusIndex)
		}

		// Simulate pressing Tab again to move to the 'Attachment' field.
		model, _ = composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != focusAttachment {
			t.Errorf("After three Tabs, focusIndex should be %d (focusAttachment), got %d", focusAttachment, composer.focusIndex)
		}

		// Simulate pressing Tab again to move to the 'Send' button.
		model, _ = composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != focusSend {
			t.Errorf("After four Tabs, focusIndex should be %d (focusSend), got %d", focusSend, composer.focusIndex)
		}

		// Simulate one more Tab to wrap around to the 'From' field.
		model, _ = composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != focusFrom {
			t.Errorf("After five Tabs, focusIndex should wrap to %d (focusFrom), got %d", focusFrom, composer.focusIndex)
		}
	})

	t.Run("Send email message", func(t *testing.T) {
		// Re-initialize composer for this test
		composer = NewComposerWithAccounts(accounts, "account-1", "", "", "")

		// Set values for the email fields.
		composer.toInput.SetValue("recipient@example.com")
		composer.subjectInput.SetValue("Test Subject")
		composer.bodyInput.SetValue("This is the body.")
		// Set focus to the Send button.
		composer.focusIndex = focusSend

		// Simulate pressing Enter to send the email.
		_, cmd := composer.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatal("Expected a command to be returned, but got nil.")
		}

		// Execute the command and check the resulting message.
		msg := cmd()
		sendMsg, ok := msg.(SendEmailMsg)
		if !ok {
			t.Fatalf("Expected a SendEmailMsg, but got %T", msg)
		}

		// Verify the content of the message.
		if sendMsg.To != "recipient@example.com" {
			t.Errorf("Expected To 'recipient@example.com', got %q", sendMsg.To)
		}
		if sendMsg.Subject != "Test Subject" {
			t.Errorf("Expected Subject 'Test Subject', got %q", sendMsg.Subject)
		}
		if sendMsg.Body != "This is the body." {
			t.Errorf("Expected Body 'This is the body.', got %q", sendMsg.Body)
		}
		if sendMsg.AccountID != "account-1" {
			t.Errorf("Expected AccountID 'account-1', got %q", sendMsg.AccountID)
		}
	})

	t.Run("Account picker with multiple accounts", func(t *testing.T) {
		multiAccounts := []config.Account{
			{ID: "account-1", Email: "test1@example.com", Name: "User 1"},
			{ID: "account-2", Email: "test2@example.com", Name: "User 2"},
		}
		multiComposer := NewComposerWithAccounts(multiAccounts, "account-1", "", "", "")

		// Move focus to From field
		multiComposer.focusIndex = focusFrom

		// Press Enter to open account picker
		model, _ := multiComposer.Update(tea.KeyMsg{Type: tea.KeyEnter})
		multiComposer = model.(*Composer)

		if !multiComposer.showAccountPicker {
			t.Error("Expected account picker to be shown")
		}

		// Navigate down to select second account
		model, _ = multiComposer.Update(tea.KeyMsg{Type: tea.KeyDown})
		multiComposer = model.(*Composer)

		if multiComposer.selectedAccountIdx != 1 {
			t.Errorf("Expected selectedAccountIdx to be 1, got %d", multiComposer.selectedAccountIdx)
		}

		// Press Enter to confirm selection
		model, _ = multiComposer.Update(tea.KeyMsg{Type: tea.KeyEnter})
		multiComposer = model.(*Composer)

		if multiComposer.showAccountPicker {
			t.Error("Expected account picker to be closed")
		}

		// Verify the selected account
		if multiComposer.GetSelectedAccountID() != "account-2" {
			t.Errorf("Expected selected account ID 'account-2', got %q", multiComposer.GetSelectedAccountID())
		}
	})

	t.Run("Single account no picker", func(t *testing.T) {
		singleAccounts := []config.Account{
			{ID: "account-1", Email: "test@example.com"},
		}
		singleComposer := NewComposerWithAccounts(singleAccounts, "account-1", "", "", "")

		// Move focus to From field
		singleComposer.focusIndex = focusFrom

		// Press Enter - should not open picker with single account
		model, _ := singleComposer.Update(tea.KeyMsg{Type: tea.KeyEnter})
		singleComposer = model.(*Composer)

		if singleComposer.showAccountPicker {
			t.Error("Account picker should not open with single account")
		}
	})
}

// TestComposerGetFromAddress verifies the from address formatting.
func TestComposerGetFromAddress(t *testing.T) {
	t.Run("With name", func(t *testing.T) {
		accounts := []config.Account{
			{ID: "account-1", Email: "test@example.com", Name: "Test User"},
		}
		composer := NewComposerWithAccounts(accounts, "account-1", "", "", "")

		fromAddr := composer.getFromAddress()
		expected := "Test User <test@example.com>"
		if fromAddr != expected {
			t.Errorf("Expected from address %q, got %q", expected, fromAddr)
		}
	})

	t.Run("Without name", func(t *testing.T) {
		accounts := []config.Account{
			{ID: "account-1", Email: "test@example.com"},
		}
		composer := NewComposerWithAccounts(accounts, "account-1", "", "", "")

		fromAddr := composer.getFromAddress()
		expected := "test@example.com"
		if fromAddr != expected {
			t.Errorf("Expected from address %q, got %q", expected, fromAddr)
		}
	})

	t.Run("No accounts", func(t *testing.T) {
		composer := NewComposer("", "", "", "")

		fromAddr := composer.getFromAddress()
		if fromAddr != "" {
			t.Errorf("Expected empty from address, got %q", fromAddr)
		}
	})
}

// TestComposerSetSelectedAccount verifies account selection.
func TestComposerSetSelectedAccount(t *testing.T) {
	accounts := []config.Account{
		{ID: "account-1", Email: "test1@example.com"},
		{ID: "account-2", Email: "test2@example.com"},
		{ID: "account-3", Email: "test3@example.com"},
	}
	composer := NewComposerWithAccounts(accounts, "account-1", "", "", "")

	composer.SetSelectedAccount("account-3")
	if composer.selectedAccountIdx != 2 {
		t.Errorf("Expected selectedAccountIdx 2, got %d", composer.selectedAccountIdx)
	}
	if composer.GetSelectedAccountID() != "account-3" {
		t.Errorf("Expected selected account ID 'account-3', got %q", composer.GetSelectedAccountID())
	}

	// Test non-existent account (should not change)
	composer.SetSelectedAccount("non-existent")
	if composer.selectedAccountIdx != 2 {
		t.Errorf("Expected selectedAccountIdx to remain 2, got %d", composer.selectedAccountIdx)
	}
}
