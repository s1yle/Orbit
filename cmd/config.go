package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// configSetCmd 配置设置命令
var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set configuration value",
	Long: `Set configuration values for Orbit backup tool.

Available configuration keys:
- backup-path: Default backup directory path
- encryption-enabled: Enable/disable encryption (true/false)
- public-key-path: Path to public key file
- private-key-path: Path to private key file
- include-store-apps: Include Windows Store apps in software list (true/false)
- auto-update-list: Automatically update software list (true/false)
- backup-setting: Enable VSCode backup (true/false)

Examples:
  orbit config set backup-path "D:\backups"
  orbit config set encryption-enabled true
  orbit config set public-key-path "./my_public_key.pem"`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		configManager := GetConfigManager()
		if configManager == nil || !configManager.IsConfigLoaded() {
			logger.Errorf("配置管理器未初始化")
			return
		}

		err := updateConfigValue(configManager, key, value)
		if err != nil {
			logger.Errorf("设置配置失败: %v", err)
			return
		}

		logger.Infof("配置已更新: %s = %s", key, value)
	},
}

// configValidateCmd 配置验证命令
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate current configuration",
	Long:  `Validate the current configuration for errors and inconsistencies.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		configManager := GetConfigManager()
		if configManager == nil || !configManager.IsConfigLoaded() {
			logger.Errorf("配置管理器未初始化")
			return
		}

		config := configManager.GetConfig()
		if config == nil {
			logger.Errorf("无法获取配置")
			return
		}

		issues := validateConfiguration(config)
		if len(issues) == 0 {
			logger.Info("配置验证通过 - 所有配置项都有效")
		} else {
			logger.Warnf("配置验证发现 %d 个问题:", len(issues))
			for _, issue := range issues {
				logger.Warnf("  - %s", issue)
			}
		}
	},
}

// configShowCmd 显示配置命令
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration values.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		configManager := GetConfigManager()
		if configManager == nil || !configManager.IsConfigLoaded() {
			logger.Errorf("配置管理器未初始化")
			return
		}

		config := configManager.GetConfig()
		if config == nil {
			logger.Errorf("无法获取配置")
			return
		}

		logger.Info("当前配置:")
		logger.Infof("  系统配置:")
		logger.Infof("    - 最后备份时间: %s", config.System.LastBackupTime)
		logger.Infof("    - 备份次数: %d", config.System.BackupCount)
		logger.Infof("    - 默认备份路径: %s", config.System.DefaultBackupPath)

		logger.Infof("  VSCode配置:")
		logger.Infof("    - 备份设置: %v", config.VSCode.BackupSetting)
		logger.Infof("    - 排除扩展: %v", config.VSCode.ExcludedExtensions)
		logger.Infof("    - 配置目录数量: %d", len(config.VSCode.ConfigDirs))

		logger.Infof("  软件配置:")
		logger.Infof("    - 包含商店应用: %v", config.Software.IncludeStoreApps)
		logger.Infof("    - 自动更新列表: %v", config.Software.AutoUpdateList)
		logger.Infof("    - 排除模式: %v", config.Software.ExcludedPatterns)

		logger.Infof("  加密配置:")
		logger.Infof("    - 启用加密: %v", config.Encryption.Enabled)
		logger.Infof("    - 公钥路径: %s", config.Encryption.PublicKeyPath)
		logger.Infof("    - 私钥路径: %s", config.Encryption.PrivateKeyPath)
		logger.Infof("    - 默认算法: %s", config.Encryption.DefaultAlgorithm)

		logger.Infof("  最后更新时间: %s", config.LastUpdate)
	},
}

// configRepairCmd 配置修复命令
var configRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair configuration issues",
	Long:  `Automatically repair common configuration issues.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		configManager := GetConfigManager()
		if configManager == nil || !configManager.IsConfigLoaded() {
			logger.Errorf("配置管理器未初始化")
			return
		}

		repairCount, err := repairConfiguration(configManager)
		if err != nil {
			logger.Errorf("配置修复失败: %v", err)
			return
		}

		if repairCount == 0 {
			logger.Info("配置修复完成 - 未发现需要修复的问题")
		} else {
			logger.Infof("配置修复完成 - 修复了 %d 个问题", repairCount)
		}
	},
}

// configCmd 配置管理主命令
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Orbit configuration",
	Long:  `Manage configuration settings for Orbit backup tool.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// updateConfigValue 更新配置值
func updateConfigValue(configManager *ConfigManager, key string, value string) error {
	return configManager.UpdateConfig(func(config *UserConfig) {
		switch strings.ToLower(key) {
		case "backup-path":
			config.System.DefaultBackupPath = value
		case "encryption-enabled":
			config.Encryption.Enabled = (value == "true" || value == "1" || value == "yes")
		case "public-key-path":
			config.Encryption.PublicKeyPath = value
		case "private-key-path":
			config.Encryption.PrivateKeyPath = value
		case "include-store-apps":
			config.Software.IncludeStoreApps = (value == "true" || value == "1" || value == "yes")
		case "auto-update-list":
			config.Software.AutoUpdateList = (value == "true" || value == "1" || value == "yes")
		case "backup-setting":
			config.VSCode.BackupSetting = (value == "true" || value == "1" || value == "yes")
		default:
			logger.Warnf("未知的配置键: %s", key)
		}
	})
}

// validateConfiguration 验证配置
func validateConfiguration(config *UserConfig) []string {
	var issues []string

	// 验证系统配置
	if config.System.DefaultBackupPath == "" {
		issues = append(issues, "默认备份路径为空")
	} else if _, err := os.Stat(config.System.DefaultBackupPath); os.IsNotExist(err) {
		issues = append(issues, fmt.Sprintf("默认备份路径不存在: %s", config.System.DefaultBackupPath))
	}

	// 验证VSCode配置
	if len(config.VSCode.ConfigDirs) == 0 {
		issues = append(issues, "VSCode配置目录为空")
	} else {
		for _, dir := range config.VSCode.ConfigDirs {
			if _, err := os.Stat(dir.Path); os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("VSCode配置目录不存在: %s", dir.Path))
			}
		}
	}

	// 验证加密配置
	if config.Encryption.Enabled {
		if config.Encryption.PublicKeyPath == "" {
			issues = append(issues, "启用加密但公钥路径为空")
		} else if _, err := os.Stat(config.Encryption.PublicKeyPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("公钥文件不存在: %s", config.Encryption.PublicKeyPath))
		}

		if config.Encryption.PrivateKeyPath == "" {
			issues = append(issues, "启用加密但私钥路径为空")
		} else if _, err := os.Stat(config.Encryption.PrivateKeyPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("私钥文件不存在: %s", config.Encryption.PrivateKeyPath))
		}
	}

	return issues
}

// repairConfiguration 修复配置问题
func repairConfiguration(configManager *ConfigManager) (int, error) {
	repairCount := 0

	err := configManager.UpdateConfig(func(config *UserConfig) {
		// 修复默认备份路径
		if config.System.DefaultBackupPath == "" {
			config.System.DefaultBackupPath = CurrentDir
			repairCount++
			logger.Info("修复: 设置默认备份路径为当前目录")
		}

		// 修复VSCode配置目录
		if len(config.VSCode.ConfigDirs) == 0 {
			config.VSCode.ConfigDirs = []ConfigDirType{
				{
					Name:         "APPDATA",
					Path:         CodeConfigDir,
					OriginalPath: "%APPDATA%\\Code",
				},
				{
					Name:         "USER",
					Path:         CodeUserDir,
					OriginalPath: "%USERPROFILE%\\.vscode",
				},
			}
			repairCount++
			logger.Info("修复: 重新设置VSCode配置目录")
		}

		// 修复加密配置不一致
		if config.Encryption.Enabled && (config.Encryption.PublicKeyPath == "" || config.Encryption.PrivateKeyPath == "") {
			config.Encryption.Enabled = false
			repairCount++
			logger.Warn("修复: 禁用加密（密钥路径缺失）")
		}

		// 修复密钥文件路径
		if config.Encryption.Enabled {
			// 检查公钥文件是否存在
			if _, err := os.Stat(config.Encryption.PublicKeyPath); os.IsNotExist(err) {
				// 尝试在当前目录查找
				possiblePaths := []string{
					filepath.Join(CurrentDir, "public.pem"),
					filepath.Join(CurrentDir, getWinUserName()+"_public_key.pem"),
					filepath.Join(filepath.Dir(configManager.configPath), "keys", "public.pem"),
				}

				for _, path := range possiblePaths {
					if _, err := os.Stat(path); err == nil {
						config.Encryption.PublicKeyPath = path
						repairCount++
						logger.Infof("修复: 找到公钥文件 %s", path)
						break
					}
				}
			}
		}
	})

	return repairCount, err
}

func init() {
	// 添加子命令到配置主命令
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configRepairCmd)

	// 添加配置命令到根命令
	rootCmd.AddCommand(configCmd)
}
