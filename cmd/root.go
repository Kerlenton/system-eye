package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command for the CLI application.
var rootCmd = &cobra.Command{
	Use:   "system-eye",
	Short: "CLI utility for system monitoring",
	Long:  `CLI tool that monitors system metrics (CPU, memory, disk) and displays them as live or historical ASCII graphs.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Display help if no subcommand is provided.
		cmd.Help()
	},
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
