package cmd

import (
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

			replies, err := client.GetReplies(ctx, api.PostID(postID), opts)
			if err != nil {
				return WrapError("failed to get replies", err)
			}

			io := iocontext.GetIO(ctx)
			out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
			if outfmt.IsJSONL(ctx) {
				return out.Output(replies.Data)
			}
			if outfmt.GetFormat(ctx) == outfmt.JSON {
				return out.Output(replies)
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
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of replies to return")
	return cmd
}

func newRepliesCreateCmd(f *Factory) *cobra.Command {
	var text string
	var textFile string

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

			content := &api.PostContent{
				Text: text,
			}
			reply, err := client.ReplyToPost(ctx, api.PostID(postID), content)
			if err != nil {
				return WrapError("failed to create reply", err)
			}

			io := iocontext.GetIO(ctx)
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

			result, err := client.GetConversation(ctx, api.PostID(postID), opts)
			if err != nil {
				return WrapError("failed to get conversation", err)
			}

			io := iocontext.GetIO(ctx)
			out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
			if outfmt.IsJSONL(ctx) {
				return out.Output(result.Data)
			}
			if outfmt.GetFormat(ctx) == outfmt.JSON {
				return out.Output(result)
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
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of posts to return")
	return cmd
}
