package cmd

import (
	"github.com/spf13/cobra"

	"github.com/longkey1/gojira/internal/config"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "gojira",
	Short: "A Go-based JIRA integration tool",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.SetConfigFile(configFile)
	},
	SilenceErrors: true,
	SilenceUsage:  true,
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
