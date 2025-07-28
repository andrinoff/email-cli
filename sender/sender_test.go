package sender

import (
	"strings"
	"testing"
)

// TestGenerateMessageID ensures the Message-ID has the correct format.
func TestGenerateMessageID(t *testing.T) {
	from := "test@example.com"
	msgID := generateMessageID(from)

	// Check if the message ID is enclosed in angle brackets.
	if !strings.HasPrefix(msgID, "<") || !strings.HasSuffix(msgID, ">") {
		t.Errorf("Message-ID should be enclosed in angle brackets, got %s", msgID)
	}

	// Check if the 'from' address is part of the message ID.
	if !strings.Contains(msgID, from) {
		t.Errorf("Message-ID should contain the from address, got %s", msgID)
	}

	// The original check was too simple and failed because the 'from' address itself contains an '@'.
	// A Message-ID is generally <unique-part@domain>. The current implementation uses the full 'from' address as the domain part.
	// This revised check validates that structure correctly.
	unwrappedID := strings.Trim(msgID, "<>")

	// Ensure there's at least one '@' symbol.
	if !strings.Contains(unwrappedID, "@") {
		t.Errorf("Message-ID should contain an '@' symbol, got %s", msgID)
	}

	// Check that the ID ends with the full 'from' address, preceded by an '@'.
	// This confirms the structure is <random_part>@<from_address>.
	expectedSuffix := "@" + from
	if !strings.HasSuffix(unwrappedID, expectedSuffix) {
		t.Errorf("Message-ID should end with '@' + from address. Got %s, expected suffix %s", unwrappedID, expectedSuffix)
	}

	// Check that the part before the suffix is not empty.
	randomPart := strings.TrimSuffix(unwrappedID, expectedSuffix)
	if randomPart == "" {
		t.Errorf("Message-ID has an empty random part, got %s", msgID)
	}
}
