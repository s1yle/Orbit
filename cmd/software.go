package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

// getInstalledSoftware retrieves installed software from Windows registry and WMI
func getInstalledSoftware() ([]Software, error) {
	var softwareList []Software

	// Get software from registry (traditional Windows programs)
	registrySoftware, err := getSoftwareFromRegistry()
	if err != nil {
		logger.Warnf("Failed to get software from registry: %v", err)
	} else {
		softwareList = append(softwareList, registrySoftware...)
	}

	// Get software from WMI (modern Windows Store apps and programs)
	wmiSoftware, err := getSoftwareFromWMI()
	if err != nil {
		logger.Warnf("Failed to get software from WMI: %v", err)
	} else {
		softwareList = append(softwareList, wmiSoftware...)
	}

	// Remove duplicates and filter out system components
	softwareList = filterSoftwareList(softwareList)

	logger.Infof("Found %d installed software applications", len(softwareList))
	return softwareList, err
}

// getSoftwareFromRegistry retrieves installed software from Windows registry
func getSoftwareFromRegistry() ([]Software, error) {
	var software []Software

	// Registry paths to check for installed software
	registryPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	for _, regPath := range registryPaths {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, regPath, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		defer k.Close()

		subkeys, err := k.ReadSubKeyNames(-1)
		if err != nil {
			continue
		}

		for _, subkey := range subkeys {
			sk, err := registry.OpenKey(k, subkey, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			displayName, _, err := sk.GetStringValue("DisplayName")
			if err != nil {
				sk.Close()
				continue
			}

			// Skip system components and updates
			if shouldSkipSoftware(displayName) {
				sk.Close()
				continue
			}

			softwareItem := Software{
				Name:   displayName,
				Source: "registry",
			}

			// Get optional fields
			if version, _, err := sk.GetStringValue("DisplayVersion"); err == nil {
				softwareItem.Version = version
			}

			if publisher, _, err := sk.GetStringValue("Publisher"); err == nil {
				softwareItem.Publisher = publisher
			}

			if installDate, _, err := sk.GetStringValue("InstallDate"); err == nil {
				softwareItem.InstallDate = installDate
			}

			if installLocation, _, err := sk.GetStringValue("InstallLocation"); err == nil {
				softwareItem.InstallPath = installLocation
			}

			if uninstallString, _, err := sk.GetStringValue("UninstallString"); err == nil {
				softwareItem.Uninstall = uninstallString
			}

			software = append(software, softwareItem)
			sk.Close()
		}
	}

	return software, nil
}

// getSoftwareFromWMI retrieves installed software using Windows Management Instrumentation
func getSoftwareFromWMI() ([]Software, error) {
	var software []Software

	// Use PowerShell to query WMI for installed software
	cmd := exec.Command("powershell", "-Command",
		"Get-WmiObject -Class Win32_Product | Select-Object Name, Version, Vendor, InstallDate | ConvertTo-Json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query WMI: %v", err)
	}

	// Parse the JSON output from PowerShell
	var wmiResults []struct {
		Name        string `json:"Name"`
		Version     string `json:"Version"`
		Vendor      string `json:"Vendor"`
		InstallDate string `json:"InstallDate"`
	}

	if err := json.Unmarshal(output, &wmiResults); err != nil {
		// Try parsing as single object (if only one result)
		var singleResult struct {
			Name        string `json:"Name"`
			Version     string `json:"Version"`
			Vendor      string `json:"Vendor"`
			InstallDate string `json:"InstallDate"`
		}
		if err := json.Unmarshal(output, &singleResult); err == nil && singleResult.Name != "" {
			wmiResults = []struct {
				Name        string `json:"Name"`
				Version     string `json:"Version"`
				Vendor      string `json:"Vendor"`
				InstallDate string `json:"InstallDate"`
			}{singleResult}
		} else {
			return nil, fmt.Errorf("failed to parse WMI output: %v", err)
		}
	}

	for _, item := range wmiResults {
		if shouldSkipSoftware(item.Name) {
			continue
		}

		softwareItem := Software{
			Name:        item.Name,
			Version:     item.Version,
			Publisher:   item.Vendor,
			InstallDate: formatWMIDate(item.InstallDate),
			Source:      "wmi",
		}

		software = append(software, softwareItem)
	}

	return software, nil
}

// shouldSkipSoftware determines if a software should be excluded from the list
func shouldSkipSoftware(name string) bool {
	skipPatterns := []string{
		"Microsoft Visual C++",
		"Microsoft .NET",
		"Windows SDK",
		"Update for",
		"Security Update",
		"Hotfix",
		"Service Pack",
		"KB",
		"Runtime",
		"Redistributable",
		"System Component",
	}

	nameLower := strings.ToLower(name)
	for _, pattern := range skipPatterns {
		if strings.Contains(nameLower, strings.ToLower(pattern)) {
			return true
		}
	}

	// Skip very short names (likely system components)
	if len(strings.TrimSpace(name)) < 3 {
		return true
	}

	return false
}

// filterSoftwareList removes duplicates and cleans up the software list
func filterSoftwareList(software []Software) []Software {
	seen := make(map[string]bool)
	var filtered []Software

	for _, item := range software {
		// Create a unique key for the software
		key := strings.ToLower(strings.TrimSpace(item.Name))
		if key == "" {
			continue
		}

		if !seen[key] {
			seen[key] = true
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// formatWMIDate converts WMI date format to readable format
func formatWMIDate(wmiDate string) string {
	if len(wmiDate) < 8 {
		return wmiDate
	}

	// WMI date format: YYYYMMDDHHMMSS.FFFFFF+TZONE
	dateStr := wmiDate[:8]
	year := dateStr[:4]
	month := dateStr[4:6]
	day := dateStr[6:8]

	return fmt.Sprintf("%s-%s-%s", year, month, day)
}

// saveSoftwareList creates software-list.json file in the temp directory
func saveSoftwareList(tempDir string) error {
	logger.Info("正在扫描系统已安装的软件...")

	software, err := getInstalledSoftware()
	if err != nil {
		return fmt.Errorf("获取已安装软件失败: %v", err)
	}

	softwareList := SoftwareList{
		Timestamp:  time.Now().Format(time.RFC3339),
		TotalCount: len(software),
		Software:   software,
	}

	jsonData, err := json.MarshalIndent(softwareList, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化软件列表失败: %v", err)
	}

	softwareListPath := filepath.Join(tempDir, "software-list.json")
	if err := os.WriteFile(softwareListPath, jsonData, 0644); err != nil {
		return fmt.Errorf("写入软件列表文件失败: %v", err)
	}

	logger.Infof("成功保存 %d 个软件信息到 software-list.json", len(software))
	return nil
}
