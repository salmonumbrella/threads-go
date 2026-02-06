package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/config"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// NewConfigCmd builds the config command group.
func NewConfigCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long:  `View and update local configuration defaults for api.`,
	}

	cmd.AddCommand(newConfigPathCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd(f))
	cmd.AddCommand(newConfigUnsetCmd(f))

	return cmd
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			io := iocontext.GetIO(cmd.Context())
			fmt.Fprintln(io.Out, config.ConfigPath()) //nolint:errcheck // Best-effort output
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			io := iocontext.GetIO(cmd.Context())
			if outfmt.IsJSON(cmd.Context()) {
				out := outfmt.FromContext(cmd.Context(), outfmt.WithWriter(io.Out))
				return out.Output(configToMap(cfg))
			}

			fmt.Fprintf(io.Out, "Account: %s\n", fallback(cfg.Account, "(none)")) //nolint:errcheck // Best-effort output
			fmt.Fprintf(io.Out, "Output:  %s\n", fallback(cfg.Output, "text"))    //nolint:errcheck // Best-effort output
			fmt.Fprintf(io.Out, "Color:   %s\n", fallback(cfg.Color, "auto"))     //nolint:errcheck // Best-effort output
			fmt.Fprintf(io.Out, "Debug:   %v\n", cfg.Debug)                       //nolint:errcheck // Best-effort output
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			value, ok := configValue(cfg, key)
			if !ok {
				return &UserFriendlyError{
					Message:    fmt.Sprintf("Unknown config key: %s", key),
					Suggestion: "Valid keys: account, output, color, debug, path",
				}
			}

			io := iocontext.GetIO(cmd.Context())
			if outfmt.IsJSON(cmd.Context()) {
				out := outfmt.FromContext(cmd.Context(), outfmt.WithWriter(io.Out))
				return out.Output(map[string]any{key: value})
			}

			fmt.Fprintln(io.Out, value) //nolint:errcheck // Best-effort output
			return nil
		},
	}
}

func newConfigSetCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])
			value := args[1]

			cfg, err := config.LoadFile(config.ConfigPath())
			if err != nil {
				return err
			}

			if err := applyConfigValue(cfg, key, value); err != nil {
				return err
			}

			if err := config.Save(cfg); err != nil {
				return err
			}

			f.Config = cfg

			io := iocontext.GetIO(cmd.Context())
			if outfmt.IsJSON(cmd.Context()) {
				out := outfmt.FromContext(cmd.Context(), outfmt.WithWriter(io.Out))
				return out.Output(map[string]any{
					"success": true,
					"config":  configToMap(cfg),
				})
			}

			fmt.Fprintf(io.Out, "Updated %s\n", key) //nolint:errcheck // Best-effort output
			return nil
		},
	}
}

func newConfigUnsetCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "unset [key]",
		Short: "Unset a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])

			cfg, err := config.LoadFile(config.ConfigPath())
			if err != nil {
				return err
			}

			if err := applyConfigValue(cfg, key, ""); err != nil {
				return err
			}

			if err := config.Save(cfg); err != nil {
				return err
			}

			f.Config = cfg

			io := iocontext.GetIO(cmd.Context())
			if outfmt.IsJSON(cmd.Context()) {
				out := outfmt.FromContext(cmd.Context(), outfmt.WithWriter(io.Out))
				return out.Output(map[string]any{
					"success": true,
					"config":  configToMap(cfg),
				})
			}

			fmt.Fprintf(io.Out, "Unset %s\n", key) //nolint:errcheck // Best-effort output
			return nil
		},
	}
}

func configToMap(cfg *config.Config) map[string]any {
	return map[string]any{
		"account": cfg.Account,
		"output":  cfg.Output,
		"color":   cfg.Color,
		"debug":   cfg.Debug,
		"path":    config.ConfigPath(),
	}
}

func configValue(cfg *config.Config, key string) (any, bool) {
	switch key {
	case "account":
		return cfg.Account, true
	case "output":
		return cfg.Output, true
	case "color":
		return cfg.Color, true
	case "debug":
		return cfg.Debug, true
	case "path":
		return config.ConfigPath(), true
	default:
		return nil, false
	}
}

func applyConfigValue(cfg *config.Config, key, value string) error {
	switch key {
	case "account":
		cfg.Account = value
	case "output":
		if value != "" && value != "text" && value != "json" && value != "jsonl" {
			return &UserFriendlyError{
				Message:    fmt.Sprintf("Invalid output value: %s", value),
				Suggestion: "Valid values: text, json, jsonl",
			}
		}
		cfg.Output = value
	case "color":
		if value != "" && value != "auto" && value != "always" && value != "never" {
			return &UserFriendlyError{
				Message:    fmt.Sprintf("Invalid color value: %s", value),
				Suggestion: "Valid values: auto, always, never",
			}
		}
		cfg.Color = value
	case "debug":
		if value == "" {
			cfg.Debug = false
			return nil
		}
		parsed, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Debug = parsed
	default:
		return &UserFriendlyError{
			Message:    fmt.Sprintf("Unknown config key: %s", key),
			Suggestion: "Valid keys: account, output, color, debug",
		}
	}
	return nil
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true", "1", "yes", "y":
		return true, nil
	case "false", "0", "no", "n":
		return false, nil
	default:
		return false, &UserFriendlyError{
			Message:    fmt.Sprintf("Invalid boolean value: %s", value),
			Suggestion: "Use true/false or 1/0",
		}
	}
}

func fallback(value, def string) string {
	if value == "" {
		return def
	}
	return value
}
