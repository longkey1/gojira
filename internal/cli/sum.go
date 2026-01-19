package cli

import (
	"context"
	"fmt"

	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jira"
	"github.com/longkey1/gojira/internal/models"
	"github.com/spf13/cobra"
)

var (
	sumJQL   string
	sumField string
)

var sumCmd = &cobra.Command{
	Use:   "sum",
	Short: "Sum a custom field value for issues matching JQL",
	Long: `Sum a custom field value for issues matching JQL.

Examples:
  # Sum story points for child issues of an epic
  gojira sum --jql 'parent = EPIC-123' --field customfield_12345

  # Sum a field for issues in a project
  gojira sum --jql 'project = PROJ AND status = Done' --field customfield_12345`,
	RunE: runSum,
}

func init() {
	sumCmd.Flags().StringVar(&sumJQL, "jql", "", "JQL query (required)")
	sumCmd.Flags().StringVar(&sumField, "field", "", "Custom field to sum (required, e.g., customfield_12345)")
	sumCmd.MarkFlagRequired("jql")
	sumCmd.MarkFlagRequired("field")
}

type SumResult struct {
	JQL        string             `json:"jql"`
	Field      string             `json:"field"`
	TotalSum   float64            `json:"totalSum"`
	ByStatus   map[string]float64 `json:"byStatus"`
	IssueCount int                `json:"issueCount"`
}

func runSum(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	fields := []string{"summary", "status", sumField}

	issues, err := client.SearchJQLAll(ctx, sumJQL, fields)
	if err != nil {
		return fmt.Errorf("failed to search issues: %w", err)
	}

	// Calculate sum
	result := SumResult{
		JQL:        sumJQL,
		Field:      sumField,
		ByStatus:   make(map[string]float64),
		IssueCount: len(issues),
	}

	for _, issue := range issues {
		value := getCustomFieldValue(issue, sumField)
		if value != 0 {
			result.TotalSum += value

			statusName := "Unknown"
			if issue.Fields.Status != nil {
				statusName = issue.Fields.Status.Name
			}
			result.ByStatus[statusName] += value
		}
	}

	return outputJSON(result)
}

func getCustomFieldValue(issue models.Issue, fieldName string) float64 {
	switch fieldName {
	case "customfield_12345":
		if issue.Fields.StoryPoints != nil {
			return *issue.Fields.StoryPoints
		}
	}
	return 0
}
