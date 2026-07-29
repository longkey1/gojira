package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/longkey1/gojira/internal/config"
)

var configFile string

// writeAnnotation marks a command as mutating JIRA state
// (see create.go, update.go, comment.go).
const writeAnnotation = "write"

var rootCmd = &cobra.Command{
	Use:   "gojira",
	Short: "A Go-based JIRA integration tool",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		config.SetConfigFile(configFile)
		if isWriteCommand(cmd) {
			return checkReadOnly()
		}
		return nil
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

// isWriteCommand returns true if cmd (or an ancestor) is annotated as a write command.
func isWriteCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations[writeAnnotation] == "true" {
			return true
		}
	}
	return false
}

// checkReadOnly returns an error if read-only mode is enabled.
func checkReadOnly() error {
	readOnly, err := config.ReadOnly()
	if err != nil {
		return err
	}
	if readOnly {
		return fmt.Errorf("read-only mode is enabled (read_only/GOJIRA_READ_ONLY); write commands are disabled")
	}
	return nil
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to TOML config file (default: $XDG_CONFIG_HOME/gojira/config.toml or $HOME/.config/gojira/config.toml)")

	rootCmd.AddCommand(commentCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(fieldsCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(mergeCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(versionCmd)
}
