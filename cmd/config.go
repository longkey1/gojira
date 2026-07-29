package cmd

import (
	"fmt"
	"strings"

	"github.com/longkey1/gojira/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration values",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a single configuration value",
	Long: fmt.Sprintf(`Get a single configuration value resolved from the config file and
environment variables.

Valid keys: %s

Examples:
  gojira config get jira_base_url
  gojira config get jira_email`, strings.Join(config.Keys(), ", ")),
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

func init() {
	configCmd.AddCommand(configGetCmd)
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	value, err := config.Get(args[0])
	if err != nil {
		return err
	}

	fmt.Println(value)
	return nil
}
