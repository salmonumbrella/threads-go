package cmd

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/config"
	"github.com/salmonumbrella/threads-go/internal/iocontext"
	"github.com/salmonumbrella/threads-go/internal/secrets"
)

type stubStore struct{}

func (s *stubStore) Set(string, secrets.Credentials) error { return errors.New("not implemented") }
func (s *stubStore) Get(string) (*secrets.Credentials, error) {
	return nil, errors.New("not implemented")
}
func (s *stubStore) Delete(string) error     { return errors.New("not implemented") }
func (s *stubStore) List() ([]string, error) { return nil, errors.New("not implemented") }
func (s *stubStore) Keys() ([]string, error) { return nil, errors.New("not implemented") }

func newTestFactory(t *testing.T) *Factory {
	t.Helper()
	io := &iocontext.IO{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		In:     &bytes.Buffer{},
	}
	cfg := config.Default()
	f, err := NewFactory(context.Background(), FactoryOptions{
		IO:     io,
		Config: cfg,
		Store: func() (secrets.Store, error) {
			return &stubStore{}, nil
		},
	})
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}
	return f
}

// Integration test infrastructure

// mockCredentialsStore implements secrets.Store for testing
type mockCredentialsStore struct {
	creds *secrets.Credentials
}

func (m *mockCredentialsStore) Set(string, secrets.Credentials) error { return nil }
func (m *mockCredentialsStore) Get(string) (*secrets.Credentials, error) {
	return m.creds, nil
}
func (m *mockCredentialsStore) Delete(string) error     { return nil }
func (m *mockCredentialsStore) List() ([]string, error) { return []string{"test-user"}, nil }
func (m *mockCredentialsStore) Keys() ([]string, error) { return []string{"test-user"}, nil }

// testCredentials returns mock credentials for testing
func testCredentials() *secrets.Credentials {
	return &secrets.Credentials{
		Name:         "test-user",
		AccessToken:  "test-access-token",
		UserID:       "12345",
		Username:     "testuser",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
	}
}

// createMockClientFactory creates a NewClient function that uses a test server
func createMockClientFactory(serverURL string) func(accessToken string, cfg *threads.Config) (*threads.Client, error) {
	return func(accessToken string, cfg *threads.Config) (*threads.Client, error) {
		// Create config with test server URL - use the captured serverURL
		config := threads.NewConfig()
		if cfg != nil {
			config.ClientID = cfg.ClientID
			config.ClientSecret = cfg.ClientSecret
		}
		if config.ClientID == "" {
			config.ClientID = "test-client-id"
		}
		if config.ClientSecret == "" {
			config.ClientSecret = "test-client-secret"
		}
		config.RedirectURI = "https://example.com/callback"
		config.BaseURL = serverURL // Always use the test server URL

		// Create client without token validation
		client, err := threads.NewClient(config)
		if err != nil {
			return nil, err
		}

		// Set a valid token to bypass authentication
		tokenInfo := &threads.TokenInfo{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(time.Hour),
			UserID:      "12345",
			CreatedAt:   time.Now(),
		}
		if err := client.SetTokenInfo(tokenInfo); err != nil {
			return nil, err
		}

		return client, nil
	}
}

// newIntegrationTestFactory creates a factory configured for integration testing
func newIntegrationTestFactory(t *testing.T, serverURL string) (*Factory, *iocontext.IO) {
	t.Helper()

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	io := &iocontext.IO{
		Out:    outBuf,
		ErrOut: errBuf,
		In:     &bytes.Buffer{},
	}
	cfg := config.Default()

	f, err := NewFactory(context.Background(), FactoryOptions{
		IO:     io,
		Config: cfg,
		Store: func() (secrets.Store, error) {
			return &mockCredentialsStore{creds: testCredentials()}, nil
		},
		NewClient: createMockClientFactory(serverURL),
	})
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	return f, io
}
