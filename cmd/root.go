package cmd

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var logger *logrus.Logger

// VSCode的配置目录
var CodeConfigDir string = filepath.Join(os.Getenv("APPDATA"), "Code")

// 当前目录的绝对路径
var CurrentDir, _ = filepath.Abs(".")

type Manifest struct {
	Timestamp string `json:"timestamp"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
}

var rootCmd = &cobra.Command{
	Use:   "orbit",
	Short: "Orbit is a backup and restore tool for software configurations",
	Long: `Orbit helps you backup and restore your software configurations
and installed software lists across different systems.`,
	Version: "0.0.0.1",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Show help if no subcommand is provided
		cmd.Help()
	},
}

func Execute(log *logrus.Logger) {
	logger = log

	if err := rootCmd.Execute(); err != nil {

		logger.Info(err)
		os.Exit(1)
	}
}
