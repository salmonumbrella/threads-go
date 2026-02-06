package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// NewLocationsCmd builds the locations command group.
func NewLocationsCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "locations",
		Aliases: []string{"location", "loc"},
		Short:   "Location search and details",
	}

	cmd.AddCommand(newLocationsSearchCmd(f))
	cmd.AddCommand(newLocationsGetCmd(f))

	return cmd
}

func newLocationsSearchCmd(f *Factory) *cobra.Command {
	var lat, lng float64
	var best bool
	var emit string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for locations",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var query string
			if len(args) > 0 {
				query = args[0]
			}

			if best {
				emit = strings.ToLower(strings.TrimSpace(emit))
				if emit == "" {
					emit = "json"
				}
				switch emit {
				case "json", "id":
				default:
					return &UserFriendlyError{
						Message:    fmt.Sprintf("Invalid --emit value: %s", emit),
						Suggestion: "Valid values are: json, id",
					}
				}
			}

			if query == "" && lat == 0 && lng == 0 {
				return &UserFriendlyError{
					Message:    "No search criteria provided",
					Suggestion: "Provide either a search query or --lat/--lng coordinates",
				}
			}

			ctx := cmd.Context()
			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			var latPtr, lngPtr *float64
			if lat != 0 || lng != 0 {
				latPtr = &lat
				lngPtr = &lng
			}

			result, err := client.SearchLocations(ctx, query, latPtr, lngPtr)
			if err != nil {
				return WrapError("location search failed", err)
			}

			io := iocontext.GetIO(ctx)
			out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))

			if best {
				if len(result.Data) == 0 {
					return &UserFriendlyError{
						Message:    "No locations found",
						Suggestion: "Try a different query or broaden your search",
					}
				}

				item := result.Data[0]

				// When best+emit is requested, allow emitting a scalar in text mode for easy chaining.
				if !outfmt.IsJSON(ctx) {
					switch emit {
					case "id":
						fmt.Fprintln(io.Out, item.ID) //nolint:errcheck // Best-effort output
						return nil
					}
				}

				// JSON mode: emit stable wrapper.
				if outfmt.IsJSON(ctx) {
					switch emit {
					case "id":
						return out.Output(map[string]any{"id": item.ID})
					default:
						return out.Output(map[string]any{
							"id":   item.ID,
							"item": item,
						})
					}
				}

				// Text mode default.
				fmt.Fprintf(io.Out, "%s\n", item.ID) //nolint:errcheck // Best-effort output
				if strings.TrimSpace(item.Name) != "" {
					fmt.Fprintf(io.Out, "%s\n", item.Name) //nolint:errcheck // Best-effort output
				}
				if strings.TrimSpace(item.Address) != "" {
					fmt.Fprintf(io.Out, "%s\n", item.Address) //nolint:errcheck // Best-effort output
				}
				return nil
			}

			if outfmt.IsJSON(ctx) {
				return out.Output(result)
			}

			if len(result.Data) == 0 {
				out.Empty("No locations found")
				return nil
			}

			headers := []string{"ID", "NAME", "ADDRESS"}
			rows := make([][]string, len(result.Data))
			for i, loc := range result.Data {
				rows[i] = []string{
					loc.ID,
					loc.Name,
					loc.Address,
				}
			}

			return out.Table(headers, rows, nil)
		},
	}

	cmd.Flags().Float64Var(&lat, "lat", 0, "Latitude for coordinate search")
	cmd.Flags().Float64Var(&lng, "lng", 0, "Longitude for coordinate search")
	cmd.Flags().BoolVar(&best, "best", false, "Auto-select the best result (non-interactive)")
	cmd.Flags().StringVar(&emit, "emit", "json", "When using --best, emit: json|id")

	return cmd
}

func newLocationsGetCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [location-id]",
		Short: "Get location details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			locationID := args[0]

			ctx := cmd.Context()
			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			location, err := client.GetLocation(ctx, api.LocationID(locationID))
			if err != nil {
				return WrapError("failed to get location", err)
			}

			io := iocontext.GetIO(ctx)
			out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
			return out.Output(location)
		},
	}
	return cmd
}
