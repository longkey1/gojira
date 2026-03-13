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

var updateCmd = &cobra.Command{
	Use:   "update <issue-key>",
	Short: "Update an existing issue",
	Long: `Update an existing issue by issue key.

Examples:
  # Update summary
  gojira update PROJ-123 --summary 'New title'

  # Update description with ADF JSON
  gojira update PROJ-123 --description '{"type":"doc","version":1,"content":[...]}'

  # Update description with Markdown (--markdown-description flag)
  gojira update PROJ-123 --markdown-description --description '## Overview
- Item 1
- Item 2'

  # Update multiple fields
  gojira update PROJ-123 --summary 'New title' --priority 'High' --labels 'bug,critical'

  # Update with arbitrary JSON data
  gojira update PROJ-123 --data '{"fields":{"customfield_10001":"value"}}'

  # Update from a JSON file
  gojira update PROJ-123 --data-file ./update.json

  # Use output from 'gojira get' as base
  gojira get PROJ-123 > issue.json
  gojira update PROJ-123 --data-file ./issue.json --summary 'Updated title'`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().String("summary", "", "Issue summary")
	updateCmd.Flags().String("description", "", "Issue description (ADF JSON by default, Markdown with --markdown-description)")
	updateCmd.Flags().String("assignee", "", "Assignee account ID (empty string to unassign)")
	updateCmd.Flags().StringSlice("labels", nil, "Labels (comma-separated)")
	updateCmd.Flags().String("priority", "", "Priority name")
	updateCmd.Flags().String("data", "", "Arbitrary JSON string for request body")
	updateCmd.Flags().String("data-file", "", "Path to JSON file for request body (mutually exclusive with --data)")
	updateCmd.Flags().Bool("markdown-description", false, "Treat --description as Markdown (converted to ADF)")
}

func buildUpdateBody(cmd *cobra.Command) (map[string]any, error) {
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
		if assignee == "" {
			fields["assignee"] = nil
		} else {
			fields["assignee"] = map[string]any{"accountId": assignee}
		}
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

	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	body["fields"] = fields
	return body, nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	issueKey := args[0]

	body, err := buildUpdateBody(cmd)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	if err := client.UpdateIssue(ctx, issueKey, body); err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	issue, err := client.GetIssue(ctx, issueKey, []string{"*all"})
	if err != nil {
		return fmt.Errorf("failed to get updated issue: %w", err)
	}

	markdown, _ := cmd.Flags().GetBool("markdown-description")
	if markdown {
		return outputJSON(convertDescriptionToMarkdown(issue))
	}
	return outputJSON(issue)
}
