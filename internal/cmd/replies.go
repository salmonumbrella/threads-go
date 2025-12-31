package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
	"github.com/salmonumbrella/threads-go/internal/ui"
)

var repliesCmd = &cobra.Command{
	Use:   "replies",
	Short: "Manage replies to posts",
	Long:  `List, create, hide, and manage replies to Threads posts.`,
}

var repliesListCmd = &cobra.Command{
	Use:   "list [post-id]",
	Short: "List replies to a post",
	Long: `List all replies to a specific post.

Results are paginated and can be filtered with --limit.`,
	Args: cobra.ExactArgs(1),
	RunE: runRepliesList,
}

var repliesCreateCmd = &cobra.Command{
	Use:   "create [post-id]",
	Short: "Reply to a post",
	Long: `Create a reply to a specific post.

Provide the text of your reply with the --text flag.`,
	Args: cobra.ExactArgs(1),
	RunE: runRepliesCreate,
}

var repliesHideCmd = &cobra.Command{
	Use:   "hide [reply-id]",
	Short: "Hide a reply",
	Long: `Hide a reply from public view.

Hidden replies are not visible to other users but can be unhidden later.
You can only hide replies on posts that you own.`,
	Args: cobra.ExactArgs(1),
	RunE: runRepliesHide,
}

var repliesUnhideCmd = &cobra.Command{
	Use:   "unhide [reply-id]",
	Short: "Unhide a reply",
	Long:  `Unhide a previously hidden reply, making it visible again.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRepliesUnhide,
}

var repliesConversationCmd = &cobra.Command{
	Use:   "conversation [post-id]",
	Short: "Get full conversation thread",
	Long: `Get the full conversation thread for a post.

Returns all replies in the conversation in a flattened format.`,
	Args: cobra.ExactArgs(1),
	RunE: runRepliesConversation,
}

// Replies command flags
var (
	replyText string
)

func init() {
	// List flags
	repliesListCmd.Flags().IntVar(&limitFlag, "limit", 25, "Maximum number of replies to return")

	// Create flags
	repliesCreateCmd.Flags().StringVarP(&replyText, "text", "t", "", "Text content for the reply (required)")
	//nolint:errcheck,gosec // MarkFlagRequired cannot fail for a flag that exists
	repliesCreateCmd.MarkFlagRequired("text")

	// Conversation flags
	repliesConversationCmd.Flags().IntVar(&limitFlag, "limit", 25, "Maximum number of posts to return")

	repliesCmd.AddCommand(repliesListCmd)
	repliesCmd.AddCommand(repliesCreateCmd)
	repliesCmd.AddCommand(repliesHideCmd)
	repliesCmd.AddCommand(repliesUnhideCmd)
	repliesCmd.AddCommand(repliesConversationCmd)
}

func runRepliesList(cmd *cobra.Command, args []string) error {
	postID := args[0]

	client, err := getClient(cmd.Context())
	if err != nil {
		return err
	}

	opts := &threads.RepliesOptions{}
	if limitFlag > 0 {
		opts.Limit = limitFlag
	}

	replies, err := client.GetReplies(cmd.Context(), threads.PostID(postID), opts)
	if err != nil {
		return WrapError("failed to get replies", err)
	}

	ctx := cmd.Context()

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(replies, jqQuery)
	}

	if len(replies.Data) == 0 {
		ui.Info("No replies found for post %s", postID)
		return nil
	}

	f := outfmt.NewFormatter()
	f.Header("ID", "USERNAME", "TEXT", "TIMESTAMP")

	for _, reply := range replies.Data {
		text := reply.Text
		if len(text) > 50 {
			text = text[:47] + "..."
		}
		f.Row(reply.ID, "@"+reply.Username, text, reply.Timestamp.Format("2006-01-02 15:04"))
	}
	f.Flush()

	if replies.Paging.Cursors != nil && replies.Paging.Cursors.After != "" {
		fmt.Printf("\nShowing %d replies. Use --limit to see more.\n", len(replies.Data))
	}

	return nil
}

func runRepliesCreate(cmd *cobra.Command, args []string) error {
	postID := args[0]

	if replyText == "" {
		return &UserFriendlyError{
			Message:    "Reply text is required",
			Suggestion: "Use the --text flag to provide your reply content",
		}
	}

	client, err := getClient(cmd.Context())
	if err != nil {
		return err
	}

	content := &threads.PostContent{
		Text: replyText,
	}

	reply, err := client.ReplyToPost(cmd.Context(), threads.PostID(postID), content)
	if err != nil {
		return WrapError("failed to create reply", err)
	}

	ctx := cmd.Context()

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(reply, jqQuery)
	}

	ui.Success("Reply created successfully!")
	fmt.Printf("  ID:        %s\n", reply.ID)
	fmt.Printf("  Permalink: %s\n", reply.Permalink)

	return nil
}

func runRepliesHide(cmd *cobra.Command, args []string) error {
	replyID := args[0]

	client, err := getClient(cmd.Context())
	if err != nil {
		return err
	}

	if err := client.HideReply(cmd.Context(), threads.PostID(replyID)); err != nil {
		return WrapError("failed to hide reply", err)
	}

	ctx := cmd.Context()

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(map[string]any{
			"success":  true,
			"reply_id": replyID,
			"action":   "hidden",
		}, jqQuery)
	}

	ui.Success("Reply %s hidden", replyID)
	return nil
}

func runRepliesUnhide(cmd *cobra.Command, args []string) error {
	replyID := args[0]

	client, err := getClient(cmd.Context())
	if err != nil {
		return err
	}

	if err := client.UnhideReply(cmd.Context(), threads.PostID(replyID)); err != nil {
		return WrapError("failed to unhide reply", err)
	}

	ctx := cmd.Context()

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(map[string]any{
			"success":  true,
			"reply_id": replyID,
			"action":   "unhidden",
		}, jqQuery)
	}

	ui.Success("Reply %s unhidden", replyID)
	return nil
}

func runRepliesConversation(cmd *cobra.Command, args []string) error {
	postID := args[0]

	client, err := getClient(cmd.Context())
	if err != nil {
		return err
	}

	opts := &threads.RepliesOptions{}
	if limitFlag > 0 {
		opts.Limit = limitFlag
	}

	conversation, err := client.GetConversation(cmd.Context(), threads.PostID(postID), opts)
	if err != nil {
		return WrapError("failed to get conversation", err)
	}

	ctx := cmd.Context()

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(conversation, jqQuery)
	}

	if len(conversation.Data) == 0 {
		ui.Info("No conversation found for post %s", postID)
		return nil
	}

	f := outfmt.NewFormatter()
	f.Header("ID", "USERNAME", "TEXT", "TIMESTAMP", "IS_REPLY")

	for _, post := range conversation.Data {
		text := post.Text
		if len(text) > 50 {
			text = text[:47] + "..."
		}
		isReply := "no"
		if post.IsReply {
			isReply = "yes"
		}
		f.Row(post.ID, "@"+post.Username, text, post.Timestamp.Format("2006-01-02 15:04"), isReply)
	}
	f.Flush()

	if conversation.Paging.Cursors != nil && conversation.Paging.Cursors.After != "" {
		fmt.Printf("\nShowing %d posts. Use --limit to see more.\n", len(conversation.Data))
	}

	return nil
}
