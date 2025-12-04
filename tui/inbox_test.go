package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/floatpane/matcha/fetcher"
)

// TestInboxUpdate verifies the state transitions in the inbox view.
func TestInboxUpdate(t *testing.T) {
	// Create a sample list of emails.
	sampleEmails := []fetcher.Email{
		{From: "a@example.com", Subject: "Email 1", Date: time.Now()},
		{From: "b@example.com", Subject: "Email 2", Date: time.Now().Add(-time.Hour)},
		{From: "c@example.com", Subject: "Email 3", Date: time.Now().Add(-2 * time.Hour)},
	}

	inbox := NewInbox(sampleEmails)

	t.Run("Select email to view", func(t *testing.T) {
		// By default, the first item is selected (index 0).
		// Move down to the second item (index 1).
		inbox.list, _ = inbox.list.Update(tea.KeyMsg{Type: tea.KeyDown})

		// Simulate pressing Enter to view the selected email.
		_, cmd := inbox.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatal("Expected a command, but got nil.")
		}

		// Check the resulting message.
		msg := cmd()
		viewMsg, ok := msg.(ViewEmailMsg)
		if !ok {
			t.Fatalf("Expected a ViewEmailMsg, but got %T", msg)
		}

		// The index should match the selected item in the list.
		expectedIndex := 1
		if viewMsg.Index != expectedIndex {
			t.Errorf("Expected index %d, but got %d", expectedIndex, viewMsg.Index)
		}
	})
}
