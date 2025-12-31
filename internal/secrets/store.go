package secrets

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/99designs/keyring"
)

const (
	serviceName   = "threads-cli"
	accountPrefix = "account:"
	rotationDays  = 55 // Warn before 60-day expiry
)

// Credentials stores authentication data for a Threads account
type Credentials struct {
	Name         string    `json:"name"`
	AccessToken  string    `json:"-"` // Excluded from JSON for security
	UserID       string    `json:"user_id,omitempty"`
	Username     string    `json:"username,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ClientID     string    `json:"client_id,omitempty"`
	ClientSecret string    `json:"-"` // Excluded from JSON for security
	RedirectURI  string    `json:"redirect_uri,omitempty"`
}

// storedCredentials is the internal format for keyring storage
type storedCredentials struct {
	AccessToken  string    `json:"access_token"`
	UserID       string    `json:"user_id,omitempty"`
	Username     string    `json:"username,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ClientID     string    `json:"client_id,omitempty"`
	ClientSecret string    `json:"client_secret,omitempty"`
	RedirectURI  string    `json:"redirect_uri,omitempty"`
}

// Store provides secure credential storage
type Store interface {
	Set(name string, creds Credentials) error
	Get(name string) (*Credentials, error)
	Delete(name string) error
	List() ([]string, error)
	Keys() ([]string, error)
}

// KeyringStore implements Store using the system keyring
type KeyringStore struct {
	ring           keyring.Keyring
	warnedAccounts map[string]bool
}

// OpenDefault opens the default keyring store
func OpenDefault() (*KeyringStore, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
		// macOS Keychain
		KeychainName:             "login",
		KeychainTrustApplication: true,
		// Linux Secret Service
		LibSecretCollectionName: serviceName,
		// Windows Credential Manager
		WinCredPrefix: serviceName,
		// File-based fallback
		FileDir:          "~/.config/threads-cli/keyring",
		FilePasswordFunc: keyring.TerminalPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}
	return &KeyringStore{
		ring:           ring,
		warnedAccounts: make(map[string]bool),
	}, nil
}

// Set stores credentials for an account
func (s *KeyringStore) Set(name string, creds Credentials) error {
	name = normalizeName(name)
	if name == "" {
		return fmt.Errorf("account name cannot be empty")
	}
	if creds.AccessToken == "" {
		return fmt.Errorf("access token cannot be empty")
	}

	stored := storedCredentials{
		AccessToken:  creds.AccessToken,
		UserID:       creds.UserID,
		Username:     creds.Username,
		ExpiresAt:    creds.ExpiresAt,
		CreatedAt:    creds.CreatedAt,
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		RedirectURI:  creds.RedirectURI,
	}
	if stored.CreatedAt.IsZero() {
		stored.CreatedAt = time.Now()
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	return s.ring.Set(keyring.Item{
		Key:  accountPrefix + name,
		Data: data,
	})
}

// Get retrieves credentials for an account
func (s *KeyringStore) Get(name string) (*Credentials, error) {
	name = normalizeName(name)
	item, err := s.ring.Get(accountPrefix + name)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, fmt.Errorf("account %q not found", name)
		}
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	var stored storedCredentials
	if err := json.Unmarshal(item.Data, &stored); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	creds := &Credentials{
		Name:         name,
		AccessToken:  stored.AccessToken,
		UserID:       stored.UserID,
		Username:     stored.Username,
		ExpiresAt:    stored.ExpiresAt,
		CreatedAt:    stored.CreatedAt,
		ClientID:     stored.ClientID,
		ClientSecret: stored.ClientSecret,
		RedirectURI:  stored.RedirectURI,
	}

	// Warn about expiring tokens (once per session)
	if !stored.ExpiresAt.IsZero() && !s.warnedAccounts[name] {
		daysUntilExpiry := time.Until(stored.ExpiresAt).Hours() / 24
		if daysUntilExpiry < float64(rotationDays-55) && daysUntilExpiry > 0 {
			// Token expiring soon
			s.warnedAccounts[name] = true
		}
	}

	return creds, nil
}

// Delete removes credentials for an account
func (s *KeyringStore) Delete(name string) error {
	name = normalizeName(name)
	return s.ring.Remove(accountPrefix + name)
}

// List returns all account names
func (s *KeyringStore) List() ([]string, error) {
	return s.Keys()
}

// Keys returns all account names
func (s *KeyringStore) Keys() ([]string, error) {
	keys, err := s.ring.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	var accounts []string
	for _, key := range keys {
		if strings.HasPrefix(key, accountPrefix) {
			accounts = append(accounts, strings.TrimPrefix(key, accountPrefix))
		}
	}
	return accounts, nil
}

// IsExpired checks if credentials are expired
func (c *Credentials) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

// IsExpiringSoon checks if credentials expire within the given duration
func (c *Credentials) IsExpiringSoon(within time.Duration) bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Until(c.ExpiresAt) < within
}

// DaysUntilExpiry returns days until token expires
func (c *Credentials) DaysUntilExpiry() float64 {
	if c.ExpiresAt.IsZero() {
		return -1
	}
	return time.Until(c.ExpiresAt).Hours() / 24
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
