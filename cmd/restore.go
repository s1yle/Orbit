package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// restoreCmd 恢复备份命令
var restoreCmd = &cobra.Command{
	Use:   "restore [backup.orbit]",
	Short: "Restore configuration from backup file",
	Long: `Restore configuration from a backup .orbit file.

This command will:
- Extract configuration files to their original locations
- Restore VSCode settings and extensions
- Update system configuration with restore statistics

Examples:
  orbit restore backup.orbit
  orbit restore my_config.orbit`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		backupFile := args[0]

		if err := restoreFromBackup(backupFile); err != nil {
			logger.Errorf("恢复失败: %v", err)
			os.Exit(1)
		}

		// 更新配置中的恢复统计
		updateRestoreStats()

		logger.Info("恢复操作完成")
	},
}

// restoreFromBackup 从备份文件恢复配置
func restoreFromBackup(backupFile string) error {
	logger.Infof("正在从 %s 恢复配置", backupFile)

	// 检查备份文件是否存在
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupFile)
	}

	// 打开备份文件
	r, err := zip.OpenReader(backupFile)
	if err != nil {
		return fmt.Errorf("无法打开备份文件: %v", err)
	}
	defer r.Close()

	// 创建临时目录用于解压
	tempDir, err := os.MkdirTemp("", "orbit-restore")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger.Infof("正在解压备份文件到临时目录: %s", tempDir)

	// 解压所有文件到临时目录
	for _, file := range r.File {
		err := extractFileFromZip(file, tempDir)
		if err != nil {
			logger.Warnf("解压文件 %s 失败: %v", file.Name, err)
			continue
		}
	}

	// 恢复VSCode配置
	if err := restoreVSCodeConfig(tempDir); err != nil {
		logger.Warnf("恢复VSCode配置失败: %v", err)
	}

	// 读取并显示manifest信息
	if err := readManifestFromBackup(tempDir); err != nil {
		logger.Warnf("读取manifest信息失败: %v", err)
	}

	logger.Info("备份文件解压完成")

	return nil
}

// extractFileFromZip 从zip文件中提取单个文件
func extractFileFromZip(file *zip.File, destDir string) error {
	// 构建目标路径
	destPath := filepath.Join(destDir, file.Name)

	// 如果是目录，创建目录
	if file.FileInfo().IsDir() {
		return os.MkdirAll(destPath, os.ModePerm)
	}

	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
		return err
	}

	// 打开源文件
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// 创建目标文件
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 复制文件内容
	_, err = io.Copy(destFile, rc)
	return err
}

// restoreVSCodeConfig 恢复VSCode配置
func restoreVSCodeConfig(tempDir string) error {
	logger.Info("正在恢复VSCode配置...")

	configsBase := filepath.Join(tempDir, "configs", "vscode_config_dir")
	if _, err := os.Stat(configsBase); os.IsNotExist(err) {
		return fmt.Errorf("VSCode配置目录不存在")
	}

	// 恢复APPDATA配置
	appdataSource := filepath.Join(configsBase, "APPDATA", "Code")
	appdataDest := filepath.Dir(CodeConfigDir) // 获取APPDATA目录的父目录

	if _, err := os.Stat(appdataSource); err == nil {
		logger.Infof("恢复APPDATA配置: %s -> %s", appdataSource, appdataDest)
		if err := copyDirectory(appdataSource, appdataDest); err != nil {
			logger.Warnf("恢复APPDATA配置失败: %v", err)
		} else {
			logger.Info("APPDATA配置恢复完成")
		}
	}

	// 恢复USER配置
	userSource := filepath.Join(configsBase, "USER", ".vscode")
	userDest := filepath.Dir(CodeUserDir) // 获取用户目录的父目录

	if _, err := os.Stat(userSource); err == nil {
		logger.Infof("恢复USER配置: %s -> %s", userSource, userDest)
		if err := copyDirectory(userSource, userDest); err != nil {
			logger.Warnf("恢复USER配置失败: %v", err)
		} else {
			logger.Info("USER配置恢复完成")
		}
	}

	logger.Info("VSCode配置恢复操作完成")
	return nil
}

// copyDirectory 复制目录及其内容
func copyDirectory(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(destPath, path)
	})
}

// readManifestFromBackup 从备份中读取manifest信息
func readManifestFromBackup(tempDir string) error {
	manifestPath := filepath.Join(tempDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest.json不存在")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("读取manifest.json失败: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("解析manifest.json失败: %v", err)
	}

	logger.Info("备份文件信息:")
	logger.Infof("  备份时间: %s", manifest.Timestamp)
	logger.Infof("  系统: %s", manifest.OS)
	logger.Infof("  架构: %s", manifest.Arch)
	logger.Infof("  主机名: %s", manifest.Hostname)
	logger.Infof("  用户名: %s", manifest.Username)

	return nil
}

// updateRestoreStats 更新恢复统计信息
func updateRestoreStats() {
	configManager := GetConfigManager()
	if configManager == nil || !configManager.IsConfigLoaded() {
		logger.Warnf("配置管理器未初始化，无法更新恢复统计")
		return
	}

	err := configManager.UpdateSystemConfig(func(systemConfig *SystemConfig) {
		// 增加恢复计数
		systemConfig.RestoreCount++
		// 更新最后恢复时间
		systemConfig.LastRestoreTime = time.Now().Format("2006-01-02 15:04:05")
	})

	if err != nil {
		logger.Warnf("更新恢复统计失败: %v", err)
	} else {
		logger.Info("恢复统计已更新")
	}
}

// 在SystemConfig中添加恢复相关的字段
// 注意：我们需要在root.go中更新SystemConfig结构体
// 由于我们已经在之前的建议中提到了，这里假设已经添加了这些字段

func init() {
	rootCmd.AddCommand(restoreCmd)
}
