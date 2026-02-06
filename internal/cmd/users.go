package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// NewUsersCmd builds the users command group.
func NewUsersCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user", "u"},
		Short:   "Manage user profiles",
		Long:    `Retrieve and view user profile information.`,
	}

	cmd.AddCommand(newUsersMeCmd(f))
	cmd.AddCommand(newUsersGetCmd(f))
	cmd.AddCommand(newUsersLookupCmd(f))
	cmd.AddCommand(newUsersMentionsCmd(f))

	return cmd
}

// NewUsersMeCmd builds a top-level alias for "users me".
func NewUsersMeCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show current authenticated user info",
		Long:  `Display the profile information for the currently authenticated user.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUsersMe(cmd, f)
		},
	}
}

func newUsersMeCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show current authenticated user info",
		Long:  `Display the profile information for the currently authenticated user.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUsersMe(cmd, f)
		},
	}
}

func newUsersGetCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "get [user-id]",
		Aliases: []string{"show"},
		Short:   "Get user by ID",
		Long:    `Retrieve user profile information by their user ID.`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw := strings.TrimSpace(args[0])

			// Desire path: allow @username and Threads profile URLs by delegating to lookup.
			if strings.HasPrefix(raw, "@") {
				return runUsersLookup(cmd, f, raw)
			}
			if strings.Contains(raw, "://") {
				if username, ok := extractUsernameFromURL(raw); ok {
					return runUsersLookup(cmd, f, username)
				}
			}

			userID, err := normalizeIDArg(raw, "user")
			if err != nil {
				return err
			}
			return runUsersGet(cmd, f, userID)
		},
	}
}

func newUsersLookupCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "lookup [username]",
		Aliases: []string{"find", "search"},
		Short:   "Lookup public profile by username",
		Long: `Look up a public profile by username.

The username can be provided with or without the @ prefix.
This returns public profile information including follower counts and engagement metrics.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUsersLookup(cmd, f, args[0])
		},
	}
}

func runUsersMe(cmd *cobra.Command, f *Factory) error {
	ctx := cmd.Context()

	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	user, err := client.GetMe(ctx)
	if err != nil {
		return WrapError("failed to get user info", err)
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSONTo(io.Out, userToMap(user), outfmt.GetQuery(ctx))
	}

	printUserText(cmd.Context(), f, user)
	return nil
}

func runUsersGet(cmd *cobra.Command, f *Factory, userID string) error {
	ctx := cmd.Context()

	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	user, err := client.GetUser(ctx, api.UserID(userID))
	if err != nil {
		return WrapError("failed to get user", err)
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSONTo(io.Out, userToMap(user), outfmt.GetQuery(ctx))
	}

	printUserText(ctx, f, user)
	return nil
}

func runUsersLookup(cmd *cobra.Command, f *Factory, username string) error {
	ctx := cmd.Context()

	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	publicUser, err := client.LookupPublicProfile(ctx, username)
	if err != nil {
		return WrapError("failed to lookup profile", err)
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSONTo(io.Out, publicUserToMap(publicUser), outfmt.GetQuery(ctx))
	}

	printPublicUserText(ctx, f, publicUser)
	return nil
}

// userToMap converts a User to a map for JSON output
func userToMap(u *api.User) map[string]any {
	return map[string]any{
		"id":              u.ID,
		"username":        u.Username,
		"name":            u.Name,
		"profile_pic_url": u.ProfilePicURL,
		"biography":       u.Biography,
		"is_verified":     u.IsVerified,
	}
}

// publicUserToMap converts a PublicUser to a map for JSON output
func publicUserToMap(u *api.PublicUser) map[string]any {
	return map[string]any{
		"username":            u.Username,
		"name":                u.Name,
		"profile_picture_url": u.ProfilePictureURL,
		"biography":           u.Biography,
		"is_verified":         u.IsVerified,
		"follower_count":      u.FollowerCount,
		"likes_count":         u.LikesCount,
		"quotes_count":        u.QuotesCount,
		"replies_count":       u.RepliesCount,
		"reposts_count":       u.RepostsCount,
		"views_count":         u.ViewsCount,
	}
}

// printUserText prints a User in text format
func printUserText(ctx context.Context, f *Factory, u *api.User) {
	p := f.UI(ctx)
	io := iocontext.GetIO(ctx)
	p.Success("User Profile")
	fmt.Fprintf(io.Out, "  ID:        %s\n", u.ID)        //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Username:  @%s\n", u.Username) //nolint:errcheck // Best-effort output
	if u.Name != "" {
		fmt.Fprintf(io.Out, "  Name:      %s\n", u.Name) //nolint:errcheck // Best-effort output
	}
	if u.Biography != "" {
		fmt.Fprintf(io.Out, "  Bio:       %s\n", u.Biography) //nolint:errcheck // Best-effort output
	}
	if u.IsVerified {
		fmt.Fprintln(io.Out, "  Verified:  yes") //nolint:errcheck // Best-effort output
	}
	if u.ProfilePicURL != "" {
		fmt.Fprintf(io.Out, "  Picture:   %s\n", u.ProfilePicURL) //nolint:errcheck // Best-effort output
	}
}

// printPublicUserText prints a PublicUser in text format
func printPublicUserText(ctx context.Context, f *Factory, u *api.PublicUser) {
	p := f.UI(ctx)
	io := iocontext.GetIO(ctx)
	p.Success("Public Profile")
	fmt.Fprintf(io.Out, "  Username:   @%s\n", u.Username) //nolint:errcheck // Best-effort output
	if u.Name != "" {
		fmt.Fprintf(io.Out, "  Name:       %s\n", u.Name) //nolint:errcheck // Best-effort output
	}
	if u.Biography != "" {
		fmt.Fprintf(io.Out, "  Bio:        %s\n", u.Biography) //nolint:errcheck // Best-effort output
	}
	if u.IsVerified {
		fmt.Fprintln(io.Out, "  Verified:   yes") //nolint:errcheck // Best-effort output
	}
	fmt.Fprintln(io.Out)                                       //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Followers:  %d\n", u.FollowerCount) //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Likes:      %d\n", u.LikesCount)    //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Replies:    %d\n", u.RepliesCount)  //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Quotes:     %d\n", u.QuotesCount)   //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Reposts:    %d\n", u.RepostsCount)  //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Views:      %d\n", u.ViewsCount)    //nolint:errcheck // Best-effort output
	if u.ProfilePictureURL != "" {
		fmt.Fprintf(io.Out, "\n  Picture:    %s\n", u.ProfilePictureURL) //nolint:errcheck // Best-effort output
	}
}

func newUsersMentionsCmd(f *Factory) *cobra.Command {
	var limit int
	var cursor string

	cmd := &cobra.Command{
		Use:   "mentions",
		Short: "List posts mentioning you",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			me, err := client.GetMe(ctx)
			if err != nil {
				return WrapError("failed to get user info", err)
			}

			opts := &api.PaginationOptions{
				Limit: limit,
				After: cursor,
			}

			result, err := client.GetUserMentions(ctx, api.UserID(me.ID), opts)
			if err != nil {
				return WrapError("failed to get mentions", err)
			}

			// JSON output
			io := iocontext.GetIO(ctx)
			if outfmt.IsJSON(ctx) {
				return outfmt.WriteJSONTo(io.Out, result, outfmt.GetQuery(ctx))
			}

			out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))

			if len(result.Data) == 0 {
				out.Empty("No mentions found")
				return nil
			}

			headers := []string{"ID", "FROM", "TEXT", "TIMESTAMP"}
			rows := make([][]string, len(result.Data))
			for i, post := range result.Data {
				text := post.Text
				if len(text) > 50 {
					text = text[:47] + "..."
				}
				rows[i] = []string{
					post.ID,
					"@" + post.Username,
					text,
					post.Timestamp.Format("2006-01-02 15:04"),
				}
			}

			return out.Table(headers, rows, []outfmt.ColumnType{
				outfmt.ColumnID,
				outfmt.ColumnPlain,
				outfmt.ColumnPlain,
				outfmt.ColumnDate,
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")

	return cmd
}
