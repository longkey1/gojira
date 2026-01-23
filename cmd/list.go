package cmd

import (
	"context"
	"fmt"

	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jira"
	"github.com/spf13/cobra"
)

var (
	listJQL    string
	listFields string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets matching JQL",
	Long: `List tickets matching JQL.

Examples:
  # List all issues in a project
  gojira list --jql 'project = PROJ'

  # List with specific fields
  gojira list --jql 'project = PROJ' --fields 'summary,status,assignee'`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&listJQL, "jql", "", "JQL query (required)")
	listCmd.Flags().StringVar(&listFields, "fields", "*all", "Fields to retrieve (comma-separated, default: *all)")
	listCmd.MarkFlagRequired("jql")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	fields := parseFields(listFields)

	issues, err := client.SearchJQLAll(ctx, listJQL, fields)
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	return outputJSON(issues)
}
