package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// DesiredSoftware represents software that can be installed
type DesiredSoftware struct {
	Name        string `json:"name"`
	OfficialURL string `json:"officialURL"`
	DownloadURL string `json:"downloadURL"`
	Installer   string `json:"installer"` // exe, msi, etc.
	SilentArgs  string `json:"silentArgs"`
	Version     string `json:"version"`
	Category    string `json:"category"`
}

// SoftwareBlacklist represents software that should not be installed
type SoftwareBlacklist struct {
	Software []string `json:"software"`
	Reason   string   `json:"reason,omitempty"`
}

// SoftwareCatalog represents the collection of available software
type SoftwareCatalog struct {
	Timestamp string            `json:"timestamp"`
	Software  []DesiredSoftware `json:"software"`
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install missing software from catalog",
	Long: `Install software that is in the catalog but not currently installed on the system.
Supports blacklist to exclude specific software from installation.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			// Install specific software
			installSpecificSoftware(args[0])
		} else {
			// Install all missing software
			installMissingSoftware()
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

// installSpecificSoftware installs a specific software by name
func installSpecificSoftware(softwareName string) {
	logger.Infof("Attempting to install: %s", softwareName)

	// Check if already installed
	installed, err := getInstalledSoftware()
	if err != nil {
		logger.Errorf("Failed to get installed software: %v", err)
		return
	}

	for _, sw := range installed {
		if strings.EqualFold(sw.Name, softwareName) {
			logger.Infof("Software '%s' is already installed", softwareName)
			return
		}
	}

	// Check blacklist
	blacklist, _ := loadBlacklist()
	for _, blacklisted := range blacklist.Software {
		if strings.EqualFold(blacklisted, softwareName) {
			logger.Warnf("Software '%s' is in blacklist and will not be installed", softwareName)
			return
		}
	}

	// Load software catalog
	catalog, err := loadSoftwareCatalog()
	if err != nil {
		logger.Errorf("Failed to load software catalog: %v", err)
		return
	}

	// Find the software in catalog
	var targetSoftware *DesiredSoftware
	for _, sw := range catalog.Software {
		if strings.EqualFold(sw.Name, softwareName) {
			targetSoftware = &sw
			break
		}
	}

	if targetSoftware == nil {
		logger.Errorf("Software '%s' not found in catalog", softwareName)
		return
	}

	// Download and install
	if err := downloadAndInstall(*targetSoftware); err != nil {
		logger.Errorf("Failed to install %s: %v", softwareName, err)
	} else {
		logger.Infof("Successfully installed %s", softwareName)
	}
}

// installMissingSoftware installs all software in catalog that's not installed and not blacklisted
func installMissingSoftware() {
	logger.Info("Scanning for missing software to install...")

	// Get installed software
	installed, err := getInstalledSoftware()
	if err != nil {
		logger.Errorf("Failed to get installed software: %v", err)
		return
	}

	// Load blacklist
	blacklist, _ := loadBlacklist()

	// Load software catalog
	catalog, err := loadSoftwareCatalog()
	if err != nil {
		logger.Errorf("Failed to load software catalog: %v", err)
		return
	}

	// Find missing software
	var missingSoftware []DesiredSoftware
	for _, desired := range catalog.Software {
		found := false

		// Check if already installed
		for _, inst := range installed {
			if strings.EqualFold(inst.Name, desired.Name) {
				found = true
				break
			}
		}

		// Check if blacklisted
		for _, blacklisted := range blacklist.Software {
			if strings.EqualFold(blacklisted, desired.Name) {
				found = true // Treat as "found" to skip installation
				logger.Infof("Skipping blacklisted software: %s", desired.Name)
				break
			}
		}

		if !found {
			missingSoftware = append(missingSoftware, desired)
		}
	}

	if len(missingSoftware) == 0 {
		logger.Info("No missing software found. All catalog software is either installed or blacklisted.")
		return
	}

	logger.Infof("Found %d missing software to install", len(missingSoftware))

	// Install missing software
	for _, software := range missingSoftware {
		logger.Infof("Installing: %s", software.Name)
		if err := downloadAndInstall(software); err != nil {
			logger.Errorf("Failed to install %s: %v", software.Name, err)
		} else {
			logger.Infof("Successfully installed %s", software.Name)
		}
	}
}

// downloadAndInstall downloads and installs a software
func downloadAndInstall(software DesiredSoftware) error {
	// Create temp directory for downloads
	tempDir := filepath.Join(os.TempDir(), "orbit-install")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Determine download URL
	downloadURL := software.DownloadURL
	if downloadURL == "" {
		downloadURL = software.OfficialURL
	}

	if downloadURL == "" {
		return fmt.Errorf("no download URL available for %s", software.Name)
	}

	// Download installer
	installerPath := filepath.Join(tempDir, filepath.Base(downloadURL))
	logger.Infof("Downloading %s from %s", software.Name, downloadURL)

	if err := downloadFile(downloadURL, installerPath); err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	// Install the software
	logger.Infof("Installing %s...", software.Name)
	if err := runInstaller(installerPath, software.SilentArgs); err != nil {
		return fmt.Errorf("installation failed: %v", err)
	}

	// Clean up installer
	os.Remove(installerPath)

	return nil
}

// 进度跟踪器，实现io.Writer接口
type ProgressTracker struct {
	TotalSize  int64
	Downloaded int64
	StartTime  time.Time
}

// Write方法用于跟踪下载进度
func (p *ProgressTracker) Write(data []byte) (int, error) {
	n := len(data)
	p.Downloaded += int64(n)
	p.PrintProgress()
	return n, nil
}

// PrintProgress打印当前下载进度信息
func (p *ProgressTracker) PrintProgress() {
	if p.TotalSize == 0 {
		return
	}

	// 计算下载百分比
	percentage := float64(p.Downloaded) / float64(p.TotalSize) * 100

	// 计算已用时间和下载速度
	elapsed := time.Since(p.StartTime)
	speed := float64(p.Downloaded) / elapsed.Seconds()

	// 计算剩余时间
	var remaining time.Duration
	if speed > 0 {
		remaining = time.Duration(float64(p.TotalSize-p.Downloaded) / speed)
	}

	// 格式化显示单位
	var downloadedStr, totalStr, speedStr string
	if p.TotalSize < 1024*1024 {
		downloadedStr = fmt.Sprintf("%.2f KB", float64(p.Downloaded)/1024)
		totalStr = fmt.Sprintf("%.2f KB", float64(p.TotalSize)/1024)
		speedStr = fmt.Sprintf("%.2f KB/s", speed/1024)
	} else {
		downloadedStr = fmt.Sprintf("%.2f MB", float64(p.Downloaded)/(1024*1024))
		totalStr = fmt.Sprintf("%.2f MB", float64(p.TotalSize)/(1024*1024))
		speedStr = fmt.Sprintf("%.2f MB/s", speed/(1024*1024))
	}

	// 打印进度信息，使用\r回到行首覆盖显示
	logger.WithField("no_newline", true).Infof("进度: %.2f%% (%s / %s) 速度: %s 剩余: %s \r",
		percentage, downloadedStr, totalStr, speedStr, remaining.Round(time.Second))
}

// downloadFile downloads a file from URL to local path
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	progress := &ProgressTracker{
		TotalSize: resp.ContentLength,
		StartTime: time.Now(),
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, progress))
	return err
}

func runInstaller(installerPath, silentArgs string) error {
	var cmdArgs []string
	ext := strings.ToLower(filepath.Ext(installerPath))

	// 构建安装命令（与原逻辑一致）
	switch ext {
	case ".msi":
		if silentArgs == "" {
			silentArgs = "/quiet /norestart"
		}
		args := []string{"/i", installerPath}
		args = append(args, strings.Fields(silentArgs)...)
		cmdArgs = append([]string{"msiexec"}, args...)
	case ".exe":
		if silentArgs == "" {
			silentArgs = "/S"
		}
		args := strings.Fields(silentArgs)
		cmdArgs = append([]string{installerPath}, args...)
	default:
		return fmt.Errorf("unsupported installer type: %s", ext)
	}

	// 通过PowerShell以管理员权限执行命令
	// -Command & { ... } 用于执行命令；-Verb RunAs 请求管理员权限
	powershellCmd := fmt.Sprintf("Start-Process -FilePath \"%s\" -ArgumentList \"%s\" -Wait -Verb RunAs",
		cmdArgs[0], strings.Join(cmdArgs[1:], "\" \""))

	cmd := exec.Command("powershell", "-Command", powershellCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("执行失败: %v, 输出: %s", err, string(output))
	}
	return nil
}

// loadSoftwareCatalog loads the software catalog from file
func loadSoftwareCatalog() (*SoftwareCatalog, error) {
	catalogPath := "software-catalog.json"
	if _, err := os.Stat(catalogPath); os.IsNotExist(err) {
		// Create default catalog if it doesn't exist
		return createDefaultCatalog(), nil
	}

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}

	var catalog SoftwareCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}

	return &catalog, nil
}

// loadBlacklist loads the software blacklist from file
func loadBlacklist() (*SoftwareBlacklist, error) {
	blacklistPath := "software-blacklist.json"
	if _, err := os.Stat(blacklistPath); os.IsNotExist(err) {
		// Return empty blacklist if file doesn't exist
		return &SoftwareBlacklist{Software: []string{}}, nil
	}

	data, err := os.ReadFile(blacklistPath)
	if err != nil {
		return nil, err
	}

	var blacklist SoftwareBlacklist
	if err := json.Unmarshal(data, &blacklist); err != nil {
		return nil, err
	}

	return &blacklist, nil
}

// createDefaultCatalog creates a default software catalog with common applications
func createDefaultCatalog() *SoftwareCatalog {
	return &SoftwareCatalog{
		Timestamp: time.Now().Format(time.RFC3339),
		Software: []DesiredSoftware{
			{
				Name:        "Google Chrome",
				OfficialURL: "https://www.google.com/chrome/",
				DownloadURL: "https://dl.google.com/chrome/install/standalone/GoogleChromeStandaloneEnterprise64.msi",
				Installer:   "msi",
				SilentArgs:  "/quiet /norestart",
				Category:    "Browser",
			},
			{
				Name:        "Mozilla Firefox",
				OfficialURL: "https://www.mozilla.org/firefox/",
				DownloadURL: "https://download.mozilla.org/?product=firefox-latest&os=win64&lang=en-US",
				Installer:   "exe",
				SilentArgs:  "/S",
				Category:    "Browser",
			},
			{
				Name:        "Visual Studio Code",
				OfficialURL: "https://code.visualstudio.com/",
				DownloadURL: "https://code.visualstudio.com/sha/download?build=stable&os=win32-x64",
				Installer:   "exe",
				SilentArgs:  "/SILENT /MERGETASKS=!runcode",
				Category:    "Development",
			},
			{
				Name:        "7-Zip",
				OfficialURL: "https://www.7-zip.org/",
				DownloadURL: "https://www.7-zip.org/a/7z2301-x64.exe",
				Installer:   "exe",
				SilentArgs:  "/S",
				Category:    "Utilities",
			},
			{
				Name:        "Notepad++",
				OfficialURL: "https://notepad-plus-plus.org/",
				DownloadURL: "https://github.com/notepad-plus-plus/notepad-plus-plus/releases/download/v8.6.4/npp.8.6.4.Installer.x64.exe",
				Installer:   "exe",
				SilentArgs:  "/S",
				Category:    "Text Editor",
			},
		},
	}
}
