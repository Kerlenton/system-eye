package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "system-eye",
	Short: "CLI utility for system monitoring",
	Long:  `CLI tool that monitors system metrics (CPU, memory, disk) and displays them as live ASCII graphs.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
