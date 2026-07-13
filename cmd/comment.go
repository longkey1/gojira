package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/longkey1/gojira/internal/adf"
	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jira"
	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage issue comments",
}

var commentListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List comments on an issue",
	Long: `List all comments on a JIRA issue.

Examples:
  # List comments as raw ADF JSON
  gojira comment list PROJ-123

  # List comments with body converted to Markdown
  gojira comment list PROJ-123 --markdown-body`,
	Args: cobra.ExactArgs(1),
	RunE: runCommentList,
}

var commentAddCmd = &cobra.Command{
	Use:   "add <issue-key>",
	Short: "Add a comment to an issue",
	Long: `Add a comment to a JIRA issue.

Examples:
  # Add a comment with Markdown body
  gojira comment add PROJ-123 --markdown-body --body 'This is a comment'

  # Add a comment with ADF JSON body
  gojira comment add PROJ-123 --body '{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}'

  # Reply to an existing comment (quotes the original)
  gojira comment add PROJ-123 --markdown-body --body 'I agree' --reply-to 10001`,
	Args: cobra.ExactArgs(1),
	RunE: runCommentAdd,
}

var commentUpdateCmd = &cobra.Command{
	Use:   "update <issue-key>",
	Short: "Update a comment on an issue",
	Long: `Update an existing comment on a JIRA issue.

Examples:
  # Update a comment with Markdown body
  gojira comment update PROJ-123 --comment-id 10001 --markdown-body --body 'Updated text'

  # Update a comment with ADF JSON body
  gojira comment update PROJ-123 --comment-id 10001 --body '{"type":"doc","version":1,"content":[...]}'`,
	Args: cobra.ExactArgs(1),
	RunE: runCommentUpdate,
}

var commentDeleteCmd = &cobra.Command{
	Use:   "delete <issue-key>",
	Short: "Delete a comment from an issue",
	Long: `Delete a comment from a JIRA issue.

Examples:
  gojira comment delete PROJ-123 --comment-id 10001`,
	Args: cobra.ExactArgs(1),
	RunE: runCommentDelete,
}

func init() {
	commentCmd.AddCommand(commentListCmd)
	commentCmd.AddCommand(commentAddCmd)
	commentCmd.AddCommand(commentUpdateCmd)
	commentCmd.AddCommand(commentDeleteCmd)

	commentListCmd.Flags().Bool("markdown-body", false, "Convert comment body from ADF to Markdown in output")

	commentAddCmd.Flags().String("body", "", "Comment body (ADF JSON by default, Markdown with --markdown-body)")
	commentAddCmd.Flags().Bool("markdown-body", false, "Treat --body as Markdown (converted to ADF)")
	commentAddCmd.Flags().String("reply-to", "", "Comment ID to reply to (quotes the original comment)")
	_ = commentAddCmd.MarkFlagRequired("body")

	commentUpdateCmd.Flags().String("comment-id", "", "Comment ID to update")
	commentUpdateCmd.Flags().String("body", "", "New comment body (ADF JSON by default, Markdown with --markdown-body)")
	commentUpdateCmd.Flags().Bool("markdown-body", false, "Treat --body as Markdown (converted to ADF)")
	_ = commentUpdateCmd.MarkFlagRequired("comment-id")
	_ = commentUpdateCmd.MarkFlagRequired("body")

	commentDeleteCmd.Flags().String("comment-id", "", "Comment ID to delete")
	_ = commentDeleteCmd.MarkFlagRequired("comment-id")
}

func runCommentList(cmd *cobra.Command, args []string) error {
	issueKey := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	commentList, err := client.ListComments(ctx, issueKey)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}

	markdownBody, _ := cmd.Flags().GetBool("markdown-body")
	if markdownBody {
		return outputJSON(convertCommentBodyToMarkdown(commentList))
	}

	return outputJSON(commentList)
}

func runCommentAdd(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	bodyStr, _ := cmd.Flags().GetString("body")
	markdownBody, _ := cmd.Flags().GetBool("markdown-body")
	replyTo, _ := cmd.Flags().GetString("reply-to")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	// Handle reply: quote the original comment
	if replyTo != "" {
		originalComment, err := client.GetComment(ctx, issueKey, replyTo)
		if err != nil {
			return fmt.Errorf("failed to get original comment for reply: %w", err)
		}

		if originalComment.Body != nil {
			bodyMap := map[string]any{
				"type":    originalComment.Body.Type,
				"version": originalComment.Body.Version,
				"content": originalComment.Body.Content,
			}
			originalMarkdown := adf.ToMarkdown(bodyMap)

			authorName := "Unknown"
			if originalComment.Author != nil {
				authorName = originalComment.Author.DisplayName
			}

			var quoted strings.Builder
			fmt.Fprintf(&quoted, "> **%s** wrote:\n", authorName)
			for line := range strings.SplitSeq(originalMarkdown, "\n") {
				quoted.WriteString("> ")
				quoted.WriteString(line)
				quoted.WriteString("\n")
			}
			quoted.WriteString("\n")
			quoted.WriteString(bodyStr)
			bodyStr = quoted.String()
			markdownBody = true
		}
	}

	var bodyADF any
	if markdownBody {
		bodyADF = adf.FromMarkdown(bodyStr)
	} else {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(bodyStr), &parsed); err != nil {
			return fmt.Errorf("failed to parse body JSON: %w", err)
		}
		bodyADF = parsed
	}

	reqBody := map[string]any{
		"body": bodyADF,
	}

	comment, err := client.AddComment(ctx, issueKey, reqBody)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	if markdownBody {
		return outputJSON(convertCommentBodyToMarkdown(comment))
	}

	return outputJSON(comment)
}

func runCommentUpdate(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	commentID, _ := cmd.Flags().GetString("comment-id")
	bodyStr, _ := cmd.Flags().GetString("body")
	markdownBody, _ := cmd.Flags().GetBool("markdown-body")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	var bodyADF any
	if markdownBody {
		bodyADF = adf.FromMarkdown(bodyStr)
	} else {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(bodyStr), &parsed); err != nil {
			return fmt.Errorf("failed to parse body JSON: %w", err)
		}
		bodyADF = parsed
	}

	reqBody := map[string]any{
		"body": bodyADF,
	}

	comment, err := client.UpdateComment(ctx, issueKey, commentID, reqBody)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	if markdownBody {
		return outputJSON(convertCommentBodyToMarkdown(comment))
	}

	return outputJSON(comment)
}

func runCommentDelete(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	commentID, _ := cmd.Flags().GetString("comment-id")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	if err := client.DeleteComment(ctx, issueKey, commentID); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	return outputJSON(map[string]any{
		"deleted":   true,
		"commentId": commentID,
	})
}
