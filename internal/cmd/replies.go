package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// NewRepliesCmd builds the replies command group.
func NewRepliesCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "replies",
		Aliases: []string{"reply", "r"},
		Short:   "Manage replies to posts",
		Long:    `List, create, hide, and manage replies to Threads posts.`,
	}

	cmd.AddCommand(newRepliesListCmd(f))
	cmd.AddCommand(newRepliesCreateCmd(f))
	cmd.AddCommand(newRepliesHideCmd(f))
	cmd.AddCommand(newRepliesUnhideCmd(f))
	cmd.AddCommand(newRepliesConversationCmd(f))

	return cmd
}

func newRepliesListCmd(f *Factory) *cobra.Command {
	var limit int
	var cursor string
	var all bool
	var noHints bool

	cmd := &cobra.Command{
		Use:     "list [post-id]",
		Aliases: []string{"ls"},
		Short:   "List replies to a post",
		Long: `List all replies to a specific post.

	Results are paginated and can be filtered with --limit.`,
		Args: cobra.ExactArgs(1),
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

			opts := &api.RepliesOptions{}
			if limit > 0 {
				opts.Limit = limit
			}
			if cursor != "" {
				opts.After = cursor
			}

			io := iocontext.GetIO(ctx)
			out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
			if !all {
				replies, errList := client.GetReplies(ctx, api.PostID(postID), opts)
				if errList != nil {
					return WrapError("failed to get replies", errList)
				}

				next := pagingAfter(replies.Paging)
				if !noHints && next != "" && io.ErrOut != nil {
					fmt.Fprintf(io.ErrOut, "\nMore results available. Use --cursor %s to see next page.\n", next) //nolint:errcheck // Best-effort output
				}

				if outfmt.IsJSONL(ctx) {
					return out.Output(replies.Data)
				}
				if outfmt.GetFormat(ctx) == outfmt.JSON {
					items := replies.Data
					if len(items) == 0 {
						items = []api.Post{}
					}
					return out.Output(itemsEnvelope(items, replies.Paging, next))
				}

				if len(replies.Data) == 0 {
					f.UI(ctx).Info("No replies found for post %s", postID)
					return nil
				}

				headers := []string{"ID", "FROM", "TEXT", "DATE"}
				rows := make([][]string, len(replies.Data))
				for i, reply := range replies.Data {
					text := strings.ReplaceAll(reply.Text, "\n", " ")
					if len(text) > 50 {
						text = text[:47] + "..."
					}
					rows[i] = []string{
						reply.ID,
						"@" + reply.Username,
						text,
						reply.Timestamp.Format("2006-01-02 15:04"),
					}
				}

				return out.Table(headers, rows, []outfmt.ColumnType{
					outfmt.ColumnID,
					outfmt.ColumnPlain,
					outfmt.ColumnPlain,
					outfmt.ColumnDate,
				})
			}

			// --all: auto-paginate.
			pageCursor := cursor
			var allReplies []api.Post
			var allRows [][]string
			var lastPaging api.Paging
			var nextCursor string

			for {
				opts.After = pageCursor
				replies, errList := client.GetReplies(ctx, api.PostID(postID), opts)
				if errList != nil {
					return WrapError("failed to get replies", errList)
				}

				lastPaging = replies.Paging
				nextCursor = pagingAfter(replies.Paging)

				if outfmt.IsJSONL(ctx) {
					if errOut := out.Output(replies.Data); errOut != nil {
						return errOut
					}
				} else if outfmt.GetFormat(ctx) == outfmt.JSON {
					allReplies = append(allReplies, replies.Data...)
				} else {
					for _, reply := range replies.Data {
						text := strings.ReplaceAll(reply.Text, "\n", " ")
						if len(text) > 50 {
							text = text[:47] + "..."
						}
						allRows = append(allRows, []string{
							reply.ID,
							"@" + reply.Username,
							text,
							reply.Timestamp.Format("2006-01-02 15:04"),
						})
					}
				}

				if nextCursor == "" || nextCursor == pageCursor || len(replies.Data) == 0 {
					break
				}
				pageCursor = nextCursor
			}

			if outfmt.GetFormat(ctx) == outfmt.JSON {
				items := allReplies
				if len(items) == 0 {
					items = []api.Post{}
				}
				return out.Output(itemsEnvelope(items, lastPaging, ""))
			}
			if outfmt.GetFormat(ctx) == outfmt.Text {
				if len(allRows) == 0 {
					f.UI(ctx).Info("No replies found for post %s", postID)
					return nil
				}
				return out.Table([]string{"ID", "FROM", "TEXT", "DATE"}, allRows, []outfmt.ColumnType{
					outfmt.ColumnID,
					outfmt.ColumnPlain,
					outfmt.ColumnPlain,
					outfmt.ColumnDate,
				})
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of replies to return")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages (auto-paginate)")
	cmd.Flags().BoolVar(&noHints, "no-hints", false, "Suppress pagination hints on stderr")
	return cmd
}

func newRepliesCreateCmd(f *Factory) *cobra.Command {
	var text string
	var textFile string
	var emit string

	cmd := &cobra.Command{
		Use:     "create [post-id]",
		Aliases: []string{"new", "add"},
		Short:   "Reply to a post",
		Long: `Create a reply to a specific post.

	Provide the text of your reply with --text or --text-file.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			postID, err := normalizeIDArg(args[0], "post")
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			if strings.TrimSpace(text) != "" && strings.TrimSpace(textFile) != "" {
				return &UserFriendlyError{
					Message:    "Cannot use both --text and --text-file",
					Suggestion: "Use --text for inline text, or --text-file to read from file/stdin",
				}
			}
			if strings.TrimSpace(text) == "" && strings.TrimSpace(textFile) == "" {
				return &UserFriendlyError{
					Message:    "No reply text provided",
					Suggestion: "Provide --text \"...\" or --text-file path (or '-' for stdin)",
				}
			}
			if strings.TrimSpace(textFile) != "" {
				txt, readErr := readTextFileOrStdin(ctx, textFile)
				if readErr != nil {
					return readErr
				}
				text = txt
			}

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			content := &api.PostContent{
				Text: text,
			}
			reply, err := client.ReplyToPost(ctx, api.PostID(postID), content)
			if err != nil {
				return WrapError("failed to create reply", err)
			}

			io := iocontext.GetIO(ctx)
			if cmd.Flags().Changed("emit") {
				mode, errEmit := parseEmitMode(emit)
				if errEmit != nil {
					return errEmit
				}
				return emitResult(ctx, io, mode, reply.ID, reply.Permalink, reply)
			}
			if outfmt.IsJSON(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				return out.Output(reply)
			}

			f.UI(ctx).Success("Reply created successfully!")
			return nil
		},
	}

	cmd.Flags().StringVarP(&text, "text", "t", "", "Text content for the reply (required)")
	cmd.Flags().StringVar(&textFile, "text-file", "", "Read reply text from a file (or '-' for stdin)")
	cmd.Flags().StringVar(&emit, "emit", "", "Emit: json|id|url (useful for chaining; suppresses extra text output)")
	return cmd
}

func newRepliesHideCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hide [reply-id]",
		Aliases: []string{"rm", "del"},
		Short:   "Hide a reply",
		Long: `Hide a reply from public view.

Hidden replies are not visible to other users but can be unhidden later.
	You can only hide replies on posts that you own.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			replyID, err := normalizeIDArg(args[0], "reply")
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			if err := client.HideReply(ctx, api.PostID(replyID)); err != nil {
				return WrapError("failed to hide reply", err)
			}

			io := iocontext.GetIO(ctx)
			if outfmt.IsJSON(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				return out.Output(map[string]any{
					"ok":       true,
					"reply_id": replyID,
					"hidden":   true,
					"action":   "hide_reply",
				})
			}

			f.UI(ctx).Success("Reply %s hidden", replyID)
			return nil
		},
	}
	return cmd
}

func newRepliesUnhideCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unhide [reply-id]",
		Aliases: []string{"restore"},
		Short:   "Unhide a reply",
		Long:    `Unhide a previously hidden reply, making it visible again.`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			replyID, err := normalizeIDArg(args[0], "reply")
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			if err := client.UnhideReply(ctx, api.PostID(replyID)); err != nil {
				return WrapError("failed to unhide reply", err)
			}

			io := iocontext.GetIO(ctx)
			if outfmt.IsJSON(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				return out.Output(map[string]any{
					"ok":       true,
					"reply_id": replyID,
					"hidden":   false,
					"action":   "unhide_reply",
				})
			}

			f.UI(ctx).Success("Reply %s unhidden", replyID)
			return nil
		},
	}
	return cmd
}

func newRepliesConversationCmd(f *Factory) *cobra.Command {
	var limit int
	var cursor string
	var all bool
	var noHints bool

	cmd := &cobra.Command{
		Use:     "conversation [post-id]",
		Aliases: []string{"thread"},
		Short:   "Get full conversation thread",
		Long: `Get the full conversation thread for a post.

	Returns all replies in the conversation in a flattened format.`,
		Args: cobra.ExactArgs(1),
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

			opts := &api.RepliesOptions{}
			if limit > 0 {
				opts.Limit = limit
			}
			if cursor != "" {
				opts.After = cursor
			}

			io := iocontext.GetIO(ctx)
			out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
			if !all {
				result, errConv := client.GetConversation(ctx, api.PostID(postID), opts)
				if errConv != nil {
					return WrapError("failed to get conversation", errConv)
				}

				next := pagingAfter(result.Paging)
				if !noHints && next != "" && io.ErrOut != nil {
					fmt.Fprintf(io.ErrOut, "\nMore results available. Use --cursor %s to see next page.\n", next) //nolint:errcheck // Best-effort output
				}

				if outfmt.IsJSONL(ctx) {
					return out.Output(result.Data)
				}
				if outfmt.GetFormat(ctx) == outfmt.JSON {
					items := result.Data
					if len(items) == 0 {
						items = []api.Post{}
					}
					return out.Output(itemsEnvelope(items, result.Paging, next))
				}

				if len(result.Data) == 0 {
					f.UI(ctx).Info("No conversation found for post %s", postID)
					return nil
				}

				headers := []string{"ID", "FROM", "TEXT", "DATE"}
				rows := make([][]string, len(result.Data))
				for i, reply := range result.Data {
					text := strings.ReplaceAll(reply.Text, "\n", " ")
					if len(text) > 50 {
						text = text[:47] + "..."
					}
					rows[i] = []string{
						reply.ID,
						"@" + reply.Username,
						text,
						reply.Timestamp.Format("2006-01-02 15:04"),
					}
				}

				return out.Table(headers, rows, []outfmt.ColumnType{
					outfmt.ColumnID,
					outfmt.ColumnPlain,
					outfmt.ColumnPlain,
					outfmt.ColumnDate,
				})
			}

			// --all: auto-paginate.
			pageCursor := cursor
			var allReplies []api.Post
			var allRows [][]string
			var lastPaging api.Paging
			var nextCursor string

			for {
				opts.After = pageCursor
				result, errConv := client.GetConversation(ctx, api.PostID(postID), opts)
				if errConv != nil {
					return WrapError("failed to get conversation", errConv)
				}

				lastPaging = result.Paging
				nextCursor = pagingAfter(result.Paging)

				if outfmt.IsJSONL(ctx) {
					if errOut := out.Output(result.Data); errOut != nil {
						return errOut
					}
				} else if outfmt.GetFormat(ctx) == outfmt.JSON {
					allReplies = append(allReplies, result.Data...)
				} else {
					for _, reply := range result.Data {
						text := strings.ReplaceAll(reply.Text, "\n", " ")
						if len(text) > 50 {
							text = text[:47] + "..."
						}
						allRows = append(allRows, []string{
							reply.ID,
							"@" + reply.Username,
							text,
							reply.Timestamp.Format("2006-01-02 15:04"),
						})
					}
				}

				if nextCursor == "" || nextCursor == pageCursor || len(result.Data) == 0 {
					break
				}
				pageCursor = nextCursor
			}

			if outfmt.GetFormat(ctx) == outfmt.JSON {
				items := allReplies
				if len(items) == 0 {
					items = []api.Post{}
				}
				return out.Output(itemsEnvelope(items, lastPaging, ""))
			}
			if outfmt.GetFormat(ctx) == outfmt.Text {
				if len(allRows) == 0 {
					f.UI(ctx).Info("No conversation found for post %s", postID)
					return nil
				}
				return out.Table([]string{"ID", "FROM", "TEXT", "DATE"}, allRows, []outfmt.ColumnType{
					outfmt.ColumnID,
					outfmt.ColumnPlain,
					outfmt.ColumnPlain,
					outfmt.ColumnDate,
				})
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of posts to return")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages (auto-paginate)")
	cmd.Flags().BoolVar(&noHints, "no-hints", false, "Suppress pagination hints on stderr")
	return cmd
}
