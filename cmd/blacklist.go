package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var blacklistCmd = &cobra.Command{
	Use:   "blacklist",
	Short: "Manage software blacklist",
	Long: `Manage the list of software that should not be installed by Orbit.
You can add, remove, or list blacklisted software.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show blacklist status by default
		listBlacklist()
	},
}

var blacklistAddCmd = &cobra.Command{
	Use:   "add [software]",
	Short: "Add software to blacklist",
	Long:  `Add a software to the blacklist to prevent it from being installed.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		addToBlacklist(args[0])
	},
}

var blacklistRemoveCmd = &cobra.Command{
	Use:   "remove [software]",
	Short: "Remove software from blacklist",
	Long:  `Remove a software from the blacklist, allowing it to be installed.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		removeFromBlacklist(args[0])
	},
}

var blacklistListCmd = &cobra.Command{
	Use:   "list",
	Short: "List blacklisted software",
	Long:  `Show all software currently in the blacklist.`,
	Run: func(cmd *cobra.Command, args []string) {
		listBlacklist()
	},
}

func init() {
	rootCmd.AddCommand(blacklistCmd)
	blacklistCmd.AddCommand(blacklistAddCmd)
	blacklistCmd.AddCommand(blacklistRemoveCmd)
	blacklistCmd.AddCommand(blacklistListCmd)
}

// addToBlacklist adds a software to the blacklist
func addToBlacklist(softwareName string) {
	blacklist, err := loadBlacklist()
	if err != nil {
		logger.Errorf("Failed to load blacklist: %v", err)
		return
	}

	// Check if already in blacklist
	for _, item := range blacklist.Software {
		if strings.EqualFold(item, softwareName) {
			logger.Infof("Software '%s' is already in blacklist", softwareName)
			return
		}
	}

	blacklist.Software = append(blacklist.Software, softwareName)

	if err := saveBlacklist(blacklist); err != nil {
		logger.Errorf("Failed to save blacklist: %v", err)
		return
	}

	logger.Infof("Added '%s' to blacklist", softwareName)
}

// removeFromBlacklist removes a software from the blacklist
func removeFromBlacklist(softwareName string) {
	blacklist, err := loadBlacklist()
	if err != nil {
		logger.Errorf("Failed to load blacklist: %v", err)
		return
	}

	// Find and remove the software
	found := false
	var newList []string
	for _, item := range blacklist.Software {
		if !strings.EqualFold(item, softwareName) {
			newList = append(newList, item)
		} else {
			found = true
		}
	}

	if !found {
		logger.Warnf("Software '%s' not found in blacklist", softwareName)
		return
	}

	blacklist.Software = newList

	if err := saveBlacklist(blacklist); err != nil {
		logger.Errorf("Failed to save blacklist: %v", err)
		return
	}

	logger.Infof("Removed '%s' from blacklist", softwareName)
}

// listBlacklist shows all blacklisted software
func listBlacklist() {
	blacklist, err := loadBlacklist()
	if err != nil {
		logger.Errorf("Failed to load blacklist: %v", err)
		return
	}

	if len(blacklist.Software) == 0 {
		logger.Info("No software in blacklist")
		return
	}

	logger.Info("Blacklisted software:")
	for i, software := range blacklist.Software {
		logger.Infof("  %d. %s", i+1, software)
	}
}

// saveBlacklist saves the blacklist to file
func saveBlacklist(blacklist *SoftwareBlacklist) error {
	blacklistPath := "software-blacklist.json"

	data, err := json.MarshalIndent(blacklist, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal blacklist: %v", err)
	}

	if err := os.WriteFile(blacklistPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write blacklist file: %v", err)
	}

	return nil
}
