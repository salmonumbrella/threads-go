package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
	"github.com/salmonumbrella/threads-cli/internal/ui"
)

const (
	// containerPollingInterval is the time between status checks when waiting
	// for media container processing to complete.
	containerPollingInterval = 2 * time.Second
)

// NewPostsCmd builds the posts command group.
func NewPostsCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "posts",
		Aliases: []string{"post", "p"},
		Short:   "Manage posts",
		Long:    `Create, read, list, and delete posts on Threads.`,
	}

	cmd.AddCommand(newPostsCreateCmd(f))
	cmd.AddCommand(newPostsGetCmd(f))
	cmd.AddCommand(newPostsListCmd(f))
	cmd.AddCommand(newPostsDeleteCmd(f))
	cmd.AddCommand(newPostsCarouselCmd(f))
	cmd.AddCommand(newPostsQuoteCmd(f))
	cmd.AddCommand(newPostsRepostCmd(f))
	cmd.AddCommand(newPostsUnrepostCmd(f))
	cmd.AddCommand(newPostsGhostListCmd(f))

	return cmd
}

type postsCreateOptions struct {
	Text         string
	TextFile     string
	Emit         string
	ImageURL     string
	VideoURL     string
	AltText      string
	ReplyTo      string
	Poll         string
	Ghost        bool
	Topic        string
	Location     string
	ReplyControl string
	GIF          string
}

func newPostsCreateCmd(f *Factory) *cobra.Command {
	opts := &postsCreateOptions{}

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"new", "add"},
		Short:   "Create a new post",
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPostsCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Text, "text", "t", "", "Post text content")
	cmd.Flags().StringVar(&opts.TextFile, "text-file", "", "Read post text content from a file (or '-' for stdin)")
	cmd.Flags().StringVar(&opts.Emit, "emit", "", "Emit: json|id|url (useful for chaining; suppresses extra text output)")
	cmd.Flags().StringVar(&opts.ImageURL, "image", "", "Image URL for image posts")
	cmd.Flags().StringVar(&opts.VideoURL, "video", "", "Video URL for video posts")
	cmd.Flags().StringVar(&opts.AltText, "alt-text", "", "Alt text for media accessibility")
	cmd.Flags().StringVar(&opts.ReplyTo, "reply-to", "", "Post ID to reply to")
	cmd.Flags().StringVar(&opts.Poll, "poll", "", "Create a poll with comma-separated options (2-4 options, e.g., \"Yes,No\" or \"A,B,C,D\")")
	cmd.Flags().BoolVar(&opts.Ghost, "ghost", false, "Create a ghost post (text-only, expires in 24 hours, no replies allowed)")
	cmd.Flags().StringVar(&opts.Topic, "topic", "", "Add a topic tag to the post")
	cmd.Flags().StringVar(&opts.Location, "location", "", "Attach a location ID to the post (use 'threads locations search' to find IDs)")
	cmd.Flags().StringVar(&opts.ReplyControl, "reply-control", "", "Control who can reply: everyone, accounts_you_follow, mentioned_only")
	cmd.Flags().StringVar(&opts.GIF, "gif", "", "Attach a GIF using a Tenor GIF ID (text-only posts)")

	return cmd
}

func runPostsCreate(cmd *cobra.Command, f *Factory, opts *postsCreateOptions) error {
	ctx := cmd.Context()

	if strings.TrimSpace(opts.TextFile) != "" {
		if strings.TrimSpace(opts.Text) != "" {
			return &UserFriendlyError{
				Message:    "Cannot use both --text and --text-file",
				Suggestion: "Use --text for inline text, or --text-file to read from file/stdin",
			}
		}
		txt, err := readTextFileOrStdin(ctx, opts.TextFile)
		if err != nil {
			return err
		}
		opts.Text = txt
	}

	hasImage := opts.ImageURL != ""
	hasVideo := opts.VideoURL != ""
	hasText := opts.Text != ""
	hasPoll := opts.Poll != ""
	hasGIF := opts.GIF != ""

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

	if opts.Ghost && (hasImage || hasVideo) {
		return &UserFriendlyError{
			Message:    "Ghost posts can only contain text",
			Suggestion: "Remove --image or --video flags to create a ghost post",
		}
	}

	if hasPoll && (hasImage || hasVideo) {
		return &UserFriendlyError{
			Message:    "Poll posts can only contain text",
			Suggestion: "Remove --image or --video flags to create a poll post",
		}
	}

	if hasGIF && (hasImage || hasVideo) {
		return &UserFriendlyError{
			Message:    "GIF posts can only contain text",
			Suggestion: "Remove --image or --video flags to create a GIF post",
		}
	}

	var replyControl api.ReplyControl
	if opts.ReplyControl != "" {
		switch opts.ReplyControl {
		case "everyone":
			replyControl = api.ReplyControlEveryone
		case "accounts_you_follow":
			replyControl = api.ReplyControlAccountsYouFollow
		case "mentioned_only":
			replyControl = api.ReplyControlMentioned
		default:
			return &UserFriendlyError{
				Message:    fmt.Sprintf("Invalid reply-control value: %s", opts.ReplyControl),
				Suggestion: "Valid values are: everyone, accounts_you_follow, mentioned_only",
			}
		}
	}

	var pollAttachment *api.PollAttachment
	if hasPoll {
		options := strings.Split(opts.Poll, ",")
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
		pollAttachment = &api.PollAttachment{
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

	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	var post *api.Post

	switch {
	case hasImage:
		content := &api.ImagePostContent{
			Text:         opts.Text,
			ImageURL:     opts.ImageURL,
			AltText:      opts.AltText,
			ReplyTo:      opts.ReplyTo,
			ReplyControl: replyControl,
			TopicTag:     opts.Topic,
			LocationID:   opts.Location,
		}
		post, err = client.CreateImagePost(ctx, content)
	case hasVideo:
		content := &api.VideoPostContent{
			Text:         opts.Text,
			VideoURL:     opts.VideoURL,
			AltText:      opts.AltText,
			ReplyTo:      opts.ReplyTo,
			ReplyControl: replyControl,
			TopicTag:     opts.Topic,
			LocationID:   opts.Location,
		}
		post, err = client.CreateVideoPost(ctx, content)
	default:
		content := &api.TextPostContent{
			Text:           opts.Text,
			ReplyTo:        opts.ReplyTo,
			ReplyControl:   replyControl,
			TopicTag:       opts.Topic,
			LocationID:     opts.Location,
			PollAttachment: pollAttachment,
			IsGhostPost:    opts.Ghost,
		}
		if hasGIF {
			content.GIFAttachment = &api.GIFAttachment{
				GIFID:    opts.GIF,
				Provider: api.GIFProviderTenor,
			}
		}
		post, err = client.CreateTextPost(ctx, content)
	}

	if err != nil {
		return WrapError("failed to create post", err)
	}

	io := iocontext.GetIO(ctx)
	if cmd.Flags().Changed("emit") {
		mode, errEmit := parseEmitMode(opts.Emit)
		if errEmit != nil {
			return errEmit
		}
		return emitResult(ctx, io, mode, post.ID, post.Permalink, post)
	}

	if outfmt.IsJSON(ctx) {
		out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
		return out.Output(post)
	}

	p := f.UI(ctx)
	if opts.Ghost {
		p.Success("Ghost post created successfully! (expires in 24 hours)")
	} else {
		p.Success("Post created successfully!")
	}
	fmt.Fprintf(io.Out, "  ID:        %s\n", post.ID)        //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Permalink: %s\n", post.Permalink) //nolint:errcheck // Best-effort output
	if post.Text != "" {
		text := post.Text
		if len(text) > 50 {
			text = text[:50] + "..."
		}
		fmt.Fprintf(io.Out, "  Text:      %s\n", text) //nolint:errcheck // Best-effort output
	}

	return nil
}

func newPostsGetCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get [post-id]",
		Aliases: []string{"show"},
		Short:   "Get a single post by ID",
		Long: `Retrieve a single post by its ID.

		Example:
	  threads posts get 12345678901234567`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			postID, err := normalizeIDArg(args[0], "post")
			if err != nil {
				return err
			}
			return runPostsGet(cmd, f, postID)
		},
	}
	return cmd
}

func runPostsGet(cmd *cobra.Command, f *Factory, postID string) error {
	ctx := cmd.Context()
	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	post, err := client.GetPost(ctx, api.PostID(postID))
	if err != nil {
		return WrapError("failed to get post", err)
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) {
		out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
		return out.Output(post)
	}

	fmt.Fprintf(io.Out, "ID:        %s\n", post.ID)                                      //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "Username:  @%s\n", post.Username)                               //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "Type:      %s\n", post.MediaType)                               //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "Permalink: %s\n", post.Permalink)                               //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "Timestamp: %s\n", post.Timestamp.Format("2006-01-02 15:04:05")) //nolint:errcheck // Best-effort output

	if post.Text != "" {
		fmt.Fprintf(io.Out, "Text:      %s\n", post.Text) //nolint:errcheck // Best-effort output
	}
	if post.MediaURL != "" {
		fmt.Fprintf(io.Out, "Media URL: %s\n", post.MediaURL) //nolint:errcheck // Best-effort output
	}
	if post.IsReply {
		fmt.Fprintf(io.Out, "Reply to:  %s\n", post.ReplyTo) //nolint:errcheck // Best-effort output
	}
	if post.IsQuotePost {
		fmt.Fprintln(io.Out, "Quote:     yes") //nolint:errcheck // Best-effort output
	}

	return nil
}

func newPostsListCmd(f *Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List user's posts",
		Long: `List posts from the authenticated user.

Examples:
  # List recent posts
  threads posts list

  # List with pagination
  threads posts list --limit 10

  # Output as JSON
  threads posts list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPostsList(cmd, f, limit)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of results")
	return cmd
}

func runPostsList(cmd *cobra.Command, f *Factory, limit int) error {
	ctx := cmd.Context()

	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	creds, err := f.ActiveCredentials(ctx)
	if err != nil {
		return err
	}

	opts := &api.PaginationOptions{}
	if limit > 0 {
		opts.Limit = limit
	}

	postsResp, err := client.GetUserPosts(ctx, api.UserID(creds.UserID), opts)
	if err != nil {
		return WrapError("failed to list posts", err)
	}

	posts := postsResp.Data
	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}

	io := iocontext.GetIO(ctx)
	out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
	if outfmt.IsJSONL(ctx) {
		return out.Output(posts)
	}
	if outfmt.GetFormat(ctx) == outfmt.JSON {
		return out.Output(map[string]any{
			"posts":  posts,
			"paging": postsResp.Paging,
		})
	}

	if len(posts) == 0 {
		f.UI(ctx).Info("No posts found")
		return nil
	}

	fmtr := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
	fmtr.Header("ID", "TYPE", "TEXT", "TIMESTAMP")

	for _, post := range posts {
		text := strings.ReplaceAll(post.Text, "\n", " ")
		if len(text) > 40 {
			text = text[:40] + "..."
		}

		fmtr.Row(
			post.ID,
			post.MediaType,
			text,
			post.Timestamp.Format("2006-01-02 15:04"),
		)
	}
	fmtr.Flush()

	return nil
}

func newPostsDeleteCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete [post-id]",
		Aliases: []string{"del", "rm"},
		Short:   "Delete a post",
		Long: `Delete a post by its ID.

Requires confirmation unless --yes flag is provided.

Example:
  threads posts delete 12345678901234567
	  threads posts delete 12345678901234567 --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			postID, err := normalizeIDArg(args[0], "post")
			if err != nil {
				return err
			}
			return runPostsDelete(cmd, f, postID)
		},
	}
	return cmd
}

func runPostsDelete(cmd *cobra.Command, f *Factory, postID string) error {
	ctx := cmd.Context()
	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) && !outfmt.GetYes(ctx) {
		return &UserFriendlyError{
			Message:    "Refusing to prompt for confirmation in JSON output mode",
			Suggestion: "Re-run with --yes (or --no-prompt) to confirm deletion",
		}
	}

	if !outfmt.GetYes(ctx) {
		post, err := client.GetPost(ctx, api.PostID(postID))
		if err != nil {
			return WrapError("failed to get post", err)
		}

		fmt.Fprintln(io.Out, "Post to delete:")             //nolint:errcheck // Best-effort output
		fmt.Fprintf(io.Out, "  ID:   %s\n", post.ID)        //nolint:errcheck // Best-effort output
		fmt.Fprintf(io.Out, "  Type: %s\n", post.MediaType) //nolint:errcheck // Best-effort output
		if post.Text != "" {
			text := post.Text
			if len(text) > 50 {
				text = text[:50] + "..."
			}
			fmt.Fprintf(io.Out, "  Text: %s\n", text) //nolint:errcheck // Best-effort output
		}
		fmt.Fprintln(io.Out) //nolint:errcheck // Best-effort output

		if !f.Confirm(ctx, "Delete this post?") {
			fmt.Fprintln(io.Out, "Cancelled.") //nolint:errcheck // Best-effort output
			return nil
		}
	}

	if err := client.DeletePost(ctx, api.PostID(postID)); err != nil {
		return WrapError("failed to delete post", err)
	}

	if outfmt.IsJSON(ctx) {
		out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
		return out.Output(map[string]any{
			"ok":      true,
			"post_id": postID,
			"deleted": true,
			"action":  "delete_post",
		})
	}

	f.UI(ctx).Success("Post deleted successfully")
	return nil
}

type postsCarouselOptions struct {
	Items       []string
	Text        string
	AltTexts    []string
	ReplyTo     string
	TimeoutSecs int
}

func newPostsCarouselCmd(f *Factory) *cobra.Command {
	opts := &postsCarouselOptions{
		TimeoutSecs: 300,
	}
	var emit string

	cmd := &cobra.Command{
		Use:     "carousel",
		Aliases: []string{"car"},
		Short:   "Create a carousel post with multiple images/videos",
		Long: `Create a carousel post with 2-20 media items.

Each item should be a URL to an image or video. Alt text can be provided
for accessibility using --alt-text (one per item, in order).`,
		Example: `  # Create carousel with 3 images
  threads posts carousel --items url1,url2,url3

  # With caption and alt text
  threads posts carousel --items url1,url2 --text "My photos" --alt-text "First" --alt-text "Second"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPostsCarousel(cmd, f, opts, emit)
		},
	}

	cmd.Flags().StringSliceVar(&opts.Items, "items", nil, "Media URLs (comma-separated)")
	cmd.Flags().StringVar(&opts.Text, "text", "", "Caption text")
	cmd.Flags().StringSliceVar(&opts.AltTexts, "alt-text", nil, "Alt text for each item (in order)")
	cmd.Flags().StringVar(&opts.ReplyTo, "reply-to", "", "Post ID to reply to")
	cmd.Flags().IntVar(&opts.TimeoutSecs, "timeout", 300, "Timeout in seconds for container processing")
	cmd.Flags().StringVar(&emit, "emit", "", "Emit: json|id|url (useful for chaining; suppresses extra text output)")
	//nolint:errcheck,gosec // MarkFlagRequired cannot fail for a flag that exists
	cmd.MarkFlagRequired("items")

	return cmd
}

func runPostsCarousel(cmd *cobra.Command, f *Factory, opts *postsCarouselOptions, emit string) error {
	if len(opts.Items) < 2 {
		return &UserFriendlyError{
			Message:    "Carousel requires at least 2 items",
			Suggestion: "Add more items with --items or use 'threads posts create' for a single media post",
		}
	}
	if len(opts.Items) > 20 {
		return &UserFriendlyError{
			Message:    "Carousel supports maximum 20 items",
			Suggestion: "Reduce the number of items to 20 or fewer",
		}
	}

	ctx := cmd.Context()
	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	var containerIDs []string
	for i, itemURL := range opts.Items {
		var altText string
		if i < len(opts.AltTexts) {
			altText = opts.AltTexts[i]
		}

		mediaType := detectMediaType(itemURL)
		containerID, errContainer := client.CreateMediaContainer(ctx, mediaType, itemURL, altText)
		if errContainer != nil {
			return WrapError(fmt.Sprintf("failed to create container for item %d", i+1), errContainer)
		}

		if errWait := waitForContainer(ctx, client, containerID, opts.TimeoutSecs); errWait != nil {
			return WrapError(fmt.Sprintf("container %d not ready", i+1), errWait)
		}

		containerIDs = append(containerIDs, string(containerID))
	}

	content := &api.CarouselPostContent{
		Text:     opts.Text,
		Children: containerIDs,
	}
	if opts.ReplyTo != "" {
		content.ReplyTo = opts.ReplyTo
	}

	post, err := client.CreateCarouselPost(ctx, content)
	if err != nil {
		return WrapError("failed to create carousel post", err)
	}

	io := iocontext.GetIO(ctx)
	if cmd.Flags().Changed("emit") {
		mode, errEmit := parseEmitMode(emit)
		if errEmit != nil {
			return errEmit
		}
		return emitResult(ctx, io, mode, post.ID, post.Permalink, post)
	}
	if outfmt.IsJSON(ctx) {
		out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
		return out.Output(post)
	}

	f.UI(ctx).Success("Carousel post created successfully!")
	fmt.Fprintf(io.Out, "  ID:        %s\n", post.ID)        //nolint:errcheck // Best-effort output
	fmt.Fprintf(io.Out, "  Permalink: %s\n", post.Permalink) //nolint:errcheck // Best-effort output
	if post.Text != "" {
		text := post.Text
		if len(text) > 50 {
			text = text[:50] + "..."
		}
		fmt.Fprintf(io.Out, "  Text:      %s\n", text) //nolint:errcheck // Best-effort output
	}
	fmt.Fprintf(io.Out, "  Items:     %d\n", len(containerIDs)) //nolint:errcheck // Best-effort output

	return nil
}

func newPostsQuoteCmd(f *Factory) *cobra.Command {
	var text string
	var textFile string
	var emit string
	var imageURL string
	var videoURL string

	cmd := &cobra.Command{
		Use:     "quote [post-id]",
		Aliases: []string{"qt"},
		Short:   "Create a quote post",
		Long:    "Quote an existing post with optional text, image, or video.",
		Args:    cobra.ExactArgs(1),
		Example: `  # Quote with text
	  threads posts quote 12345 --text "Great point!"

	  # Quote with image
	  threads posts quote 12345 --image https://example.com/image.jpg --text "Check this out"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			quotedPostID, err := normalizeIDArg(args[0], "post")
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			if strings.TrimSpace(textFile) != "" {
				if strings.TrimSpace(text) != "" {
					return &UserFriendlyError{
						Message:    "Cannot use both --text and --text-file",
						Suggestion: "Use --text for inline text, or --text-file to read from file/stdin",
					}
				}
				txt, err := readTextFileOrStdin(ctx, textFile)
				if err != nil {
					return err
				}
				text = txt
			}

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			var content interface{}
			switch {
			case videoURL != "":
				content = &api.VideoPostContent{
					VideoURL: videoURL,
					Text:     text,
				}
			case imageURL != "":
				content = &api.ImagePostContent{
					ImageURL: imageURL,
					Text:     text,
				}
			default:
				content = &api.TextPostContent{
					Text: text,
				}
			}

			post, err := client.CreateQuotePost(ctx, content, quotedPostID)
			if err != nil {
				return WrapError("failed to create quote post", err)
			}

			io := iocontext.GetIO(ctx)
			if cmd.Flags().Changed("emit") {
				mode, errEmit := parseEmitMode(emit)
				if errEmit != nil {
					return errEmit
				}
				return emitResult(ctx, io, mode, post.ID, post.Permalink, post)
			}
			if outfmt.IsJSON(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				return out.Output(post)
			}

			f.UI(ctx).Success("Quote post created successfully!")
			fmt.Fprintf(io.Out, "  ID:        %s\n", post.ID)        //nolint:errcheck // Best-effort output
			fmt.Fprintf(io.Out, "  Permalink: %s\n", post.Permalink) //nolint:errcheck // Best-effort output
			if post.Text != "" {
				txt := post.Text
				if len(txt) > 50 {
					txt = txt[:50] + "..."
				}
				fmt.Fprintf(io.Out, "  Text:      %s\n", txt) //nolint:errcheck // Best-effort output
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&text, "text", "", "Quote text")
	cmd.Flags().StringVar(&textFile, "text-file", "", "Read quote text from a file (or '-' for stdin)")
	cmd.Flags().StringVar(&emit, "emit", "", "Emit: json|id|url (useful for chaining; suppresses extra text output)")
	cmd.Flags().StringVar(&imageURL, "image", "", "Image URL to include")
	cmd.Flags().StringVar(&videoURL, "video", "", "Video URL to include")

	return cmd
}

func newPostsRepostCmd(f *Factory) *cobra.Command {
	var emit string
	cmd := &cobra.Command{
		Use:     "repost [post-id]",
		Aliases: []string{"boost"},
		Short:   "Repost an existing post",
		Args:    cobra.ExactArgs(1),
		Example: `  threads posts repost 12345`,
		RunE: func(cmd *cobra.Command, args []string) error {
			postID, err := normalizeIDArg(args[0], "post")
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			post, err := client.RepostPost(ctx, api.PostID(postID))
			if err != nil {
				return WrapError("failed to repost", err)
			}

			io := iocontext.GetIO(ctx)
			if cmd.Flags().Changed("emit") {
				mode, errEmit := parseEmitMode(emit)
				if errEmit != nil {
					return errEmit
				}
				return emitResult(ctx, io, mode, post.ID, post.Permalink, post)
			}
			if outfmt.IsJSON(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				return out.Output(post)
			}

			f.UI(ctx).Success("Repost created successfully!")
			fmt.Fprintf(io.Out, "  ID:        %s\n", post.ID)        //nolint:errcheck // Best-effort output
			fmt.Fprintf(io.Out, "  Permalink: %s\n", post.Permalink) //nolint:errcheck // Best-effort output
			return nil
		},
	}
	cmd.Flags().StringVar(&emit, "emit", "", "Emit: json|id|url (useful for chaining; suppresses extra text output)")
	return cmd
}

func newPostsUnrepostCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unrepost [repost-id]",
		Aliases: []string{"undo-repost"},
		Short:   "Remove a repost",
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
			repostID, err := normalizeIDArg(args[0], "post")
			if err != nil {
				return err
			}

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			io := iocontext.GetIO(ctx)
			if outfmt.IsJSON(ctx) && !outfmt.GetYes(ctx) {
				return &UserFriendlyError{
					Message:    "Refusing to prompt for confirmation in JSON output mode",
					Suggestion: "Re-run with --yes (or --no-prompt) to confirm unrepost",
				}
			}
			if !outfmt.GetYes(ctx) {
				fmt.Fprintf(io.Out, "Repost to remove: %s\n\n", repostID) //nolint:errcheck // Best-effort output
				if !f.Confirm(ctx, "Remove this repost?") {
					fmt.Fprintln(io.Out, "Cancelled.") //nolint:errcheck // Best-effort output
					return nil
				}
			}

			if err := client.UnrepostPost(ctx, api.PostID(repostID)); err != nil {
				return WrapError("failed to unrepost", err)
			}

			if outfmt.IsJSON(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				return out.Output(map[string]any{
					"ok":        true,
					"repost_id": repostID,
					"deleted":   true,
					"action":    "unrepost",
				})
			}

			f.UI(ctx).Success("Repost removed successfully")
			return nil
		},
	}
	return cmd
}

func newPostsGhostListCmd(f *Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "ghost-list",
		Aliases: []string{"ghosts"},
		Short:   "List ghost posts",
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
			return runPostsGhostList(cmd, f, limit)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of results")
	return cmd
}

func runPostsGhostList(cmd *cobra.Command, f *Factory, limit int) error {
	ctx := cmd.Context()

	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	creds, err := f.ActiveCredentials(ctx)
	if err != nil {
		return err
	}

	opts := &api.PaginationOptions{}
	if limit > 0 {
		opts.Limit = limit
	}

	postsResp, err := client.GetUserGhostPosts(ctx, api.UserID(creds.UserID), opts)
	if err != nil {
		return WrapError("failed to list ghost posts", err)
	}

	posts := postsResp.Data
	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}

	io := iocontext.GetIO(ctx)
	out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
	if outfmt.IsJSONL(ctx) {
		return out.Output(posts)
	}
	if outfmt.GetFormat(ctx) == outfmt.JSON {
		return out.Output(map[string]any{
			"posts":  posts,
			"paging": postsResp.Paging,
		})
	}

	if len(posts) == 0 {
		f.UI(ctx).Info("No ghost posts found")
		return nil
	}

	fmtr := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
	fmtr.Header("ID", "TEXT", "EXPIRES", "STATUS")

	for _, post := range posts {
		text := strings.ReplaceAll(post.Text, "\n", " ")
		if len(text) > 40 {
			text = text[:40] + "..."
		}

		expires := "N/A"
		if !post.GhostPostExpirationTimestamp.IsZero() {
			absTime := post.GhostPostExpirationTimestamp.Format("2006-01-02 15:04")
			relTime := ui.FormatRelativeTime(post.GhostPostExpirationTimestamp.Time)
			expires = fmt.Sprintf("%s (%s)", absTime, relTime)
		}

		status := post.GhostPostStatus
		if status == "" {
			status = "active"
		}

		fmtr.Row(
			post.ID,
			text,
			expires,
			status,
		)
	}
	fmtr.Flush()

	return nil
}

// detectMediaType determines if URL is image or video based on file extension
func detectMediaType(rawURL string) string {
	lower := strings.ToLower(rawURL)
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
func waitForContainer(ctx context.Context, client *api.Client, containerID api.ContainerID, timeoutSecs int) error {
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

	timeout := time.After(time.Duration(timeoutSecs) * time.Second)
	ticker := time.NewTicker(containerPollingInterval)
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
		}
	}
}
