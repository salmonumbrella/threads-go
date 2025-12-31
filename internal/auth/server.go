package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	threads "github.com/salmonumbrella/threads-go"
)

// OAuthResult contains the result of OAuth authentication
type OAuthResult struct {
	AccessToken string
	UserID      string
	Username    string
	ExpiresAt   time.Time
}

// OAuthServer handles the browser-based OAuth flow
type OAuthServer struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
	result       chan *OAuthResult
	errChan      chan error
	shutdown     chan struct{}
	csrfToken    string
	authCode     string
	mu           sync.Mutex
}

// NewOAuthServer creates a new OAuth server
func NewOAuthServer(clientID, clientSecret, redirectURI string, scopes []string) *OAuthServer {
	// Generate CSRF token
	tokenBytes := make([]byte, 32)
	//nolint:errcheck,gosec // crypto/rand.Read never returns an error on supported systems
	rand.Read(tokenBytes)

	return &OAuthServer{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		scopes:       scopes,
		result:       make(chan *OAuthResult, 1),
		errChan:      make(chan error, 1),
		shutdown:     make(chan struct{}),
		csrfToken:    hex.EncodeToString(tokenBytes),
	}
}

// Start starts the OAuth server and opens the browser
func (s *OAuthServer) Start(ctx context.Context) (*OAuthResult, error) {
	// Parse redirect URI to get port
	u, err := url.Parse(s.redirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect URI: %w", err)
	}

	// Find available port
	listener, err := net.Listen("tcp", u.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}
	defer listener.Close() //nolint:errcheck // Best-effort cleanup

	port := listener.Addr().(*net.TCPAddr).Port
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Update redirect URI with actual port if using dynamic port
	if u.Port() == "0" || u.Port() == "" {
		s.redirectURI = baseURL + u.Path
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/callback", s.handleCallback)
	mux.HandleFunc("/success", s.handleSuccess)

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in background
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	// Build authorization URL
	authURL := s.buildAuthURL()

	// Open browser
	go func() {
		if err := openBrowser(authURL); err != nil {
			slog.Info("failed to open browser, please navigate manually", "url", authURL)
			fmt.Printf("\nOpen this URL in your browser:\n%s\n\n", authURL)
		}
	}()

	// Wait for result or context cancellation
	select {
	case result := <-s.result:
		//nolint:errcheck,gosec // Shutdown errors are not actionable here
		server.Shutdown(context.Background())
		return result, nil
	case err := <-s.errChan:
		//nolint:errcheck,gosec // Shutdown errors are not actionable here
		server.Shutdown(context.Background())
		return nil, err
	case <-ctx.Done():
		//nolint:errcheck,gosec // Shutdown errors are not actionable here
		server.Shutdown(context.Background())
		return nil, ctx.Err()
	case <-s.shutdown:
		//nolint:errcheck,gosec // Shutdown errors are not actionable here
		server.Shutdown(context.Background())
		return nil, fmt.Errorf("authentication cancelled")
	}
}

func (s *OAuthServer) buildAuthURL() string {
	params := url.Values{
		"client_id":     {s.clientID},
		"redirect_uri":  {s.redirectURI},
		"scope":         {strings.Join(s.scopes, ",")},
		"response_type": {"code"},
		"state":         {s.csrfToken},
	}
	return fmt.Sprintf("https://www.threads.net/oauth/authorize?%s", params.Encode())
}

func (s *OAuthServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Redirect to Threads authorization
	http.Redirect(w, r, s.buildAuthURL(), http.StatusTemporaryRedirect)
}

func (s *OAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state/CSRF token
	state := r.URL.Query().Get("state")
	if subtle.ConstantTimeCompare([]byte(state), []byte(s.csrfToken)) != 1 {
		http.Error(w, "Invalid state parameter", http.StatusForbidden)
		s.errChan <- fmt.Errorf("CSRF validation failed")
		return
	}

	// Check for error
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		errDesc := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("Authorization failed: %s", errDesc), http.StatusBadRequest)
		s.errChan <- fmt.Errorf("authorization denied: %s - %s", errCode, errDesc)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		s.errChan <- fmt.Errorf("missing authorization code")
		return
	}

	s.mu.Lock()
	s.authCode = code
	s.mu.Unlock()

	// Exchange code for token
	go func() {
		result, err := s.exchangeCodeForToken(code)
		if err != nil {
			s.errChan <- err
			return
		}
		s.result <- result
	}()

	// Show success page
	http.Redirect(w, r, "/success", http.StatusTemporaryRedirect)
}

func (s *OAuthServer) exchangeCodeForToken(code string) (*OAuthResult, error) {
	config := &threads.Config{
		ClientID:     s.clientID,
		ClientSecret: s.clientSecret,
		RedirectURI:  s.redirectURI,
		Scopes:       s.scopes,
	}

	client, err := threads.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Exchange code for token
	if errExchange := client.ExchangeCodeForToken(ctx, code); errExchange != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", errExchange)
	}

	// Convert to long-lived token
	if errLongLived := client.GetLongLivedToken(ctx); errLongLived != nil {
		// Non-fatal - we can continue with short-lived token
		slog.Warn("failed to get long-lived token, using short-lived", "error", errLongLived)
	}

	// Get user info
	user, err := client.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	tokenInfo := client.GetTokenInfo()

	return &OAuthResult{
		AccessToken: tokenInfo.AccessToken,
		UserID:      tokenInfo.UserID,
		Username:    user.Username,
		ExpiresAt:   tokenInfo.ExpiresAt,
	}, nil
}

func (s *OAuthServer) handleSuccess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'")

	tmpl := template.Must(template.New("success").Parse(successTemplate))
	//nolint:errcheck,gosec // Best-effort template render to browser
	tmpl.Execute(w, nil)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

const successTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Threads CLI - Authentication Successful</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .card {
            background: white;
            border-radius: 16px;
            padding: 48px;
            text-align: center;
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
            max-width: 400px;
        }
        .icon {
            width: 64px;
            height: 64px;
            background: #22c55e;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 24px;
        }
        .icon svg {
            width: 32px;
            height: 32px;
            color: white;
        }
        h1 {
            font-size: 24px;
            font-weight: 600;
            color: #1f2937;
            margin-bottom: 12px;
        }
        p {
            color: #6b7280;
            line-height: 1.6;
        }
        .hint {
            margin-top: 24px;
            padding: 16px;
            background: #f3f4f6;
            border-radius: 8px;
            font-size: 14px;
        }
        code {
            background: #e5e7eb;
            padding: 2px 6px;
            border-radius: 4px;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
            </svg>
        </div>
        <h1>Authentication Successful!</h1>
        <p>You've successfully authenticated with Threads.</p>
        <p style="margin-top: 8px;">You can close this window and return to your terminal.</p>
        <div class="hint">
            <p>Get started with <code>threads me</code> or <code>threads posts list</code></p>
        </div>
    </div>
</body>
</html>`
