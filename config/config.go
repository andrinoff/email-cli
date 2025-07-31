package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config stores the user's email configuration.
type Config struct {
	ServiceProvider string `json:"service_provider"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	Name            string `json:"name"`
}

// configDir returns the path to the configuration directory.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "email-cli"), nil
}

// configFile returns the full path to the configuration file.
func configFile() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// SaveConfig saves the given configuration to the config file.
func SaveConfig(config *Config) error {
	path, err := configFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadConfig loads the configuration from the config file.
func LoadConfig() (*Config, error) {
	path, err := configFile()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// IMAPServer returns the IMAP server address based on the service provider.
// This is used to connect to the email provider's IMAP server.
// It returns an empty string if the service provider is not supported.
func (c *Config) IMAPServer() string {
	switch c.ServiceProvider {
	case "gmail":
		return "imap.gmail.com"
	case "icloud":
		return "imap.mail.me.com"
	// Add other providers here
	default:
		return ""
	}
}
