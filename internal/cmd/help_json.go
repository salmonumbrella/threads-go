package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type helpJSON struct {
	CommandPath string   `json:"command_path"`
	Use         string   `json:"use"`
	Short       string   `json:"short,omitempty"`
	Long        string   `json:"long,omitempty"`
	Example     string   `json:"example,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`

	Flags          []helpFlag `json:"flags,omitempty"`
	InheritedFlags []helpFlag `json:"inherited_flags,omitempty"`

	Subcommands []helpSubcommand `json:"subcommands,omitempty"`
}

type helpSubcommand struct {
	Name    string   `json:"name"`
	Use     string   `json:"use"`
	Short   string   `json:"short,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
	Hidden  bool     `json:"hidden,omitempty"`
}

type helpFlag struct {
	Name       string `json:"name"`
	Shorthand  string `json:"shorthand,omitempty"`
	Type       string `json:"type,omitempty"`
	Usage      string `json:"usage,omitempty"`
	Default    string `json:"default,omitempty"`
	Required   bool   `json:"required,omitempty"`
	Hidden     bool   `json:"hidden,omitempty"`
	Deprecated bool   `json:"deprecated,omitempty"`
}

func NewHelpJSONCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "help-json [command...]",
		Aliases: []string{"hj"},
		Short:   "Output command help as JSON (agent discovery)",
		Long: `Output machine-readable help for any command.

Examples:
  threads help-json
  threads help-json posts get
  threads help-json auth login`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			target := root
			if len(args) > 0 {
				found, rest, err := root.Find(args)
				if err != nil {
					return err
				}
				if len(rest) > 0 {
					return fmt.Errorf("unknown command path segment(s): %s", strings.Join(rest, " "))
				}
				target = found
			}

			payload := buildHelpJSON(target)

			io := iocontext.GetIO(cmd.Context())
			// This command always emits JSON.
			return outfmt.WriteJSONTo(io.Out, payload, outfmt.GetQuery(cmd.Context()))
		},
	}
}

func buildHelpJSON(cmd *cobra.Command) helpJSON {
	h := helpJSON{
		CommandPath: cmd.CommandPath(),
		Use:         cmd.Use,
		Short:       strings.TrimSpace(cmd.Short),
		Long:        strings.TrimSpace(cmd.Long),
		Example:     strings.TrimSpace(cmd.Example),
	}
	if len(cmd.Aliases) > 0 {
		h.Aliases = append([]string(nil), cmd.Aliases...)
		sort.Strings(h.Aliases)
	}

	h.Flags = flagsToHelp(cmd.NonInheritedFlags())
	h.InheritedFlags = flagsToHelp(cmd.InheritedFlags())

	for _, sub := range cmd.Commands() {
		h.Subcommands = append(h.Subcommands, helpSubcommand{
			Name:    sub.Name(),
			Use:     sub.Use,
			Short:   strings.TrimSpace(sub.Short),
			Aliases: append([]string(nil), sub.Aliases...),
			Hidden:  sub.Hidden,
		})
	}
	sort.Slice(h.Subcommands, func(i, j int) bool { return h.Subcommands[i].Name < h.Subcommands[j].Name })
	for i := range h.Subcommands {
		sort.Strings(h.Subcommands[i].Aliases)
	}

	return h
}

func flagsToHelp(fs *pflag.FlagSet) []helpFlag {
	var out []helpFlag
	if fs == nil {
		return out
	}
	fs.VisitAll(func(f *pflag.Flag) {
		if f == nil {
			return
		}
		out = append(out, helpFlag{
			Name:       f.Name,
			Shorthand:  f.Shorthand,
			Type:       f.Value.Type(),
			Usage:      strings.TrimSpace(f.Usage),
			Default:    f.DefValue,
			Required:   isFlagRequired(f),
			Hidden:     f.Hidden,
			Deprecated: f.Deprecated != "",
		})
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func isFlagRequired(f *pflag.Flag) bool {
	if f == nil {
		return false
	}
	req := f.Annotations[cobra.BashCompOneRequiredFlag]
	return len(req) > 0 && req[0] == "true"
}
