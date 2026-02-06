package cmd

import (
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
				if emit == "" {
					emit = "json"
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
				mode, errMode := parseEmitMode(emit)
				if errMode != nil {
					return errMode
				}
				return emitResult(ctx, io, mode, item.ID, "", item)
			}

			if outfmt.IsJSONL(ctx) {
				return out.Output(result.Data)
			}
			if outfmt.GetFormat(ctx) == outfmt.JSON {
				items := result.Data
				if len(items) == 0 {
					items = []api.Location{}
				}
				return out.Output(itemsEnvelope(items, nil, ""))
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
			locationID, err := normalizeIDArg(args[0], "location")
			if err != nil {
				return err
			}

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
