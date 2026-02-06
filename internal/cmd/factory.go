package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"golang.org/x/term"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/config"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
	"github.com/salmonumbrella/threads-cli/internal/secrets"
	"github.com/salmonumbrella/threads-cli/internal/ui"
)

// Factory provides shared dependencies and helpers for commands.
type Factory struct {
	IO         *iocontext.IO
	Config     *config.Config
	Store      func() (secrets.Store, error)
	NewClient  func(accessToken string, cfg *api.Config) (*api.Client, error)
	Output     outfmt.Format
	ColorMode  outfmt.ColorMode
	Debug      bool
	Account    string
	debugLog   api.Logger
	loggerOnce sync.Once
}

// FactoryOptions allows overriding factory dependencies (mainly for tests).
type FactoryOptions struct {
	IO        *iocontext.IO
	Config    *config.Config
	Store     func() (secrets.Store, error)
	NewClient func(accessToken string, cfg *api.Config) (*api.Client, error)
}

// NewFactory creates a new Factory with defaults.
func NewFactory(ctx context.Context, opts FactoryOptions) (*Factory, error) {
	io := opts.IO
	if io == nil {
		io = iocontext.GetIO(ctx)
	}

	cfg := opts.Config
	if cfg == nil {
		loaded, err := config.Load()
		if err != nil {
			return nil, err
		}
		cfg = loaded
	}

	store := opts.Store
	if store == nil {
		store = func() (secrets.Store, error) {
			return secrets.OpenDefault()
		}
	}

	newClient := opts.NewClient
	if newClient == nil {
		newClient = api.NewClientWithToken
	}

	return &Factory{
		IO:        io,
		Config:    cfg,
		Store:     store,
		NewClient: newClient,
		Output:    outfmt.ParseFormat(cfg.Output),
		ColorMode: outfmt.ParseColorMode(cfg.Color),
		Debug:     cfg.Debug,
		Account:   cfg.Account,
	}, nil
}

// UI returns a configured UI printer.
func (f *Factory) UI(ctx context.Context) *ui.Printer {
	io := iocontext.GetIO(ctx)
	if io == nil {
		io = f.IO
	}

	color := outfmt.GetColorMode(ctx)

	// In JSON mode, keep stdout clean for machine-readable output by routing
	// UI/status messages to stderr.
	out := io.Out
	if outfmt.IsJSON(ctx) && io.ErrOut != nil {
		out = io.ErrOut
	}
	return ui.NewWithWriters(out, io.ErrOut, color)
}

// ActiveCredentials returns the stored credentials for the active account.
// This is useful for avoiding extra API calls (e.g. GetMe) when we already
// have stable identifiers like user_id.
func (f *Factory) ActiveCredentials(_ context.Context) (*secrets.Credentials, error) {
	account, err := f.resolveAccount()
	if err != nil {
		return nil, err
	}

	store, err := f.Store()
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

	return creds, nil
}

// Client returns a Threads client for the active account.
func (f *Factory) Client(ctx context.Context) (*api.Client, error) {
	creds, err := f.ActiveCredentials(ctx)
	if err != nil {
		return nil, err
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
		return nil, WrapError("failed to create API client", err)
	}

	return client, nil
}

func (f *Factory) resolveAccount() (string, error) {
	if f.Account != "" {
		return f.Account, nil
	}

	store, err := f.Store()
	if err != nil {
		return "", FormatError(err)
	}

	accounts, err := store.List()
	if err != nil {
		return "", FormatError(err)
	}

	if len(accounts) == 0 {
		return "", &UserFriendlyError{
			Message:    "No Threads account configured",
			Suggestion: "Run 'threads auth login' to authenticate with your Threads account",
		}
	}

	return accounts[0], nil
}

func (f *Factory) logger() api.Logger {
	f.loggerOnce.Do(func() {
		f.debugLog = newStderrLogger(f.IO.ErrOut)
	})
	return f.debugLog
}

// Confirm prompts for confirmation unless --yes is set.
// Returns false when stdin is not a TTY.
func (f *Factory) Confirm(ctx context.Context, prompt string) bool {
	if outfmt.GetYes(ctx) {
		return true
	}

	io := iocontext.GetIO(ctx)
	if !isTerminalReader(io.In) {
		fmt.Fprintln(io.ErrOut, "error: cannot prompt for confirmation (stdin is not a terminal)")   //nolint:errcheck // Best-effort output
		fmt.Fprintln(io.ErrOut, "hint: use --yes (-y) to skip confirmation in non-interactive mode") //nolint:errcheck // Best-effort output
		return false
	}

	fmt.Fprintf(io.Out, "%s [y/N]: ", prompt) //nolint:errcheck // Best-effort output
	var response string
	//nolint:errcheck,gosec // Scanln error is fine - empty response means "no"
	fmt.Fscanln(io.In, &response)
	return response == "y" || response == "Y" || response == "yes"
}

func isTerminalReader(r any) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
