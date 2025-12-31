package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
	"github.com/salmonumbrella/threads-go/internal/ui"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage user profiles",
	Long:  `Retrieve and view user profile information.`,
}

var usersMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show current authenticated user info",
	Long:  `Display the profile information for the currently authenticated user.`,
	RunE:  runUsersMe,
}

var usersGetCmd = &cobra.Command{
	Use:   "get [user-id]",
	Short: "Get user by ID",
	Long:  `Retrieve user profile information by their user ID.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersGet,
}

var usersLookupCmd = &cobra.Command{
	Use:   "lookup [username]",
	Short: "Lookup public profile by username",
	Long: `Look up a public profile by username.

The username can be provided with or without the @ prefix.
This returns public profile information including follower counts and engagement metrics.`,
	Args: cobra.ExactArgs(1),
	RunE: runUsersLookup,
}

// meCmd is a top-level alias for "users me"
var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show current authenticated user info",
	Long:  `Display the profile information for the currently authenticated user.`,
	RunE:  runUsersMe,
}

func init() {
	usersCmd.AddCommand(usersMeCmd)
	usersCmd.AddCommand(usersGetCmd)
	usersCmd.AddCommand(usersLookupCmd)
	usersCmd.AddCommand(newUsersMentionsCmd())
}

func runUsersMe(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	user, err := client.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(userToMap(user), jqQuery)
	}

	printUserText(user)
	return nil
}

func runUsersGet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	userID := args[0]

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	user, err := client.GetUser(ctx, threads.UserID(userID))
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(userToMap(user), jqQuery)
	}

	printUserText(user)
	return nil
}

func runUsersLookup(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	username := args[0]

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	publicUser, err := client.LookupPublicProfile(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to lookup profile: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(publicUserToMap(publicUser), jqQuery)
	}

	printPublicUserText(publicUser)
	return nil
}

// userToMap converts a User to a map for JSON output
func userToMap(u *threads.User) map[string]any {
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
func publicUserToMap(u *threads.PublicUser) map[string]any {
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
func printUserText(u *threads.User) {
	ui.Success("User Profile")
	fmt.Printf("  ID:        %s\n", u.ID)
	fmt.Printf("  Username:  @%s\n", u.Username)
	if u.Name != "" {
		fmt.Printf("  Name:      %s\n", u.Name)
	}
	if u.Biography != "" {
		fmt.Printf("  Bio:       %s\n", u.Biography)
	}
	if u.IsVerified {
		fmt.Printf("  Verified:  yes\n")
	}
	if u.ProfilePicURL != "" {
		fmt.Printf("  Picture:   %s\n", u.ProfilePicURL)
	}
}

// printPublicUserText prints a PublicUser in text format
func printPublicUserText(u *threads.PublicUser) {
	ui.Success("Public Profile")
	fmt.Printf("  Username:   @%s\n", u.Username)
	if u.Name != "" {
		fmt.Printf("  Name:       %s\n", u.Name)
	}
	if u.Biography != "" {
		fmt.Printf("  Bio:        %s\n", u.Biography)
	}
	if u.IsVerified {
		fmt.Printf("  Verified:   yes\n")
	}
	fmt.Println()
	fmt.Printf("  Followers:  %d\n", u.FollowerCount)
	fmt.Printf("  Likes:      %d\n", u.LikesCount)
	fmt.Printf("  Replies:    %d\n", u.RepliesCount)
	fmt.Printf("  Quotes:     %d\n", u.QuotesCount)
	fmt.Printf("  Reposts:    %d\n", u.RepostsCount)
	fmt.Printf("  Views:      %d\n", u.ViewsCount)
	if u.ProfilePictureURL != "" {
		fmt.Printf("\n  Picture:    %s\n", u.ProfilePictureURL)
	}
}

func newUsersMentionsCmd() *cobra.Command {
	var limit int
	var after string

	cmd := &cobra.Command{
		Use:   "mentions",
		Short: "List posts mentioning you",
		Long:  `List posts where the authenticated user is mentioned.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := getClient(ctx)
			if err != nil {
				return err
			}

			// Get authenticated user
			me, err := client.GetMe(ctx)
			if err != nil {
				return fmt.Errorf("failed to get user info: %w", err)
			}

			opts := &threads.PaginationOptions{
				Limit: limit,
				After: after,
			}

			result, err := client.GetUserMentions(ctx, threads.UserID(me.ID), opts)
			if err != nil {
				return fmt.Errorf("failed to get mentions: %w", err)
			}

			// JSON output
			if outfmt.IsJSON(ctx) {
				return outfmt.WriteJSON(map[string]any{
					"posts":  result.Data,
					"paging": result.Paging,
				}, jqQuery)
			}

			// Text output
			if len(result.Data) == 0 {
				ui.Info("No mentions found")
				return nil
			}

			f := outfmt.NewFormatter()
			f.Header("ID", "FROM", "TEXT", "TIMESTAMP")

			for _, post := range result.Data {
				text := post.Text
				if len(text) > 50 {
					text = text[:47] + "..."
				}
				f.Row(
					post.ID,
					"@"+post.Username,
					text,
					post.Timestamp.Format("2006-01-02 15:04"),
				)
			}
			f.Flush()

			// Show pagination hint if there are more results
			if result.Paging.Cursors != nil && result.Paging.Cursors.After != "" {
				fmt.Printf("\nMore results available. Use --after %s to see next page.\n", result.Paging.Cursors.After)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum results")
	cmd.Flags().StringVar(&after, "after", "", "Pagination cursor for next page")

	return cmd
}
