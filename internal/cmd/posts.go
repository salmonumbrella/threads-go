package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
	"github.com/salmonumbrella/threads-go/internal/ui"
)

var postsCmd = &cobra.Command{
	Use:   "posts",
	Short: "Manage posts",
	Long:  `Create, read, list, and delete posts on Threads.`,
}

// Posts command flags
var (
	postsText     string
	postsImageURL string
	postsVideoURL string
	postsAltText  string
	postsReplyTo  string
)

func init() {
	// Create command flags
	postsCreateCmd.Flags().StringVarP(&postsText, "text", "t", "", "Post text content")
	postsCreateCmd.Flags().StringVar(&postsImageURL, "image", "", "Image URL for image posts")
	postsCreateCmd.Flags().StringVar(&postsVideoURL, "video", "", "Video URL for video posts")
	postsCreateCmd.Flags().StringVar(&postsAltText, "alt-text", "", "Alt text for media accessibility")
	postsCreateCmd.Flags().StringVar(&postsReplyTo, "reply-to", "", "Post ID to reply to")

	postsCmd.AddCommand(postsCreateCmd)
	postsCmd.AddCommand(postsGetCmd)
	postsCmd.AddCommand(postsListCmd)
	postsCmd.AddCommand(postsDeleteCmd)
}

var postsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new post",
	Long: `Create a new post on Threads.

Supports text, image, and video posts. For carousel posts, use the API directly.

Examples:
  # Create a text post
  threads posts create --text "Hello, Threads!"

  # Create an image post
  threads posts create --text "Check out this image" --image "https://example.com/image.jpg"

  # Create a video post
  threads posts create --video "https://example.com/video.mp4"

  # Create a reply
  threads posts create --text "Great post!" --reply-to 12345678901234567`,
	RunE: runPostsCreate,
}

var postsGetCmd = &cobra.Command{
	Use:   "get [post-id]",
	Short: "Get a single post by ID",
	Long: `Retrieve a single post by its ID.

Example:
  threads posts get 12345678901234567`,
	Args: cobra.ExactArgs(1),
	RunE: runPostsGet,
}

var postsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List user's posts",
	Long: `List posts from the authenticated user.

Examples:
  # List recent posts
  threads posts list

  # List with pagination
  threads posts list --limit 10

  # Output as JSON
  threads posts list --output json`,
	RunE: runPostsList,
}

var postsDeleteCmd = &cobra.Command{
	Use:   "delete [post-id]",
	Short: "Delete a post",
	Long: `Delete a post by its ID.

Requires confirmation unless --yes flag is provided.

Example:
  threads posts delete 12345678901234567
  threads posts delete 12345678901234567 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runPostsDelete,
}

func runPostsCreate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Determine post type based on flags
	hasImage := postsImageURL != ""
	hasVideo := postsVideoURL != ""
	hasText := postsText != ""

	if !hasText && !hasImage && !hasVideo {
		return fmt.Errorf("at least one of --text, --image, or --video is required")
	}

	if hasImage && hasVideo {
		return fmt.Errorf("cannot specify both --image and --video")
	}

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	var post *threads.Post

	switch {
	case hasImage:
		content := &threads.ImagePostContent{
			Text:     postsText,
			ImageURL: postsImageURL,
			AltText:  postsAltText,
			ReplyTo:  postsReplyTo,
		}
		post, err = client.CreateImagePost(ctx, content)
	case hasVideo:
		content := &threads.VideoPostContent{
			Text:     postsText,
			VideoURL: postsVideoURL,
			AltText:  postsAltText,
			ReplyTo:  postsReplyTo,
		}
		post, err = client.CreateVideoPost(ctx, content)
	default:
		content := &threads.TextPostContent{
			Text:    postsText,
			ReplyTo: postsReplyTo,
		}
		post, err = client.CreateTextPost(ctx, content)
	}

	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(post, jqQuery)
	}

	ui.Success("Post created successfully!")
	fmt.Printf("  ID:        %s\n", post.ID)
	fmt.Printf("  Permalink: %s\n", post.Permalink)
	if post.Text != "" {
		text := post.Text
		if len(text) > 50 {
			text = text[:50] + "..."
		}
		fmt.Printf("  Text:      %s\n", text)
	}

	return nil
}

func runPostsGet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	postID := args[0]

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	post, err := client.GetPost(ctx, threads.PostID(postID))
	if err != nil {
		return fmt.Errorf("failed to get post: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(post, jqQuery)
	}

	fmt.Printf("ID:        %s\n", post.ID)
	fmt.Printf("Username:  @%s\n", post.Username)
	fmt.Printf("Type:      %s\n", post.MediaType)
	fmt.Printf("Permalink: %s\n", post.Permalink)
	fmt.Printf("Timestamp: %s\n", post.Timestamp.Format("2006-01-02 15:04:05"))

	if post.Text != "" {
		fmt.Printf("Text:      %s\n", post.Text)
	}
	if post.MediaURL != "" {
		fmt.Printf("Media URL: %s\n", post.MediaURL)
	}
	if post.IsReply {
		fmt.Printf("Reply to:  %s\n", post.ReplyTo)
	}
	if post.IsQuotePost {
		fmt.Println("Quote:     yes")
	}

	return nil
}

func runPostsList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	// Get user info to get user ID
	me, err := client.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	opts := &threads.PostsOptions{}
	if limitFlag > 0 {
		opts.Limit = limitFlag
	}

	postsResp, err := client.GetUserPosts(ctx, threads.UserID(me.ID), nil)
	if err != nil {
		return fmt.Errorf("failed to list posts: %w", err)
	}

	// Apply limit if specified
	posts := postsResp.Data
	if limitFlag > 0 && len(posts) > limitFlag {
		posts = posts[:limitFlag]
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(map[string]any{
			"posts":  posts,
			"paging": postsResp.Paging,
		}, jqQuery)
	}

	if len(posts) == 0 {
		ui.Info("No posts found")
		return nil
	}

	f := outfmt.NewFormatter()
	f.Header("ID", "TYPE", "TEXT", "TIMESTAMP")

	for _, post := range posts {
		text := post.Text
		if len(text) > 40 {
			text = text[:40] + "..."
		}
		text = strings.ReplaceAll(text, "\n", " ")

		f.Row(
			post.ID,
			post.MediaType,
			text,
			post.Timestamp.Format("2006-01-02 15:04"),
		)
	}
	f.Flush()

	return nil
}

func runPostsDelete(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	postID := args[0]

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	// Get post details for confirmation
	post, err := client.GetPost(ctx, threads.PostID(postID))
	if err != nil {
		return fmt.Errorf("failed to get post: %w", err)
	}

	// Show post details and confirm
	if !yesFlag {
		fmt.Printf("Post to delete:\n")
		fmt.Printf("  ID:   %s\n", post.ID)
		fmt.Printf("  Type: %s\n", post.MediaType)
		if post.Text != "" {
			text := post.Text
			if len(text) > 50 {
				text = text[:50] + "..."
			}
			fmt.Printf("  Text: %s\n", text)
		}
		fmt.Println()

		if !confirm("Delete this post?") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := client.DeletePost(ctx, threads.PostID(postID)); err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	ui.Success("Post deleted successfully")
	return nil
}
