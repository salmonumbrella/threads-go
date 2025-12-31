package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	postsText         string
	postsImageURL     string
	postsVideoURL     string
	postsAltText      string
	postsReplyTo      string
	postsPoll         string
	postsGhost        bool
	postsTopic        string
	postsLocation     string
	postsReplyControl string
	postsGIF          string
)

// Carousel command flags
var (
	carouselItems       []string
	carouselText        string
	carouselAltTexts    []string
	carouselReplyTo     string
	carouselWaitTimeout int
)

func init() {
	// Create command flags
	postsCreateCmd.Flags().StringVarP(&postsText, "text", "t", "", "Post text content")
	postsCreateCmd.Flags().StringVar(&postsImageURL, "image", "", "Image URL for image posts")
	postsCreateCmd.Flags().StringVar(&postsVideoURL, "video", "", "Video URL for video posts")
	postsCreateCmd.Flags().StringVar(&postsAltText, "alt-text", "", "Alt text for media accessibility")
	postsCreateCmd.Flags().StringVar(&postsReplyTo, "reply-to", "", "Post ID to reply to")
	postsCreateCmd.Flags().StringVar(&postsPoll, "poll", "", "Create a poll with comma-separated options (2-4 options, e.g., \"Yes,No\" or \"A,B,C,D\")")
	postsCreateCmd.Flags().BoolVar(&postsGhost, "ghost", false, "Create a ghost post (text-only, expires in 24 hours, no replies allowed)")
	postsCreateCmd.Flags().StringVar(&postsTopic, "topic", "", "Add a topic tag to the post")
	postsCreateCmd.Flags().StringVar(&postsLocation, "location", "", "Attach a location ID to the post (use 'threads locations search' to find IDs)")
	postsCreateCmd.Flags().StringVar(&postsReplyControl, "reply-control", "", "Control who can reply: everyone, accounts_you_follow, mentioned_only")
	postsCreateCmd.Flags().StringVar(&postsGIF, "gif", "", "Attach a GIF using a Tenor GIF ID (text-only posts)")

	// Carousel command flags
	postsCarouselCmd.Flags().StringSliceVar(&carouselItems, "items", nil, "Media URLs (comma-separated)")
	postsCarouselCmd.Flags().StringVar(&carouselText, "text", "", "Caption text")
	postsCarouselCmd.Flags().StringSliceVar(&carouselAltTexts, "alt-text", nil, "Alt text for each item (in order)")
	postsCarouselCmd.Flags().StringVar(&carouselReplyTo, "reply-to", "", "Post ID to reply to")
	postsCarouselCmd.Flags().IntVar(&carouselWaitTimeout, "timeout", 300, "Timeout in seconds for container processing")
	//nolint:errcheck,gosec // MarkFlagRequired cannot fail for a flag that exists
	postsCarouselCmd.MarkFlagRequired("items")

	postsCmd.AddCommand(postsCreateCmd)
	postsCmd.AddCommand(postsGetCmd)
	postsCmd.AddCommand(postsListCmd)
	postsCmd.AddCommand(postsDeleteCmd)
	postsCmd.AddCommand(postsCarouselCmd)
	postsCmd.AddCommand(newPostsQuoteCmd())
	postsCmd.AddCommand(newPostsRepostCmd())
	postsCmd.AddCommand(newPostsUnrepostCmd())
	postsCmd.AddCommand(newPostsGhostListCmd())
}

var postsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new post",
	Long: `Create a new post on Threads.

Supports text, image, and video posts. For carousel posts, use 'threads posts carousel'.

Examples:
  # Create a text post
  threads posts create --text "Hello, Threads!"

  # Create an image post
  threads posts create --text "Check out this image" --image "https://example.com/image.jpg"

  # Create a video post
  threads posts create --video "https://example.com/video.mp4"

  # Create a reply
  threads posts create --text "Great post!" --reply-to 12345678901234567

  # Create a poll
  threads posts create --text "What's your favorite?" --poll "Option A,Option B,Option C"

  # Create a ghost post (expires in 24h)
  threads posts create --text "This will disappear!" --ghost

  # Create a post with topic and location
  threads posts create --text "At the coffee shop" --topic "coffee" --location "123456789"

  # Control who can reply
  threads posts create --text "Followers only discussion" --reply-control accounts_you_follow

  # Create a post with a GIF
  threads posts create --text "This is hilarious" --gif TENOR_GIF_ID`,
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
	hasPoll := postsPoll != ""
	hasGIF := postsGIF != ""

	if !hasText && !hasImage && !hasVideo {
		return &UserFriendlyError{
			Message:    "No content provided for the post",
			Suggestion: "Provide at least one of --text, --image, or --video",
		}
	}

	if hasImage && hasVideo {
		return &UserFriendlyError{
			Message:    "Cannot combine image and video in a single post",
			Suggestion: "Use --image OR --video, not both. For multiple media items, use 'threads posts carousel'",
		}
	}

	// Ghost posts are text-only
	if postsGhost && (hasImage || hasVideo) {
		return &UserFriendlyError{
			Message:    "Ghost posts can only contain text",
			Suggestion: "Remove --image or --video flags to create a ghost post",
		}
	}

	// Polls are text-only
	if hasPoll && (hasImage || hasVideo) {
		return &UserFriendlyError{
			Message:    "Poll posts can only contain text",
			Suggestion: "Remove --image or --video flags to create a poll post",
		}
	}

	// GIF posts are text-only
	if hasGIF && (hasImage || hasVideo) {
		return &UserFriendlyError{
			Message:    "GIF posts can only contain text",
			Suggestion: "Remove --image or --video flags to create a GIF post",
		}
	}

	// Parse and validate reply-control
	var replyControl threads.ReplyControl
	if postsReplyControl != "" {
		switch postsReplyControl {
		case "everyone":
			replyControl = threads.ReplyControlEveryone
		case "accounts_you_follow":
			replyControl = threads.ReplyControlAccountsYouFollow
		case "mentioned_only":
			replyControl = threads.ReplyControlMentioned
		default:
			return &UserFriendlyError{
				Message:    fmt.Sprintf("Invalid reply-control value: %s", postsReplyControl),
				Suggestion: "Valid values are: everyone, accounts_you_follow, mentioned_only",
			}
		}
	}

	// Parse poll options
	var pollAttachment *threads.PollAttachment
	if hasPoll {
		options := strings.Split(postsPoll, ",")
		for i := range options {
			options[i] = strings.TrimSpace(options[i])
		}
		if len(options) < 2 {
			return &UserFriendlyError{
				Message:    "Poll requires at least 2 options",
				Suggestion: "Provide comma-separated options, e.g., --poll \"Yes,No\"",
			}
		}
		if len(options) > 4 {
			return &UserFriendlyError{
				Message:    "Poll supports maximum 4 options",
				Suggestion: "Reduce the number of options to 4 or fewer",
			}
		}
		pollAttachment = &threads.PollAttachment{
			OptionA: options[0],
			OptionB: options[1],
		}
		if len(options) > 2 {
			pollAttachment.OptionC = options[2]
		}
		if len(options) > 3 {
			pollAttachment.OptionD = options[3]
		}
	}

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	var post *threads.Post

	switch {
	case hasImage:
		content := &threads.ImagePostContent{
			Text:         postsText,
			ImageURL:     postsImageURL,
			AltText:      postsAltText,
			ReplyTo:      postsReplyTo,
			ReplyControl: replyControl,
			TopicTag:     postsTopic,
			LocationID:   postsLocation,
		}
		post, err = client.CreateImagePost(ctx, content)
	case hasVideo:
		content := &threads.VideoPostContent{
			Text:         postsText,
			VideoURL:     postsVideoURL,
			AltText:      postsAltText,
			ReplyTo:      postsReplyTo,
			ReplyControl: replyControl,
			TopicTag:     postsTopic,
			LocationID:   postsLocation,
		}
		post, err = client.CreateVideoPost(ctx, content)
	default:
		content := &threads.TextPostContent{
			Text:           postsText,
			ReplyTo:        postsReplyTo,
			ReplyControl:   replyControl,
			TopicTag:       postsTopic,
			LocationID:     postsLocation,
			PollAttachment: pollAttachment,
			IsGhostPost:    postsGhost,
		}
		if hasGIF {
			content.GIFAttachment = &threads.GIFAttachment{
				GIFID:    postsGIF,
				Provider: threads.GIFProviderTenor,
			}
		}
		post, err = client.CreateTextPost(ctx, content)
	}

	if err != nil {
		return WrapError("failed to create post", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(post, jqQuery)
	}

	if postsGhost {
		ui.Success("Ghost post created successfully! (expires in 24 hours)")
	} else {
		ui.Success("Post created successfully!")
	}
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
		return WrapError("failed to get post", err)
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
		return WrapError("failed to get user info", err)
	}

	opts := &threads.PostsOptions{}
	if limitFlag > 0 {
		opts.Limit = limitFlag
	}

	postsResp, err := client.GetUserPosts(ctx, threads.UserID(me.ID), nil)
	if err != nil {
		return WrapError("failed to list posts", err)
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
		return WrapError("failed to get post", err)
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
		return WrapError("failed to delete post", err)
	}

	ui.Success("Post deleted successfully")
	return nil
}

var postsCarouselCmd = &cobra.Command{
	Use:   "carousel",
	Short: "Create a carousel post with multiple images/videos",
	Long: `Create a carousel post with 2-20 media items.

Each item should be a URL to an image or video. Alt text can be provided
for accessibility using --alt-text (one per item, in order).`,
	Example: `  # Create carousel with 3 images
  threads posts carousel --items url1,url2,url3

  # With caption and alt text
  threads posts carousel --items url1,url2 --text "My photos" --alt-text "First" --alt-text "Second"`,
	RunE: runPostsCarousel,
}

func runPostsCarousel(cmd *cobra.Command, args []string) error {
	// Validate: 2-20 items required
	if len(carouselItems) < 2 {
		return &UserFriendlyError{
			Message:    "Carousel requires at least 2 items",
			Suggestion: "Add more items with --items or use 'threads posts create' for a single media post",
		}
	}
	if len(carouselItems) > 20 {
		return &UserFriendlyError{
			Message:    "Carousel supports maximum 20 items",
			Suggestion: "Reduce the number of items to 20 or fewer",
		}
	}

	ctx := cmd.Context()
	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	// Create media containers for each item
	var containerIDs []string
	for i, itemURL := range carouselItems {
		var altText string
		if i < len(carouselAltTexts) {
			altText = carouselAltTexts[i]
		}

		// Detect media type from URL
		mediaType := detectMediaType(itemURL)
		containerID, errContainer := client.CreateMediaContainer(ctx, mediaType, itemURL, altText)
		if errContainer != nil {
			return WrapError(fmt.Sprintf("failed to create container for item %d", i+1), errContainer)
		}

		// Wait for container to be ready
		if errWait := waitForContainer(ctx, client, containerID, carouselWaitTimeout); errWait != nil {
			return WrapError(fmt.Sprintf("container %d not ready", i+1), errWait)
		}

		containerIDs = append(containerIDs, string(containerID))
	}

	// Build carousel content
	content := &threads.CarouselPostContent{
		Text:     carouselText,
		Children: containerIDs,
	}
	if carouselReplyTo != "" {
		content.ReplyTo = carouselReplyTo
	}

	post, err := client.CreateCarouselPost(ctx, content)
	if err != nil {
		return WrapError("failed to create carousel post", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(post, jqQuery)
	}

	ui.Success("Carousel post created successfully!")
	fmt.Printf("  ID:        %s\n", post.ID)
	fmt.Printf("  Permalink: %s\n", post.Permalink)
	if post.Text != "" {
		text := post.Text
		if len(text) > 50 {
			text = text[:50] + "..."
		}
		fmt.Printf("  Text:      %s\n", text)
	}
	fmt.Printf("  Items:     %d\n", len(containerIDs))

	return nil
}

// detectMediaType determines if URL is image or video based on file extension
func detectMediaType(rawURL string) string {
	lower := strings.ToLower(rawURL)
	// Remove query parameters for extension matching
	if idx := strings.Index(lower, "?"); idx != -1 {
		lower = lower[:idx]
	}
	videoExts := []string{".mp4", ".mov", ".m4v", ".webm"}
	for _, ext := range videoExts {
		if strings.HasSuffix(lower, ext) {
			return "VIDEO"
		}
	}
	return "IMAGE"
}

// waitForContainer polls container status until ready or timeout
func waitForContainer(ctx context.Context, client *threads.Client, containerID threads.ContainerID, timeoutSecs int) error {
	// Check status immediately first
	status, err := client.GetContainerStatus(ctx, containerID)
	if err != nil {
		return FormatError(err)
	}
	switch status.Status {
	case "FINISHED":
		return nil
	case "ERROR":
		return &UserFriendlyError{
			Message:    fmt.Sprintf("Media processing failed: %s", status.ErrorMessage),
			Suggestion: "Check that the media URL is accessible and the format is supported (JPEG, PNG for images; MP4 for videos)",
		}
	case "EXPIRED":
		return &UserFriendlyError{
			Message:    "Media container expired before publishing",
			Suggestion: "Re-upload the media and publish immediately after container creation",
		}
	}

	// If not ready, start polling
	timeout := time.After(time.Duration(timeoutSecs) * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return &UserFriendlyError{
				Message:    "Operation cancelled",
				Suggestion: "Try again if this was unintentional",
			}
		case <-timeout:
			return &UserFriendlyError{
				Message:    "Timeout waiting for media processing",
				Suggestion: "Media processing is taking too long. Try using a smaller file or increase timeout with --timeout",
			}
		case <-ticker.C:
			status, err := client.GetContainerStatus(ctx, containerID)
			if err != nil {
				return FormatError(err)
			}
			switch status.Status {
			case "FINISHED":
				return nil
			case "ERROR":
				return &UserFriendlyError{
					Message:    fmt.Sprintf("Media processing failed: %s", status.ErrorMessage),
					Suggestion: "Check that the media URL is accessible and the format is supported",
				}
			case "EXPIRED":
				return &UserFriendlyError{
					Message:    "Media container expired before publishing",
					Suggestion: "Re-upload the media and publish immediately after container creation",
				}
			}
			// Still IN_PROGRESS, continue waiting
		}
	}
}

func newPostsQuoteCmd() *cobra.Command {
	var text string
	var imageURL string
	var videoURL string

	cmd := &cobra.Command{
		Use:   "quote [post-id]",
		Short: "Create a quote post",
		Long:  "Quote an existing post with optional text, image, or video.",
		Args:  cobra.ExactArgs(1),
		Example: `  # Quote with text
  threads posts quote 12345 --text "Great point!"

  # Quote with image
  threads posts quote 12345 --image https://example.com/image.jpg --text "Check this out"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			quotedPostID := args[0]

			client, err := getClient(cmd.Context())
			if err != nil {
				return err
			}

			var content interface{}
			switch {
			case videoURL != "":
				content = &threads.VideoPostContent{
					VideoURL: videoURL,
					Text:     text,
				}
			case imageURL != "":
				content = &threads.ImagePostContent{
					ImageURL: imageURL,
					Text:     text,
				}
			default:
				content = &threads.TextPostContent{
					Text: text,
				}
			}

			post, err := client.CreateQuotePost(cmd.Context(), content, quotedPostID)
			if err != nil {
				return WrapError("failed to create quote post", err)
			}

			f := outfmt.FromContext(cmd.Context())
			return f.Output(post)
		},
	}

	cmd.Flags().StringVar(&text, "text", "", "Quote text")
	cmd.Flags().StringVar(&imageURL, "image", "", "Image URL to include")
	cmd.Flags().StringVar(&videoURL, "video", "", "Video URL to include")

	return cmd
}

func newPostsRepostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repost [post-id]",
		Short:   "Repost an existing post",
		Args:    cobra.ExactArgs(1),
		Example: `  threads posts repost 12345`,
		RunE: func(cmd *cobra.Command, args []string) error {
			postID := args[0]

			client, err := getClient(cmd.Context())
			if err != nil {
				return err
			}

			post, err := client.RepostPost(cmd.Context(), threads.PostID(postID))
			if err != nil {
				return WrapError("failed to repost", err)
			}

			f := outfmt.FromContext(cmd.Context())
			return f.Output(post)
		},
	}
	return cmd
}

func newPostsUnrepostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unrepost [repost-id]",
		Short: "Remove a repost",
		Long: `Remove a repost by its ID.

This undoes a repost action. Note that you need the repost ID, not the original post ID.
The repost ID is returned when you create a repost.

Requires confirmation unless --yes flag is provided.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Remove a repost with confirmation
  threads posts unrepost 12345678901234567

  # Remove a repost without confirmation
  threads posts unrepost 12345678901234567 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			repostID := args[0]

			client, err := getClient(ctx)
			if err != nil {
				return err
			}

			// Show confirmation unless --yes is set
			if !yesFlag {
				fmt.Printf("Repost to remove: %s\n\n", repostID)

				if !confirm("Remove this repost?") {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := client.UnrepostPost(ctx, threads.PostID(repostID)); err != nil {
				return WrapError("failed to unrepost", err)
			}

			ui.Success("Repost removed successfully")
			return nil
		},
	}
	return cmd
}

func newPostsGhostListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ghost-list",
		Short: "List ghost posts",
		Long: `List ghost posts from the authenticated user.

Ghost posts are text-only posts that automatically expire after 24 hours.
They do not allow replies.

Examples:
  # List ghost posts
  threads posts ghost-list

  # List with pagination
  threads posts ghost-list --limit 10

  # Output as JSON
  threads posts ghost-list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := getClient(ctx)
			if err != nil {
				return err
			}

			// Get user info to get user ID
			me, err := client.GetMe(ctx)
			if err != nil {
				return WrapError("failed to get user info", err)
			}

			opts := &threads.PaginationOptions{}
			if limitFlag > 0 {
				opts.Limit = limitFlag
			}

			postsResp, err := client.GetUserGhostPosts(ctx, threads.UserID(me.ID), opts)
			if err != nil {
				return WrapError("failed to list ghost posts", err)
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
				ui.Info("No ghost posts found")
				return nil
			}

			f := outfmt.NewFormatter()
			f.Header("ID", "TEXT", "EXPIRES", "STATUS")

			for _, post := range posts {
				text := post.Text
				if len(text) > 40 {
					text = text[:40] + "..."
				}
				text = strings.ReplaceAll(text, "\n", " ")

				expires := "N/A"
				if !post.GhostPostExpirationTimestamp.IsZero() {
					expires = post.GhostPostExpirationTimestamp.Format("2006-01-02 15:04")
				}

				status := post.GhostPostStatus
				if status == "" {
					status = "active"
				}

				f.Row(
					post.ID,
					text,
					expires,
					status,
				)
			}
			f.Flush()

			return nil
		},
	}

	return cmd
}
