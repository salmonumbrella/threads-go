package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/auth"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
	"github.com/salmonumbrella/threads-cli/internal/secrets"
	"github.com/salmonumbrella/threads-cli/internal/ui"
)

var defaultAuthScopes = []string{
	"threads_basic",
	"threads_content_publish",
	"threads_manage_insights",
	"threads_manage_replies",
	"threads_read_replies",
}

// NewAuthCmd builds the auth command group.
func NewAuthCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Aliases: []string{"a"},
		Short:   "Manage authentication",
		Long:    `Authenticate with Threads and manage stored credentials.`,
	}

	cmd.AddCommand(newAuthLoginCmd(f))
	cmd.AddCommand(newAuthTokenCmd(f))
	cmd.AddCommand(newAuthRefreshCmd(f))
	cmd.AddCommand(newAuthStatusCmd(f))
	cmd.AddCommand(newAuthListCmd(f))
	cmd.AddCommand(newAuthRemoveCmd(f))

	return cmd
}

type authLoginOptions struct {
	Name         string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

func newAuthLoginCmd(f *Factory) *cobra.Command {
	opts := &authLoginOptions{
		Name:   "default",
		Scopes: append([]string{}, defaultAuthScopes...),
	}

	cmd := &cobra.Command{
		Use:     "login",
		Aliases: []string{"signin"},
		Short:   "Authenticate with Threads via browser",
		Long: `Opens a browser to authenticate with Threads using OAuth 2.0.

After authentication, your credentials are securely stored in the system keychain.
Tokens are automatically converted to long-lived tokens (60 days).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthLogin(cmd, f, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "default", "Account name for this login")
	cmd.Flags().StringVar(&opts.ClientID, "client-id", "", "Meta App Client ID (or THREADS_CLIENT_ID)")
	cmd.Flags().StringVar(&opts.ClientSecret, "client-secret", "", "Meta App Client Secret (or THREADS_CLIENT_SECRET)")
	cmd.Flags().StringVar(&opts.RedirectURI, "redirect-uri", "", "OAuth Redirect URI (or THREADS_REDIRECT_URI)")
	cmd.Flags().StringSliceVar(&opts.Scopes, "scopes", opts.Scopes, "OAuth scopes to request")

	return cmd
}

func runAuthLogin(cmd *cobra.Command, f *Factory, opts *authLoginOptions) error {
	clientID := opts.ClientID
	if clientID == "" {
		clientID = os.Getenv("THREADS_CLIENT_ID")
	}
	clientSecret := opts.ClientSecret
	if clientSecret == "" {
		clientSecret = os.Getenv("THREADS_CLIENT_SECRET")
	}
	redirectURI := opts.RedirectURI
	if redirectURI == "" {
		redirectURI = os.Getenv("THREADS_REDIRECT_URI")
	}

	if clientID == "" || clientSecret == "" {
		return &UserFriendlyError{
			Message:    "Client ID and secret are required for authentication",
			Suggestion: "Set via --client-id and --client-secret flags, or THREADS_CLIENT_ID and THREADS_CLIENT_SECRET environment variables. Get these from the Meta Developer Console",
		}
	}

	if redirectURI == "" {
		redirectURI = "http://127.0.0.1:8585/callback"
	}

	store, err := f.Store()
	if err != nil {
		return FormatError(err)
	}

	ctx := cmd.Context()
	p := f.UI(ctx)
	p.Info("Starting authentication flow...")
	p.Info("Opening browser for Threads authorization...")

	server := auth.NewOAuthServer(clientID, clientSecret, redirectURI, opts.Scopes)
	result, err := server.Start(ctx)
	if err != nil {
		return WrapError("authentication failed", err)
	}

	creds := secrets.Credentials{
		Name:         opts.Name,
		AccessToken:  result.AccessToken,
		UserID:       result.UserID,
		Username:     result.Username,
		ExpiresAt:    result.ExpiresAt,
		CreatedAt:    time.Now(),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
	}

	if err := store.Set(opts.Name, creds); err != nil {
		return WrapError("failed to store credentials", err)
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSONTo(io.Out, map[string]any{
			"account":           opts.Name,
			"user_id":           result.UserID,
			"username":          result.Username,
			"expires_at":        result.ExpiresAt,
			"is_expired":        time.Now().After(result.ExpiresAt),
			"days_until_expiry": time.Until(result.ExpiresAt).Hours() / 24,
			"scopes":            opts.Scopes,
		}, outfmt.GetQuery(ctx))
	}

	p.Success("Authentication successful!")
	fmt.Fprintf(io.Out, "  Account:  %s\n", opts.Name)                                                                                  //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  User:     @%s\n", result.Username)                                                                           //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Expires:  %s (%.0f days)\n", result.ExpiresAt.Format("2006-01-02"), time.Until(result.ExpiresAt).Hours()/24) //nolint:errcheck // Best-effort output

	return nil
}

type authTokenOptions struct {
	Name         string
	ClientID     string
	ClientSecret string
}

func newAuthTokenCmd(f *Factory) *cobra.Command {
	opts := &authTokenOptions{
		Name: "default",
	}

	cmd := &cobra.Command{
		Use:     "token [access-token]",
		Aliases: []string{"set-token"},
		Short:   "Authenticate with an existing access token",
		Long: `Use an existing access token to authenticate.

You can provide the token as an argument or via THREADS_ACCESS_TOKEN environment variable.
The CLI will validate the token and store it in your keychain.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthToken(cmd, f, opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "default", "Account name for this token")
	cmd.Flags().StringVar(&opts.ClientID, "client-id", "", "Meta App Client ID")
	cmd.Flags().StringVar(&opts.ClientSecret, "client-secret", "", "Meta App Client Secret")

	return cmd
}

func runAuthToken(cmd *cobra.Command, f *Factory, opts *authTokenOptions, args []string) error {
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

	clientID := opts.ClientID
	if clientID == "" {
		clientID = os.Getenv("THREADS_CLIENT_ID")
	}
	clientSecret := opts.ClientSecret
	if clientSecret == "" {
		clientSecret = os.Getenv("THREADS_CLIENT_SECRET")
	}

	cfg := &api.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Debug:        f.Debug,
	}
	if f.Debug {
		cfg.Logger = f.logger()
	}

	client, err := f.NewClient(token, cfg)
	if err != nil {
		return WrapError("failed to create client", err)
	}

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

	user, err := client.GetMe(ctx)
	if err != nil {
		return WrapError("failed to get user info", err)
	}

	store, err := f.Store()
	if err != nil {
		return FormatError(err)
	}

	expiresAt := time.Unix(debugInfo.Data.ExpiresAt, 0)
	creds := secrets.Credentials{
		Name:         opts.Name,
		AccessToken:  token,
		UserID:       debugInfo.Data.UserID,
		Username:     user.Username,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	if err := store.Set(opts.Name, creds); err != nil {
		return WrapError("failed to store credentials", err)
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSONTo(io.Out, map[string]any{
			"account":           opts.Name,
			"user_id":           debugInfo.Data.UserID,
			"username":          user.Username,
			"expires_at":        expiresAt,
			"is_expired":        time.Now().After(expiresAt),
			"days_until_expiry": time.Until(expiresAt).Hours() / 24,
		}, outfmt.GetQuery(ctx))
	}

	p := f.UI(ctx)
	p.Success("Token stored successfully!")
	fmt.Fprintf(io.Out, "  Account:  %s\n", opts.Name)                                                                    //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  User:     @%s\n", user.Username)                                                               //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Expires:  %s (%.0f days)\n", expiresAt.Format("2006-01-02"), time.Until(expiresAt).Hours()/24) //nolint:errcheck // Best-effort output

	return nil
}

func newAuthRefreshCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "refresh",
		Aliases: []string{"renew"},
		Short:   "Refresh the access token",
		Long:    `Refresh the current access token before it expires.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthRefresh(cmd, f)
		},
	}
}

func runAuthRefresh(cmd *cobra.Command, f *Factory) error {
	store, err := f.Store()
	if err != nil {
		return FormatError(err)
	}

	account := f.Account
	if account == "" {
		accounts, listErr := store.List()
		if listErr != nil {
			return FormatError(listErr)
		}
		if len(accounts) == 0 {
			return &UserFriendlyError{
				Message:    "No Threads account configured",
				Suggestion: "Run 'threads auth login' to authenticate with your Threads account",
			}
		}
		account = accounts[0]
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

	cfg := &api.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Debug:        f.Debug,
	}
	if f.Debug {
		cfg.Logger = f.logger()
	}

	client, err := f.NewClient(creds.AccessToken, cfg)
	if err != nil {
		return WrapError("failed to create client", err)
	}

	ctx := cmd.Context()
	if err := client.RefreshToken(ctx); err != nil {
		return WrapError("failed to refresh token", err)
	}

	tokenInfo := client.GetTokenInfo()
	creds.AccessToken = tokenInfo.AccessToken
	creds.ExpiresAt = tokenInfo.ExpiresAt

	if err := store.Set(account, *creds); err != nil {
		return WrapError("failed to update stored credentials", err)
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSONTo(io.Out, map[string]any{
			"account":           account,
			"expires_at":        creds.ExpiresAt,
			"is_expired":        creds.IsExpired(),
			"days_until_expiry": creds.DaysUntilExpiry(),
			"refreshed":         true,
		}, outfmt.GetQuery(ctx))
	}

	p := f.UI(ctx)
	p.Success("Token refreshed successfully!")
	fmt.Fprintf(io.Out, "  Account:  %s\n", account)                                                                                  //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Expires:  %s (%.0f days)\n", creds.ExpiresAt.Format("2006-01-02"), time.Until(creds.ExpiresAt).Hours()/24) //nolint:errcheck // Best-effort output

	return nil
}

func newAuthStatusCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Show authentication status",
		Long:    `Display the current authentication status and token expiry information.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthStatus(cmd, f)
		},
	}
}

func runAuthStatus(cmd *cobra.Command, f *Factory) error {
	store, err := f.Store()
	if err != nil {
		return FormatError(err)
	}

	account := f.Account
	if account == "" {
		accounts, listErr := store.List()
		if listErr != nil {
			return FormatError(listErr)
		}
		if len(accounts) == 0 {
			ctx := cmd.Context()
			io := iocontext.GetIO(ctx)
			if outfmt.IsJSON(ctx) {
				return outfmt.WriteJSONTo(io.Out, map[string]any{
					"configured": false,
				}, outfmt.GetQuery(ctx))
			}

			p := f.UI(cmd.Context())
			p.Warning("No account configured")
			fmt.Fprintln(io.Out, "\nRun 'threads auth login' to authenticate.") //nolint:errcheck // Best-effort output
			return nil
		}
		account = accounts[0]
	}

	creds, err := store.Get(account)
	if err != nil {
		return FormatError(err)
	}

	ctx := cmd.Context()
	io := iocontext.GetIO(ctx)

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSONTo(io.Out, map[string]any{
			"account":           account,
			"user_id":           creds.UserID,
			"username":          creds.Username,
			"expires_at":        creds.ExpiresAt,
			"is_expired":        creds.IsExpired(),
			"days_until_expiry": creds.DaysUntilExpiry(),
		}, outfmt.GetQuery(ctx))
	}

	p := f.UI(ctx)
	status := "active"
	statusColor := p.Green
	if creds.IsExpired() {
		status = "expired"
		statusColor = p.Red
	} else if creds.IsExpiringSoon(7 * 24 * time.Hour) {
		status = "expiring soon"
		statusColor = p.Yellow
	}

	fmt.Fprintf(io.Out, "Account:  %s\n", p.Bold(account))                 //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "User:     @%s\n", creds.Username)                 //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "User ID:  %s\n", creds.UserID)                    //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "Status:   %s\n", p.Colorize(status, statusColor)) //nolint:errcheck // Best-effort output

	if !creds.ExpiresAt.IsZero() {
		days := creds.DaysUntilExpiry()
		fmt.Fprintf(io.Out, "Expires:  %s (%s)\n", creds.ExpiresAt.Format("2006-01-02 15:04"), ui.FormatDuration(days)) //nolint:errcheck // Best-effort output
	}

	return nil
}

func newAuthListCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthList(cmd, f)
		},
	}
}

func runAuthList(cmd *cobra.Command, f *Factory) error {
	store, err := f.Store()
	if err != nil {
		return FormatError(err)
	}

	accounts, err := store.List()
	if err != nil {
		return WrapError("failed to list accounts", err)
	}

	ctx := cmd.Context()
	io := iocontext.GetIO(ctx)

	if len(accounts) == 0 {
		if outfmt.IsJSON(ctx) {
			// Keep stdout machine-readable.
			return outfmt.WriteJSONTo(io.Out, []any{}, outfmt.GetQuery(ctx))
		}

		p := f.UI(ctx)
		p.Info("No accounts configured")
		fmt.Fprintln(io.Out, "\nRun 'threads auth login' to authenticate.") //nolint:errcheck // Best-effort output
		return nil
	}

	if outfmt.IsJSON(ctx) {
		var result []map[string]any
		for _, name := range accounts {
			creds, _ := store.Get(name) //nolint:errcheck // handled via nil check
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
		return outfmt.WriteJSONTo(io.Out, result, outfmt.GetQuery(ctx))
	}

	fmtr := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
	fmtr.Header("ACCOUNT", "USERNAME", "EXPIRES", "STATUS")

	currentAccount := f.Account
	if currentAccount == "" && len(accounts) > 0 {
		currentAccount = accounts[0]
	}

	for _, name := range accounts {
		creds, _ := store.Get(name) //nolint:errcheck // handled via nil check
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

		fmtr.Row(displayName, "@"+creds.Username, expires, status)
	}
	fmtr.Flush()

	return nil
}

func newAuthRemoveCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "remove [account]",
		Aliases: []string{"rm", "del"},
		Short:   "Remove a stored account",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthRemove(cmd, f, args[0])
		},
	}
}

func runAuthRemove(cmd *cobra.Command, f *Factory, name string) error {
	store, err := f.Store()
	if err != nil {
		return FormatError(err)
	}

	if _, err := store.Get(name); err != nil {
		return FormatError(err)
	}

	ctx := cmd.Context()
	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) && !outfmt.GetYes(ctx) {
		return &UserFriendlyError{
			Message:    "Refusing to prompt for confirmation in JSON output mode",
			Suggestion: "Re-run with --yes (or --no-prompt) to confirm removal",
		}
	}

	if !f.Confirm(ctx, fmt.Sprintf("Remove account %q?", name)) {
		io := iocontext.GetIO(cmd.Context())
		fmt.Fprintln(io.Out, "Cancelled.") //nolint:errcheck // Best-effort output
		return nil
	}

	if err := store.Delete(name); err != nil {
		return WrapError("failed to remove account", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSONTo(io.Out, map[string]any{
			"ok":      true,
			"account": name,
			"removed": true,
		}, outfmt.GetQuery(ctx))
	}

	p := f.UI(ctx)
	p.Success("Account %q removed", name)
	return nil
}
