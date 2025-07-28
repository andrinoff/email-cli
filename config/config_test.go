package config

import (
	"reflect"
	"testing"
)

// TestSaveAndLoadConfig verifies that the config can be saved to and loaded from a file correctly.
func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for the test to avoid interfering with actual user config.
	tempDir := t.TempDir()

	// Temporarily override the user home directory to our temp directory.
	// This ensures that our config file is written to a predictable, temporary location.
	t.Setenv("HOME", tempDir)

	// Define a sample configuration to save.
	expectedConfig := &Config{
		ServiceProvider: "gmail",
		Email:           "test@example.com",
		Password:        "supersecret",
		Name:            "Test User",
	}

	// Attempt to save the configuration.
	err := SaveConfig(expectedConfig)
	if err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	// Attempt to load the configuration back.
	loadedConfig, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Compare the loaded configuration with the original one.
	// reflect.DeepEqual is used for a deep comparison of the structs.
	if !reflect.DeepEqual(loadedConfig, expectedConfig) {
		t.Errorf("Loaded config does not match expected config.\nGot:  %+v\nWant: %+v", loadedConfig, expectedConfig)
	}
}

// TestIMAPServer tests the logic that determines the IMAP server address.
func TestIMAPServer(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
		want     string
	}{
		{"Gmail", "gmail", "imap.gmail.com"},
		{"iCloud", "icloud", "imap.mail.me.com"},
		{"Unsupported", "yahoo", ""},
		{"Empty", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{ServiceProvider: tc.provider}
			got := cfg.IMAPServer()
			if got != tc.want {
				t.Errorf("IMAPServer() = %q, want %q", got, tc.want)
			}
		})
	}
}
