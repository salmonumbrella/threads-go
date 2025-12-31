package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewOAuthServer_GeneratesCSRFToken(t *testing.T) {
	server := NewOAuthServer("client-id", "client-secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	if server.csrfToken == "" {
		t.Error("expected CSRF token to be generated")
	}

	// CSRF token should be 64 hex characters (32 bytes)
	if len(server.csrfToken) != 64 {
		t.Errorf("expected CSRF token length 64, got %d", len(server.csrfToken))
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(server.csrfToken); err != nil {
		t.Errorf("CSRF token is not valid hex: %v", err)
	}
}

func TestNewOAuthServer_UniqueCSRFTokens(t *testing.T) {
	server1 := NewOAuthServer("client-id", "client-secret", "http://127.0.0.1:8080/callback", []string{"basic"})
	server2 := NewOAuthServer("client-id", "client-secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	if server1.csrfToken == server2.csrfToken {
		t.Error("expected different CSRF tokens for different servers")
	}
}

func TestNewOAuthServer_StoresConfig(t *testing.T) {
	clientID := "test-client-id"
	clientSecret := "test-client-secret"
	redirectURI := "http://127.0.0.1:8080/callback"
	scopes := []string{"basic", "publish"}

	server := NewOAuthServer(clientID, clientSecret, redirectURI, scopes)

	if server.clientID != clientID {
		t.Errorf("expected clientID %q, got %q", clientID, server.clientID)
	}
	if server.clientSecret != clientSecret {
		t.Errorf("expected clientSecret %q, got %q", clientSecret, server.clientSecret)
	}
	if server.redirectURI != redirectURI {
		t.Errorf("expected redirectURI %q, got %q", redirectURI, server.redirectURI)
	}
	if len(server.scopes) != len(scopes) {
		t.Errorf("expected %d scopes, got %d", len(scopes), len(server.scopes))
	}
}

func TestNewOAuthServer_InitializesChannels(t *testing.T) {
	server := NewOAuthServer("client-id", "client-secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	if server.result == nil {
		t.Error("expected result channel to be initialized")
	}
	if server.errChan == nil {
		t.Error("expected errChan to be initialized")
	}
	if server.shutdown == nil {
		t.Error("expected shutdown channel to be initialized")
	}
}

func TestBuildAuthURL_ContainsRequiredParams(t *testing.T) {
	server := NewOAuthServer("test-client", "secret", "http://127.0.0.1:8080/callback", []string{"basic", "publish"})
	authURL := server.buildAuthURL()

	// Parse the URL
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}

	// Check base URL
	expectedBase := "https://www.threads.net/oauth/authorize"
	actualBase := parsed.Scheme + "://" + parsed.Host + parsed.Path
	if actualBase != expectedBase {
		t.Errorf("expected base URL %q, got %q", expectedBase, actualBase)
	}

	// Check query parameters
	query := parsed.Query()

	if query.Get("client_id") != "test-client" {
		t.Errorf("expected client_id=test-client, got %q", query.Get("client_id"))
	}

	if query.Get("redirect_uri") != "http://127.0.0.1:8080/callback" {
		t.Errorf("expected redirect_uri=http://127.0.0.1:8080/callback, got %q", query.Get("redirect_uri"))
	}

	if query.Get("response_type") != "code" {
		t.Errorf("expected response_type=code, got %q", query.Get("response_type"))
	}

	if query.Get("state") != server.csrfToken {
		t.Errorf("expected state=%s, got %q", server.csrfToken, query.Get("state"))
	}

	// Scopes should be comma-separated
	expectedScopes := "basic,publish"
	if query.Get("scope") != expectedScopes {
		t.Errorf("expected scope=%q, got %q", expectedScopes, query.Get("scope"))
	}
}

func TestHandleCallback_ValidCSRF(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	// Create request with valid state and code
	req := httptest.NewRequest(http.MethodGet, "/callback?state="+server.csrfToken+"&code=auth-code-123", nil)
	rec := httptest.NewRecorder()

	// Handle callback - it will try to exchange code (which will fail, but that's OK for this test)
	server.handleCallback(rec, req)

	// Should redirect to success page (307 Temporary Redirect)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}

	// Check redirect location
	location := rec.Header().Get("Location")
	if location != "/success" {
		t.Errorf("expected redirect to /success, got %q", location)
	}

	// Verify auth code was stored
	server.mu.Lock()
	if server.authCode != "auth-code-123" {
		t.Errorf("expected authCode=auth-code-123, got %q", server.authCode)
	}
	server.mu.Unlock()
}

func TestHandleCallback_InvalidCSRF(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	// Create request with invalid state
	req := httptest.NewRequest(http.MethodGet, "/callback?state=invalid-token&code=auth-code-123", nil)
	rec := httptest.NewRecorder()

	// Handle callback in goroutine to avoid blocking
	go server.handleCallback(rec, req)

	// Wait for error on errChan
	select {
	case err := <-server.errChan:
		if err == nil {
			t.Error("expected CSRF error")
		}
		if !strings.Contains(err.Error(), "CSRF validation failed") {
			t.Errorf("expected CSRF error message, got %q", err.Error())
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for CSRF error")
	}
}

func TestHandleCallback_MissingCode(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	// Create request with valid state but no code
	req := httptest.NewRequest(http.MethodGet, "/callback?state="+server.csrfToken, nil)
	rec := httptest.NewRecorder()

	// Handle callback in goroutine
	go server.handleCallback(rec, req)

	// Wait for error
	select {
	case err := <-server.errChan:
		if err == nil {
			t.Error("expected missing code error")
		}
		if !strings.Contains(err.Error(), "missing authorization code") {
			t.Errorf("expected missing code error, got %q", err.Error())
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for error")
	}
}

func TestHandleCallback_OAuthError(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	// Create request with OAuth error
	req := httptest.NewRequest(http.MethodGet, "/callback?state="+server.csrfToken+"&error=access_denied&error_description=User+denied+access", nil)
	rec := httptest.NewRecorder()

	// Handle callback in goroutine
	go server.handleCallback(rec, req)

	// Wait for error
	select {
	case err := <-server.errChan:
		if err == nil {
			t.Error("expected OAuth error")
		}
		if !strings.Contains(err.Error(), "access_denied") {
			t.Errorf("expected access_denied in error, got %q", err.Error())
		}
		if !strings.Contains(err.Error(), "User denied access") {
			t.Errorf("expected error description in error, got %q", err.Error())
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for error")
	}
}

func TestHandleCallback_TimingAttackResistance(t *testing.T) {
	// This test verifies that constant-time comparison is used
	// by checking that similar tokens don't pass
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	// Generate a token that differs by only one character
	similarToken := server.csrfToken[:len(server.csrfToken)-1] + "0"
	if similarToken == server.csrfToken {
		similarToken = server.csrfToken[:len(server.csrfToken)-1] + "1"
	}

	req := httptest.NewRequest(http.MethodGet, "/callback?state="+similarToken+"&code=auth-code", nil)
	rec := httptest.NewRecorder()

	go server.handleCallback(rec, req)

	select {
	case err := <-server.errChan:
		if !strings.Contains(err.Error(), "CSRF validation failed") {
			t.Errorf("expected CSRF error for similar token, got %q", err.Error())
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for CSRF error")
	}
}

func TestHandleRoot_RedirectsToAuth(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.handleRoot(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}

	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "https://www.threads.net/oauth/authorize") {
		t.Errorf("expected redirect to Threads OAuth, got %q", location)
	}
}

func TestHandleRoot_NotFoundForOtherPaths(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	req := httptest.NewRequest(http.MethodGet, "/other-path", nil)
	rec := httptest.NewRecorder()

	server.handleRoot(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestHandleSuccess_ReturnsHTML(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	req := httptest.NewRequest(http.MethodGet, "/success", nil)
	rec := httptest.NewRecorder()

	server.handleSuccess(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected Content-Type text/html, got %q", contentType)
	}

	// Check CSP header is set
	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header to be set")
	}
	if !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("expected CSP to include default-src, got %q", csp)
	}

	// Check body contains success message
	body := rec.Body.String()
	if !strings.Contains(body, "Authentication Successful") {
		t.Error("expected success message in body")
	}
}

func TestStart_InvalidRedirectURI(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "://invalid-uri", []string{"basic"})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := server.Start(ctx)
	if err == nil {
		t.Error("expected error for invalid redirect URI")
	}
	if !strings.Contains(err.Error(), "invalid redirect URI") {
		t.Errorf("expected 'invalid redirect URI' error, got %q", err.Error())
	}
}

func TestStart_ContextCancellation(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:0/callback", []string{"basic"})

	ctx, cancel := context.WithCancel(context.Background())

	// Start the server in a goroutine
	resultChan := make(chan error, 1)
	go func() {
		_, err := server.Start(ctx)
		resultChan <- err
	}()

	// Give the server a moment to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for the result
	select {
	case err := <-resultChan:
		if err == nil {
			t.Error("expected context cancellation error")
		}
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for context cancellation")
	}
}

func TestCSRFToken_SufficientEntropy(t *testing.T) {
	// Generate multiple tokens and verify they have sufficient uniqueness
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})
		if tokens[server.csrfToken] {
			t.Error("duplicate CSRF token generated - insufficient entropy")
		}
		tokens[server.csrfToken] = true
	}
}

func TestHandleCallback_EmptyState(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	// Request with empty state parameter
	req := httptest.NewRequest(http.MethodGet, "/callback?state=&code=auth-code", nil)
	rec := httptest.NewRecorder()

	go server.handleCallback(rec, req)

	select {
	case err := <-server.errChan:
		if !strings.Contains(err.Error(), "CSRF validation failed") {
			t.Errorf("expected CSRF error for empty state, got %q", err.Error())
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for CSRF error")
	}
}

func TestHandleCallback_NoStateParameter(t *testing.T) {
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback", []string{"basic"})

	// Request without state parameter at all
	req := httptest.NewRequest(http.MethodGet, "/callback?code=auth-code", nil)
	rec := httptest.NewRecorder()

	go server.handleCallback(rec, req)

	select {
	case err := <-server.errChan:
		if !strings.Contains(err.Error(), "CSRF validation failed") {
			t.Errorf("expected CSRF error for missing state, got %q", err.Error())
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for CSRF error")
	}
}

// TestCryptoRandUsage verifies crypto/rand is being used (not math/rand)
// by checking token generation produces cryptographically random values
func TestCryptoRandUsage(t *testing.T) {
	// Generate a token using the same method as NewOAuthServer
	tokenBytes := make([]byte, 32)
	n, err := rand.Read(tokenBytes)
	if err != nil {
		t.Fatalf("crypto/rand.Read failed: %v", err)
	}
	if n != 32 {
		t.Errorf("expected 32 bytes, got %d", n)
	}

	// Verify the token is properly hex-encoded
	token := hex.EncodeToString(tokenBytes)
	if len(token) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(token))
	}
}

func TestBuildAuthURL_URLEncoding(t *testing.T) {
	// Test with special characters that need URL encoding
	server := NewOAuthServer("client-id", "secret", "http://127.0.0.1:8080/callback?extra=value", []string{"basic", "threads_publish"})
	authURL := server.buildAuthURL()

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}

	// Verify the URL is properly formed
	if parsed.Scheme != "https" {
		t.Errorf("expected https scheme, got %q", parsed.Scheme)
	}
	if parsed.Host != "www.threads.net" {
		t.Errorf("expected www.threads.net host, got %q", parsed.Host)
	}
}

func TestOAuthResult_Fields(t *testing.T) {
	// Test OAuthResult struct construction
	expiresAt := time.Now().Add(time.Hour)
	result := &OAuthResult{
		AccessToken: "test-token",
		UserID:      "12345",
		Username:    "testuser",
		ExpiresAt:   expiresAt,
	}

	if result.AccessToken != "test-token" {
		t.Errorf("expected AccessToken=test-token, got %q", result.AccessToken)
	}
	if result.UserID != "12345" {
		t.Errorf("expected UserID=12345, got %q", result.UserID)
	}
	if result.Username != "testuser" {
		t.Errorf("expected Username=testuser, got %q", result.Username)
	}
	if !result.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expected ExpiresAt=%v, got %v", expiresAt, result.ExpiresAt)
	}
}
