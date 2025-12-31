package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/auth"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
	"github.com/salmonumbrella/threads-go/internal/secrets"
	"github.com/salmonumbrella/threads-go/internal/ui"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  `Authenticate with Threads and manage stored credentials.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Threads via browser",
	Long: `Opens a browser to authenticate with Threads using OAuth 2.0.

After authentication, your credentials are securely stored in the system keychain.
Tokens are automatically converted to long-lived tokens (60 days).`,
	RunE: runAuthLogin,
}

var authTokenCmd = &cobra.Command{
	Use:   "token [access-token]",
	Short: "Authenticate with an existing access token",
	Long: `Use an existing access token to authenticate.

You can provide the token as an argument or via THREADS_ACCESS_TOKEN environment variable.
The CLI will validate the token and store it in your keychain.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAuthToken,
}

var authRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the access token",
	Long:  `Refresh the current access token before it expires.`,
	RunE:  runAuthRefresh,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Display the current authentication status and token expiry information.`,
	RunE:  runAuthStatus,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured accounts",
	RunE:  runAuthList,
}

var authRemoveCmd = &cobra.Command{
	Use:   "remove [account]",
	Short: "Remove a stored account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthRemove,
}

// Auth command flags
var (
	authAccountName  string
	authClientID     string
	authClientSecret string
	authRedirectURI  string
	authScopes       []string
)

func init() {
	// Login flags
	authLoginCmd.Flags().StringVarP(&authAccountName, "name", "n", "default", "Account name for this login")
	authLoginCmd.Flags().StringVar(&authClientID, "client-id", "", "Meta App Client ID (or THREADS_CLIENT_ID)")
	authLoginCmd.Flags().StringVar(&authClientSecret, "client-secret", "", "Meta App Client Secret (or THREADS_CLIENT_SECRET)")
	authLoginCmd.Flags().StringVar(&authRedirectURI, "redirect-uri", "", "OAuth Redirect URI (or THREADS_REDIRECT_URI)")
	authLoginCmd.Flags().StringSliceVar(&authScopes, "scopes", []string{
		"threads_basic",
		"threads_content_publish",
		"threads_manage_insights",
		"threads_manage_replies",
		"threads_read_replies",
	}, "OAuth scopes to request")

	// Token flags
	authTokenCmd.Flags().StringVarP(&authAccountName, "name", "n", "default", "Account name for this token")
	authTokenCmd.Flags().StringVar(&authClientID, "client-id", "", "Meta App Client ID")
	authTokenCmd.Flags().StringVar(&authClientSecret, "client-secret", "", "Meta App Client Secret")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authTokenCmd)
	authCmd.AddCommand(authRefreshCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authRemoveCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	// Get credentials from flags or environment
	clientID := authClientID
	if clientID == "" {
		clientID = os.Getenv("THREADS_CLIENT_ID")
	}
	clientSecret := authClientSecret
	if clientSecret == "" {
		clientSecret = os.Getenv("THREADS_CLIENT_SECRET")
	}
	redirectURI := authRedirectURI
	if redirectURI == "" {
		redirectURI = os.Getenv("THREADS_REDIRECT_URI")
	}

	if clientID == "" || clientSecret == "" {
		return &UserFriendlyError{
			Message:    "Client ID and secret are required for authentication",
			Suggestion: "Set via --client-id and --client-secret flags, or THREADS_CLIENT_ID and THREADS_CLIENT_SECRET environment variables. Get these from the Meta Developer Console",
		}
	}

	// Default redirect URI for CLI OAuth
	if redirectURI == "" {
		redirectURI = "http://127.0.0.1:8585/callback"
	}

	store, err := getStore()
	if err != nil {
		return FormatError(err)
	}

	// Start OAuth server
	server := auth.NewOAuthServer(clientID, clientSecret, redirectURI, authScopes)

	ui.Info("Starting authentication flow...")
	ui.Info("Opening browser for Threads authorization...")

	result, err := server.Start(cmd.Context())
	if err != nil {
		return WrapError("authentication failed", err)
	}

	// Store credentials
	creds := secrets.Credentials{
		Name:         authAccountName,
		AccessToken:  result.AccessToken,
		UserID:       result.UserID,
		Username:     result.Username,
		ExpiresAt:    result.ExpiresAt,
		CreatedAt:    time.Now(),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
	}

	if err := store.Set(authAccountName, creds); err != nil {
		return WrapError("failed to store credentials", err)
	}

	ui.Success("Authentication successful!")
	fmt.Printf("  Account:  %s\n", authAccountName)
	fmt.Printf("  User:     @%s\n", result.Username)
	fmt.Printf("  Expires:  %s (%.0f days)\n", result.ExpiresAt.Format("2006-01-02"), time.Until(result.ExpiresAt).Hours()/24)

	return nil
}

func runAuthToken(cmd *cobra.Command, args []string) error {
	var token string
	if len(args) > 0 {
		token = args[0]
	} else {
		token = os.Getenv("THREADS_ACCESS_TOKEN")
	}

	if token == "" {
		return &UserFriendlyError{
			Message:    "Access token is required",
			Suggestion: "Provide the token as an argument or set the THREADS_ACCESS_TOKEN environment variable",
		}
	}

	// Get optional client credentials for token refresh capability
	clientID := authClientID
	if clientID == "" {
		clientID = os.Getenv("THREADS_CLIENT_ID")
	}
	clientSecret := authClientSecret
	if clientSecret == "" {
		clientSecret = os.Getenv("THREADS_CLIENT_SECRET")
	}

	// Validate token by making API call
	client, err := threads.NewClientWithToken(token, &threads.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return WrapError("failed to create client", err)
	}

	// Debug token to get expiry info
	ctx := cmd.Context()
	debugInfo, err := client.DebugToken(ctx, "")
	if err != nil {
		return WrapError("token validation failed", err)
	}

	if !debugInfo.Data.IsValid {
		return &UserFriendlyError{
			Message:    "The provided token is not valid",
			Suggestion: "Ensure the token is correct and has not expired. Get a new token from the Threads API",
		}
	}

	// Get user info
	user, err := client.GetMe(ctx)
	if err != nil {
		return WrapError("failed to get user info", err)
	}

	store, err := getStore()
	if err != nil {
		return FormatError(err)
	}

	expiresAt := time.Unix(debugInfo.Data.ExpiresAt, 0)
	creds := secrets.Credentials{
		Name:         authAccountName,
		AccessToken:  token,
		UserID:       debugInfo.Data.UserID,
		Username:     user.Username,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	if err := store.Set(authAccountName, creds); err != nil {
		return WrapError("failed to store credentials", err)
	}

	ui.Success("Token stored successfully!")
	fmt.Printf("  Account:  %s\n", authAccountName)
	fmt.Printf("  User:     @%s\n", user.Username)
	fmt.Printf("  Expires:  %s (%.0f days)\n", expiresAt.Format("2006-01-02"), time.Until(expiresAt).Hours()/24)

	return nil
}

func runAuthRefresh(cmd *cobra.Command, args []string) error {
	account, err := requireAccount()
	if err != nil {
		return FormatError(err)
	}

	store, err := getStore()
	if err != nil {
		return FormatError(err)
	}

	creds, err := store.Get(account)
	if err != nil {
		return FormatError(err)
	}

	if creds.ClientSecret == "" {
		return &UserFriendlyError{
			Message:    "Cannot refresh token: client secret not stored",
			Suggestion: "Re-authenticate with 'threads auth login' to enable token refresh",
		}
	}

	client, err := threads.NewClientWithToken(creds.AccessToken, &threads.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
	})
	if err != nil {
		return WrapError("failed to create client", err)
	}

	ctx := cmd.Context()
	if err := client.RefreshToken(ctx); err != nil {
		return WrapError("failed to refresh token", err)
	}

	// Get new token info
	tokenInfo := client.GetTokenInfo()
	creds.AccessToken = tokenInfo.AccessToken
	creds.ExpiresAt = tokenInfo.ExpiresAt

	if err := store.Set(account, *creds); err != nil {
		return WrapError("failed to update stored credentials", err)
	}

	ui.Success("Token refreshed successfully!")
	fmt.Printf("  Account:  %s\n", account)
	fmt.Printf("  Expires:  %s (%.0f days)\n", creds.ExpiresAt.Format("2006-01-02"), time.Until(creds.ExpiresAt).Hours()/24)

	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	account := getAccount()
	if account == "" {
		ui.Warning("No account configured")
		fmt.Println("\nRun 'threads auth login' to authenticate.")
		return nil
	}

	store, err := getStore()
	if err != nil {
		return FormatError(err)
	}

	creds, err := store.Get(account)
	if err != nil {
		return FormatError(err)
	}

	ctx := cmd.Context()

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(map[string]any{
			"account":           account,
			"user_id":           creds.UserID,
			"username":          creds.Username,
			"expires_at":        creds.ExpiresAt,
			"is_expired":        creds.IsExpired(),
			"days_until_expiry": creds.DaysUntilExpiry(),
		}, jqQuery)
	}

	status := "active"
	statusColor := ui.Green
	if creds.IsExpired() {
		status = "expired"
		statusColor = ui.Red
	} else if creds.IsExpiringSoon(7 * 24 * time.Hour) {
		status = "expiring soon"
		statusColor = ui.Yellow
	}

	fmt.Printf("Account:  %s\n", ui.Bold(account))
	fmt.Printf("User:     @%s\n", creds.Username)
	fmt.Printf("User ID:  %s\n", creds.UserID)
	fmt.Printf("Status:   %s\n", ui.Colorize(status, statusColor))

	if !creds.ExpiresAt.IsZero() {
		days := creds.DaysUntilExpiry()
		fmt.Printf("Expires:  %s (%s)\n", creds.ExpiresAt.Format("2006-01-02 15:04"), ui.FormatDuration(days))
	}

	return nil
}

func runAuthList(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return FormatError(err)
	}

	accounts, err := store.List()
	if err != nil {
		return WrapError("failed to list accounts", err)
	}

	if len(accounts) == 0 {
		ui.Info("No accounts configured")
		fmt.Println("\nRun 'threads auth login' to authenticate.")
		return nil
	}

	ctx := cmd.Context()

	if outfmt.IsJSON(ctx) {
		var result []map[string]any
		for _, name := range accounts {
			//nolint:errcheck // Error means account not found, which we handle by checking creds != nil
			creds, _ := store.Get(name)
			if creds != nil {
				result = append(result, map[string]any{
					"name":       name,
					"username":   creds.Username,
					"user_id":    creds.UserID,
					"expires_at": creds.ExpiresAt,
					"is_expired": creds.IsExpired(),
				})
			}
		}
		return outfmt.WriteJSON(result, jqQuery)
	}

	f := outfmt.NewFormatter()
	f.Header("ACCOUNT", "USERNAME", "EXPIRES", "STATUS")

	currentAccount := getAccount()
	for _, name := range accounts {
		//nolint:errcheck // Error means account not found, which we handle by checking creds == nil
		creds, _ := store.Get(name)
		if creds == nil {
			continue
		}

		displayName := name
		if name == currentAccount {
			displayName = name + " *"
		}

		status := "active"
		if creds.IsExpired() {
			status = "expired"
		} else if creds.IsExpiringSoon(7 * 24 * time.Hour) {
			status = "expiring"
		}

		expires := "unknown"
		if !creds.ExpiresAt.IsZero() {
			expires = creds.ExpiresAt.Format("2006-01-02")
		}

		f.Row(displayName, "@"+creds.Username, expires, status)
	}
	f.Flush()

	return nil
}

func runAuthRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	store, err := getStore()
	if err != nil {
		return FormatError(err)
	}

	// Verify account exists
	if _, err := store.Get(name); err != nil {
		return FormatError(err)
	}

	if !confirm(fmt.Sprintf("Remove account %q?", name)) {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := store.Delete(name); err != nil {
		return WrapError("failed to remove account", err)
	}

	ui.Success("Account %q removed", name)
	return nil
}

// getClient returns a Threads API client for the current account.
// All errors are formatted with user-friendly messages and suggestions.
//
//nolint:unparam // ctx is required for interface consistency but not used internally
func getClient(ctx context.Context) (*threads.Client, error) {
	account, err := requireAccount()
	if err != nil {
		return nil, FormatError(err)
	}

	store, err := getStore()
	if err != nil {
		return nil, FormatError(err)
	}

	creds, err := store.Get(account)
	if err != nil {
		return nil, FormatError(err)
	}

	if creds.IsExpired() {
		return nil, &UserFriendlyError{
			Message:    "Your access token has expired",
			Suggestion: "Run 'threads auth refresh' to get a new token, or 'threads auth login' to re-authenticate",
		}
	}

	client, err := threads.NewClientWithToken(creds.AccessToken, &threads.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
	})
	if err != nil {
		return nil, WrapError("failed to create API client", err)
	}

	return client, nil
}
