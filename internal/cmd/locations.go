package cmd

import (
	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
)

func newLocationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "locations",
		Aliases: []string{"location", "loc"},
		Short:   "Location search and details",
	}

	cmd.AddCommand(newLocationsSearchCmd())
	cmd.AddCommand(newLocationsGetCmd())

	return cmd
}

func newLocationsSearchCmd() *cobra.Command {
	var lat, lng float64

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for locations",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var query string
			if len(args) > 0 {
				query = args[0]
			}

			if query == "" && lat == 0 && lng == 0 {
				return &UserFriendlyError{
					Message:    "No search criteria provided",
					Suggestion: "Provide either a search query or --lat/--lng coordinates",
				}
			}

			client, err := getClient(cmd.Context())
			if err != nil {
				return err
			}

			var latPtr, lngPtr *float64
			if lat != 0 || lng != 0 {
				latPtr = &lat
				lngPtr = &lng
			}

			result, err := client.SearchLocations(cmd.Context(), query, latPtr, lngPtr)
			if err != nil {
				return WrapError("location search failed", err)
			}

			f := outfmt.FromContext(cmd.Context())

			if outfmt.IsJSON(cmd.Context()) {
				return f.Output(result)
			}

			if len(result.Data) == 0 {
				f.Empty("No locations found")
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

			return f.Table(headers, rows, nil)
		},
	}

	cmd.Flags().Float64Var(&lat, "lat", 0, "Latitude for coordinate search")
	cmd.Flags().Float64Var(&lng, "lng", 0, "Longitude for coordinate search")

	return cmd
}

func newLocationsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [location-id]",
		Short: "Get location details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			locationID := args[0]

			client, err := getClient(cmd.Context())
			if err != nil {
				return err
			}

			location, err := client.GetLocation(cmd.Context(), threads.LocationID(locationID))
			if err != nil {
				return WrapError("failed to get location", err)
			}

			f := outfmt.FromContext(cmd.Context())
			return f.Output(location)
		},
	}
	return cmd
}
