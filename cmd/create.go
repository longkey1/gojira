package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/longkey1/gojira/internal/adf"
	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jira"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Long: `Create a new issue in JIRA.

Examples:
  # Create a task
  gojira create --project PROJ --type Task --summary 'New task'

  # Create a bug with priority and labels
  gojira create --project PROJ --type Bug --summary 'Bug report' --priority 'High' --labels 'bug'

  # Create with Markdown description (--markdown-description flag)
  gojira create --project PROJ --type Story --summary 'Feature' --markdown-description --description '## Background
See [here](https://example.com) for details'

  # Create a subtask
  gojira create --project PROJ --type Sub-task --summary 'Subtask' --parent PROJ-123

  # Create from a JSON file
  gojira create --data-file ./new-issue.json`,
	RunE: runCreate,
}

func init() {
	createCmd.Flags().String("project", "", "Project key (required unless using --data or --data-file)")
	createCmd.Flags().String("type", "", "Issue type name (required unless using --data or --data-file)")
	createCmd.Flags().String("summary", "", "Issue summary (required unless using --data or --data-file)")
	createCmd.Flags().String("description", "", "Issue description (ADF JSON by default, Markdown with --markdown-description)")
	createCmd.Flags().String("assignee", "", "Assignee account ID")
	createCmd.Flags().StringSlice("labels", nil, "Labels (comma-separated)")
	createCmd.Flags().String("priority", "", "Priority name")
	createCmd.Flags().String("parent", "", "Parent issue key (for subtasks)")
	createCmd.Flags().String("data", "", "Arbitrary JSON string for request body")
	createCmd.Flags().String("data-file", "", "Path to JSON file for request body (mutually exclusive with --data)")
	createCmd.Flags().Bool("markdown-description", false, "Treat --description as Markdown (converted to ADF)")
}

func buildCreateBody(cmd *cobra.Command) (map[string]any, error) {
	data, _ := cmd.Flags().GetString("data")
	dataFile, _ := cmd.Flags().GetString("data-file")

	if data != "" && dataFile != "" {
		return nil, fmt.Errorf("--data and --data-file are mutually exclusive")
	}

	body := map[string]any{}

	if dataFile != "" {
		content, err := os.ReadFile(dataFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read data file: %w", err)
		}
		if err := json.Unmarshal(content, &body); err != nil {
			return nil, fmt.Errorf("failed to parse data file: %w", err)
		}
	} else if data != "" {
		if err := json.Unmarshal([]byte(data), &body); err != nil {
			return nil, fmt.Errorf("failed to parse data: %w", err)
		}
	}

	fields, ok := body["fields"].(map[string]any)
	if !ok {
		fields = map[string]any{}
	}

	if cmd.Flags().Changed("project") {
		project, _ := cmd.Flags().GetString("project")
		fields["project"] = map[string]any{"key": project}
	}

	if cmd.Flags().Changed("type") {
		issueType, _ := cmd.Flags().GetString("type")
		fields["issuetype"] = map[string]any{"name": issueType}
	}

	if cmd.Flags().Changed("summary") {
		summary, _ := cmd.Flags().GetString("summary")
		fields["summary"] = summary
	}

	if cmd.Flags().Changed("description") {
		desc, _ := cmd.Flags().GetString("description")
		markdown, _ := cmd.Flags().GetBool("markdown-description")
		if markdown {
			fields["description"] = adf.FromMarkdown(desc)
		} else {
			var descADF map[string]any
			if err := json.Unmarshal([]byte(desc), &descADF); err != nil {
				return nil, fmt.Errorf("failed to parse description JSON: %w", err)
			}
			fields["description"] = descADF
		}
	}

	if cmd.Flags().Changed("assignee") {
		assignee, _ := cmd.Flags().GetString("assignee")
		fields["assignee"] = map[string]any{"accountId": assignee}
	}

	if cmd.Flags().Changed("labels") {
		labels, _ := cmd.Flags().GetStringSlice("labels")
		trimmed := make([]string, len(labels))
		for i, l := range labels {
			trimmed[i] = strings.TrimSpace(l)
		}
		fields["labels"] = trimmed
	}

	if cmd.Flags().Changed("priority") {
		priority, _ := cmd.Flags().GetString("priority")
		fields["priority"] = map[string]any{"name": priority}
	}

	if cmd.Flags().Changed("parent") {
		parent, _ := cmd.Flags().GetString("parent")
		fields["parent"] = map[string]any{"key": parent}
	}

	// Validate required fields
	if fields["project"] == nil {
		return nil, fmt.Errorf("--project is required")
	}
	if fields["issuetype"] == nil {
		return nil, fmt.Errorf("--type is required")
	}
	if fields["summary"] == nil {
		return nil, fmt.Errorf("--summary is required")
	}

	body["fields"] = fields
	return body, nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	body, err := buildCreateBody(cmd)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	issue, err := client.CreateIssue(ctx, body)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	return outputJSON(issue)
}
