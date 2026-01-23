package cmd

import (
	"context"
	"fmt"

	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jira"
	"github.com/spf13/cobra"
)

var getFields string

var getCmd = &cobra.Command{
	Use:   "get <issue-key>",
	Short: "Get a single ticket by issue key",
	Long: `Get a single ticket by issue key.

Examples:
  # Get all fields
  gojira get PROJ-123

  # Get specific fields
  gojira get PROJ-123 --fields 'summary,status,assignee'`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func init() {
	getCmd.Flags().StringVar(&getFields, "fields", "*all", "Fields to retrieve (comma-separated, default: *all)")
}

func runGet(cmd *cobra.Command, args []string) error {
	issueKey := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	fields := parseFields(getFields)

	issue, err := client.GetIssue(ctx, issueKey, fields)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	return outputJSON(issue)
}
