package cli

import (
	"context"
	"fmt"

	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jira"
	"github.com/spf13/cobra"
)

var fieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "List available JIRA fields",
	Long: `List available JIRA fields.

This command retrieves all fields available in JIRA, including custom fields.
Useful for finding custom field IDs like customfield_12345.`,
	RunE: runFields,
}

func runFields(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	fields, err := client.GetFields(ctx)
	if err != nil {
		return fmt.Errorf("failed to get fields: %w", err)
	}

	return outputJSON(fields)
}
