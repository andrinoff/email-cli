package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Account stores the configuration for a single email account.
type Account struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ServiceProvider string `json:"service_provider"` // "gmail", "icloud", or "custom"

	// Custom server settings (used when ServiceProvider is "custom")
	IMAPServer string `json:"imap_server,omitempty"`
	IMAPPort   int    `json:"imap_port,omitempty"`
	SMTPServer string `json:"smtp_server,omitempty"`
	SMTPPort   int    `json:"smtp_port,omitempty"`
}

// Config stores the user's email configuration with multiple accounts.
type Config struct {
	Accounts []Account `json:"accounts"`
}

// GetIMAPServer returns the IMAP server address for the account.
func (a *Account) GetIMAPServer() string {
	switch a.ServiceProvider {
	case "gmail":
		return "imap.gmail.com"
	case "icloud":
		return "imap.mail.me.com"
	case "custom":
		return a.IMAPServer
	default:
		return ""
	}
}

// GetIMAPPort returns the IMAP port for the account.
func (a *Account) GetIMAPPort() int {
	switch a.ServiceProvider {
	case "gmail", "icloud":
		return 993
	case "custom":
		if a.IMAPPort != 0 {
			return a.IMAPPort
		}
		return 993 // Default IMAP SSL port
	default:
		return 993
	}
}

// GetSMTPServer returns the SMTP server address for the account.
func (a *Account) GetSMTPServer() string {
	switch a.ServiceProvider {
	case "gmail":
		return "smtp.gmail.com"
	case "icloud":
		return "smtp.mail.me.com"
	case "custom":
		return a.SMTPServer
	default:
		return ""
	}
}

// GetSMTPPort returns the SMTP port for the account.
func (a *Account) GetSMTPPort() int {
	switch a.ServiceProvider {
	case "gmail", "icloud":
		return 587
	case "custom":
		if a.SMTPPort != 0 {
			return a.SMTPPort
		}
		return 587 // Default SMTP TLS port
	default:
		return 587
	}
}

// configDir returns the path to the configuration directory.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "matcha"), nil
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
		// Try to load legacy single-account config
		var legacyConfig legacyConfigFormat
		if legacyErr := json.Unmarshal(data, &legacyConfig); legacyErr == nil && legacyConfig.Email != "" {
			// Convert legacy config to new format
			config = Config{
				Accounts: []Account{
					{
						ID:              uuid.New().String(),
						Name:            legacyConfig.Name,
						Email:           legacyConfig.Email,
						Password:        legacyConfig.Password,
						ServiceProvider: legacyConfig.ServiceProvider,
					},
				},
			}
			// Save the migrated config
			if saveErr := SaveConfig(&config); saveErr != nil {
				return nil, saveErr
			}
			return &config, nil
		}
		return nil, err
	}
	return &config, nil
}

// legacyConfigFormat represents the old single-account configuration format.
type legacyConfigFormat struct {
	ServiceProvider string `json:"service_provider"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	Name            string `json:"name"`
}

// AddAccount adds a new account to the configuration.
func (c *Config) AddAccount(account Account) {
	if account.ID == "" {
		account.ID = uuid.New().String()
	}
	c.Accounts = append(c.Accounts, account)
}

// RemoveAccount removes an account by its ID.
func (c *Config) RemoveAccount(id string) bool {
	for i, acc := range c.Accounts {
		if acc.ID == id {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
			return true
		}
	}
	return false
}

// GetAccountByID returns an account by its ID.
func (c *Config) GetAccountByID(id string) *Account {
	for i := range c.Accounts {
		if c.Accounts[i].ID == id {
			return &c.Accounts[i]
		}
	}
	return nil
}

// GetAccountByEmail returns an account by its email address.
func (c *Config) GetAccountByEmail(email string) *Account {
	for i := range c.Accounts {
		if c.Accounts[i].Email == email {
			return &c.Accounts[i]
		}
	}
	return nil
}

// HasAccounts returns true if there are any configured accounts.
func (c *Config) HasAccounts() bool {
	return len(c.Accounts) > 0
}

// GetFirstAccount returns the first account or nil if none exist.
func (c *Config) GetFirstAccount() *Account {
	if len(c.Accounts) > 0 {
		return &c.Accounts[0]
	}
	return nil
}
