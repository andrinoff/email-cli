package tui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestComposerUpdate verifies the state transitions in the email composer.
func TestComposerUpdate(t *testing.T) {
	// Initialize a new composer.
	composer := NewComposer("test@example.com", "", "", "")

	t.Run("Focus cycling", func(t *testing.T) {
		// Initial focus is on the 'To' input (index 0).
		if composer.focusIndex != 0 {
			t.Errorf("Initial focusIndex should be 0, got %d", composer.focusIndex)
		}

		// Simulate pressing Tab to move to the 'Subject' field.
		model, _ := composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != 1 {
			t.Errorf("After one Tab, focusIndex should be 1 (Subject), got %d", composer.focusIndex)
		}

		// Simulate pressing Tab again to move to the 'Body' field.
		model, _ = composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != 2 {
			t.Errorf("After two Tabs, focusIndex should be 2 (Body), got %d", composer.focusIndex)
		}

		// Simulate pressing Tab again to move to the 'Attachment' field.
		model, _ = composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != 3 {
			t.Errorf("After three Tabs, focusIndex should be 3 (Attachment), got %d", composer.focusIndex)
		}

		// Simulate pressing Tab again to move to the 'Send' button.
		model, _ = composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != 4 {
			t.Errorf("After four Tabs, focusIndex should be 4 (Send), got %d", composer.focusIndex)
		}

		// Simulate one more Tab to wrap around to the 'To' field.
		model, _ = composer.Update(tea.KeyMsg{Type: tea.KeyTab})
		composer = model.(*Composer)
		if composer.focusIndex != 0 {
			t.Errorf("After five Tabs, focusIndex should wrap to 0, got %d", composer.focusIndex)
		}
	})

	t.Run("Send email message", func(t *testing.T) {
		// Set values for the email fields.
		composer.toInput.SetValue("recipient@example.com")
		composer.subjectInput.SetValue("Test Subject")
		composer.bodyInput.SetValue("This is the body.")
		// Set focus to the Send button.
		composer.focusIndex = 4

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
		expectedMsg := SendEmailMsg{
			To:             "recipient@example.com",
			Subject:        "Test Subject",
			Body:           "This is the body.",
			AttachmentPath: "", // Expect empty attachment path in this test
		}
		if !reflect.DeepEqual(sendMsg, expectedMsg) {
			t.Errorf("Mismatched SendEmailMsg.\nGot:  %+v\nWant: %+v", sendMsg, expectedMsg)
		}
	})
}
