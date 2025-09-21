package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "orbit",
	Short: "Orbit is a backup and restore tool for software configurations",
	Long: `Orbit helps you backup and restore your software configurations
and installed software lists across different systems.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show help if no subcommand is provided
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
