package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

var (
	// Version information (set via ldflags)
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// RootOptions captures global flags.
type RootOptions struct {
	Account  string
	Output   string
	JSON     bool
	Color    string
	NoColor  bool
	Debug    bool
	Query    string
	Yes      bool
	NoPrompt bool
}

// Execute runs the CLI with a new factory and root command.
func Execute(ctx context.Context) error {
	f, err := NewFactory(ctx, FactoryOptions{})
	if err != nil {
		return err
	}

	cmd := NewRootCmd(f)
	cmd.SetContext(ctx)
	return ExecuteCommand(cmd, f)
}

// ExecuteCommand runs a prepared command and handles formatted errors.
func ExecuteCommand(cmd *cobra.Command, f *Factory) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	io := iocontext.GetIO(cmd.Context())
	if io != nil {
		cmd.SetOut(io.Out)
		cmd.SetErr(io.ErrOut)
	} else if f != nil && f.IO != nil {
		cmd.SetOut(f.IO.Out)
		cmd.SetErr(f.IO.ErrOut)
	}

	err := cmd.Execute()
	if err != nil {
		if io == nil && f != nil {
			io = f.IO
		}
		if io == nil {
			io = iocontext.DefaultIO()
		}
		WriteErrorTo(cmd.Context(), io.ErrOut, err)
	}
	return err
}

// NewRootCmd constructs the root command and wires subcommands.
func NewRootCmd(f *Factory) *cobra.Command {
	opts := &RootOptions{
		Account: f.Config.Account,
		Output:  f.Config.Output,
		Color:   f.Config.Color,
		Debug:   f.Config.Debug,
	}

	cmd := &cobra.Command{
		Use:   "threads",
		Short: "Threads CLI - Interact with Meta Threads from the command line",
		Long: `Threads CLI is a command-line interface for Meta's Threads API.

It provides full access to Threads functionality including:
  - Creating and managing posts (text, images, videos, carousels)
  - Reading and replying to threads
  - Managing your profile and viewing others
  - Accessing insights and analytics
  - Searching content

Designed to be agent-friendly for automation with Claude and other AI assistants.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if !iocontext.HasIO(ctx) {
				ctx = iocontext.WithIO(ctx, f.IO)
			}

			output := f.Config.Output
			if cmd.Flags().Changed("output") {
				output = opts.Output
			} else if cmd.Flags().Changed("json") && opts.JSON {
				output = "json"
			}
			if output == "" {
				output = "text"
			}
			if output != "text" && output != "json" {
				return &UserFriendlyError{
					Message:    fmt.Sprintf("Invalid output value: %s", output),
					Suggestion: "Valid values are: text, json",
				}
			}

			color := f.Config.Color
			if cmd.Flags().Changed("color") {
				color = opts.Color
			} else if cmd.Flags().Changed("no-color") && opts.NoColor {
				color = "never"
			}
			if color == "" {
				color = "auto"
			}
			if os.Getenv("NO_COLOR") != "" && !cmd.Flags().Changed("color") {
				color = "never"
			}
			if color != "auto" && color != "always" && color != "never" {
				return &UserFriendlyError{
					Message:    fmt.Sprintf("Invalid color value: %s", color),
					Suggestion: "Valid values are: auto, always, never",
				}
			}

			debug := f.Config.Debug
			if cmd.Flags().Changed("debug") {
				debug = opts.Debug
			}

			account := f.Config.Account
			if cmd.Flags().Changed("account") {
				account = opts.Account
			}

			f.Output = outfmt.ParseFormat(output)
			f.ColorMode = outfmt.ParseColorMode(color)
			f.Debug = debug
			f.Account = account

			ctx = outfmt.NewContext(ctx, f.Output)
			ctx = outfmt.WithQuery(ctx, opts.Query)
			ctx = outfmt.WithYes(ctx, opts.Yes || opts.NoPrompt)
			ctx = outfmt.WithColorMode(ctx, f.ColorMode)
			cmd.SetContext(ctx)

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.Account, "account", "a", opts.Account, "Account name to use (or set THREADS_ACCOUNT)")
	cmd.PersistentFlags().StringVarP(&opts.Output, "output", "o", opts.Output, "Output format: text, json")
	cmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "Shortcut for --output json")
	cmd.PersistentFlags().StringVar(&opts.Color, "color", opts.Color, "Color output: auto, always, never")
	cmd.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "Shortcut for --color never")
	cmd.PersistentFlags().BoolVar(&opts.Debug, "debug", opts.Debug, "Enable debug output")
	cmd.PersistentFlags().StringVarP(&opts.Query, "query", "q", "", "JQ query to filter JSON output")
	cmd.PersistentFlags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompts")
	cmd.PersistentFlags().BoolVar(&opts.NoPrompt, "no-prompt", false, "Alias for --yes (skip confirmations)")

	cmd.AddCommand(NewAuthCmd(f))
	cmd.AddCommand(NewCompletionCmd())
	cmd.AddCommand(NewInsightsCmd(f))
	cmd.AddCommand(NewLocationsCmd(f))
	cmd.AddCommand(NewUsersMeCmd(f))
	cmd.AddCommand(NewPostsCmd(f))
	cmd.AddCommand(NewRateLimitCmd(f))
	cmd.AddCommand(NewRepliesCmd(f))
	cmd.AddCommand(NewSearchCmd(f))
	cmd.AddCommand(NewUsersCmd(f))
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewWebhooksCmd(f))
	cmd.AddCommand(NewConfigCmd(f))
	cmd.AddCommand(NewHelpJSONCmd())

	return cmd
}

// NewVersionCmd shows version information.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			io := iocontext.GetIO(cmd.Context())
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSONTo(io.Out, map[string]any{
					"version":    Version,
					"commit":     Commit,
					"build_date": BuildDate,
				}, outfmt.GetQuery(cmd.Context()))
			}
			fmt.Fprintf(io.Out, "threads %s\n", Version)     //nolint:errcheck // Best-effort output
			fmt.Fprintf(io.Out, "  commit: %s\n", Commit)    //nolint:errcheck // Best-effort output
			fmt.Fprintf(io.Out, "  built:  %s\n", BuildDate) //nolint:errcheck // Best-effort output
			return nil
		},
	}
}
