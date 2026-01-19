package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jira"
	"github.com/longkey1/gojira/internal/models"
	"github.com/spf13/cobra"
)

var (
	sumJQL    string
	sumFields []string
)

var sumCmd = &cobra.Command{
	Use:   "sum",
	Short: "Sum custom field values for issues matching JQL",
	Long: `Sum custom field values for issues matching JQL.

Examples:
  # Sum a single custom numeric field
  gojira sum --jql 'project = PROJ AND status = Done' --fields customfield_12345

  # Sum multiple custom numeric fields
  gojira sum --jql 'project = PROJ' --fields customfield_12345,customfield_67890`,
	RunE: runSum,
}

func init() {
	sumCmd.Flags().StringVar(&sumJQL, "jql", "", "JQL query (required)")
	sumCmd.Flags().StringSliceVar(&sumFields, "fields", nil, "Custom fields to sum (comma-separated, required)")
	sumCmd.MarkFlagRequired("jql")
	sumCmd.MarkFlagRequired("fields")
}

type FieldSum struct {
	Field     string  `json:"field"`
	TotalSum  float64 `json:"totalSum"`
	SkipCount int     `json:"skipCount"`
}

type SumResult struct {
	JQL        string     `json:"jql"`
	FieldSums  []FieldSum `json:"fieldSums"`
	IssueCount int        `json:"issueCount"`
}

func runSum(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	// Build fields list: summary and all requested custom fields
	fields := []string{"summary"}
	fields = append(fields, sumFields...)

	issues, err := client.SearchJQLAll(ctx, sumJQL, fields)
	if err != nil {
		return fmt.Errorf("failed to search issues: %w", err)
	}

	// Calculate sum for each field
	result := SumResult{
		JQL:        sumJQL,
		FieldSums:  make([]FieldSum, 0, len(sumFields)),
		IssueCount: len(issues),
	}

	for _, fieldName := range sumFields {
		fieldSum := FieldSum{
			Field: fieldName,
		}

		for _, issue := range issues {
			value, ok := getCustomFieldNumericValue(issue, fieldName)
			if !ok {
				fieldSum.SkipCount++
				continue
			}

			fieldSum.TotalSum += value
		}

		if fieldSum.SkipCount > 0 {
			fmt.Fprintf(os.Stderr, "Warning: %d issues skipped for field %s (non-numeric or null values)\n", fieldSum.SkipCount, fieldName)
		}

		result.FieldSums = append(result.FieldSums, fieldSum)
	}

	return outputJSON(result)
}

func getCustomFieldNumericValue(issue models.Issue, fieldName string) (float64, bool) {
	val, exists := issue.Fields.CustomFields[fieldName]
	if !exists || val == nil {
		return 0, false
	}

	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
