package cmd

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jira"
	"github.com/spf13/cobra"
)

var validSprintStates = []string{"future", "active", "closed"}

var (
	sprintListBoard int
	sprintListState string
)

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "Manage board sprints",
}

var sprintListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sprints of a board",
	Long: `List all sprints of a board via the Jira Agile API.

Examples:
  # List all sprints of a board
  gojira sprint list --board 7417

  # List only active and closed sprints
  gojira sprint list --board 7417 --state active,closed

  # Use the default board (GOJIRA_DEFAULT_BOARD or default_board in config file)
  gojira sprint list --state active`,
	Args: cobra.NoArgs,
	RunE: runSprintList,
}

func init() {
	sprintCmd.AddCommand(sprintListCmd)

	sprintListCmd.Flags().IntVar(&sprintListBoard, "board", 0, "Board ID (default: GOJIRA_DEFAULT_BOARD or 'default_board' in config file)")
	sprintListCmd.Flags().StringVar(&sprintListState, "state", "", "Sprint states to include (comma-separated: future,active,closed; default: all)")
}

// resolveBoardID returns the --board value if given, otherwise the configured default board.
func resolveBoardID(flagSet bool, flagValue, defaultBoard int) (int, error) {
	if flagSet {
		if flagValue <= 0 {
			return 0, fmt.Errorf("invalid --board %d (must be a positive integer)", flagValue)
		}
		return flagValue, nil
	}
	if defaultBoard > 0 {
		return defaultBoard, nil
	}
	return 0, errors.New("board ID is required (use --board, or set GOJIRA_DEFAULT_BOARD or 'default_board' in config file)")
}

func parseSprintStates(s string) ([]string, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var states []string
	for st := range strings.SplitSeq(s, ",") {
		st = strings.TrimSpace(st)
		if !slices.Contains(validSprintStates, st) {
			return nil, fmt.Errorf("invalid sprint state %q (valid: %s)", st, strings.Join(validSprintStates, ", "))
		}
		states = append(states, st)
	}
	return states, nil
}

func runSprintList(cmd *cobra.Command, args []string) error {
	states, err := parseSprintStates(sprintListState)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	boardID, err := resolveBoardID(cmd.Flags().Changed("board"), sprintListBoard, cfg.DefaultBoard)
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)

	sprints, err := client.ListSprints(context.Background(), boardID, states)
	if err != nil {
		return fmt.Errorf("failed to list sprints: %w", err)
	}

	return outputJSON(sprints)
}
