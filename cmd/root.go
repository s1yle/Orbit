package cmd

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var logger *logrus.Logger

// 全局版本变量
var Version string = "0.0.2.1"

// VSCode在 APPDATA 中的配置目录
var CodeConfigDir string = filepath.Join(os.Getenv("APPDATA"), "Code") //C:\Users\mmili985\AppData\Roaming\Code

// VSCode在 User 中的目录
var CodeUserDir string = filepath.Join(os.Getenv("USERPROFILE"), ".vscode") //C:\Users\mmili985\.vscode

// 当前目录的绝对路径
var CurrentDir, _ = filepath.Abs(".")

var EncryptedVerStr string = "ORBIT_ENCRYPTED_v" + Version + "\n"

type Manifest struct {
	Timestamp string `json:"timestamp"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
}

// Software represents an installed software application
type Software struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Publisher   string `json:"publisher,omitempty"`
	InstallDate string `json:"installDate,omitempty"`
	InstallPath string `json:"installPath,omitempty"`
	Uninstall   string `json:"uninstall,omitempty"`
	Source      string `json:"source,omitempty"`
}

// SoftwareList represents the collection of installed software
type SoftwareList struct {
	Timestamp  string     `json:"timestamp"`
	TotalCount int        `json:"totalCount"`
	Software   []Software `json:"software"`
}

type ConfigDirType struct {
	Name         string
	Path         string
	OriginalPath string
}

var rootCmd = &cobra.Command{
	Use:   "orbit",
	Short: "Orbit is a backup and restore tool for software configurations",
	Long: `Orbit helps you backup and restore your software configurations
and installed software lists across different systems.`,
	Version: Version,
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
