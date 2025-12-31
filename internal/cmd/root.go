package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-go/internal/iocontext"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
	"github.com/salmonumbrella/threads-go/internal/secrets"
)

var (
	// Version information (set via ldflags)
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Global flags
var (
	accountName  string
	outputFormat string
	colorMode    string
	debugMode    bool
	jqQuery      string
	yesFlag      bool
	limitFlag    int
)

// rootCmd is the base command
var rootCmd = &cobra.Command{
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

		// Inject IO context (if not already set for testing)
		if !iocontext.HasIO(ctx) {
			ctx = iocontext.WithIO(ctx, iocontext.DefaultIO())
		}

		// Set up output format context
		format := outfmt.Text
		if outputFormat == "json" {
			format = outfmt.JSON
		}
		ctx = outfmt.NewContext(ctx, format)

		// Add additional context values
		ctx = outfmt.WithQuery(ctx, jqQuery)
		ctx = outfmt.WithYes(ctx, yesFlag)
		ctx = outfmt.WithLimit(ctx, limitFlag)

		cmd.SetContext(ctx)
		return nil
	},
}

// Execute runs the root command
func Execute(ctx context.Context) error {
	rootCmd.SetContext(ctx)
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&accountName, "account", "a", "", "Account name to use (or set THREADS_ACCOUNT)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text, json")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "auto", "Color output: auto, always, never")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().StringVarP(&jqQuery, "query", "q", "", "JQ query to filter JSON output")
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().IntVar(&limitFlag, "limit", 0, "Limit number of results")

	// Environment variable fallbacks
	if accountName == "" {
		accountName = os.Getenv("THREADS_ACCOUNT")
	}
	if os.Getenv("THREADS_OUTPUT") != "" {
		outputFormat = os.Getenv("THREADS_OUTPUT")
	}
	if os.Getenv("NO_COLOR") != "" {
		colorMode = "never"
	}

	// Add subcommands
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(insightsCmd)
	rootCmd.AddCommand(newLocationsCmd())
	rootCmd.AddCommand(meCmd)
	rootCmd.AddCommand(postsCmd)
	rootCmd.AddCommand(newRateLimitCmd())
	rootCmd.AddCommand(repliesCmd)
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(versionCmd)
}

// versionCmd shows version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("threads %s\n", Version)
		fmt.Printf("  commit: %s\n", Commit)
		fmt.Printf("  built:  %s\n", BuildDate)
	},
}

// getStore returns the credential store
func getStore() (*secrets.KeyringStore, error) {
	return secrets.OpenDefault()
}

// getAccount returns the active account name
func getAccount() string {
	if accountName != "" {
		return accountName
	}
	// Try to find a default account
	store, err := getStore()
	if err != nil {
		return ""
	}
	accounts, err := store.List()
	if err != nil || len(accounts) == 0 {
		return ""
	}
	// Return first account as default
	return accounts[0]
}

// requireAccount ensures an account is selected
func requireAccount() (string, error) {
	account := getAccount()
	if account == "" {
		return "", fmt.Errorf("no account configured. Run 'threads auth login' to authenticate")
	}
	return account, nil
}

// confirm prompts for confirmation unless --yes is set
func confirm(prompt string) bool {
	if yesFlag {
		return true
	}
	fmt.Printf("%s [y/N]: ", prompt)
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y" || response == "yes"
}
