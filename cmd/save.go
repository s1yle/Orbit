package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var (
	publicKeyPath string
)

// 保存vscode相关配置扩展文件
func saveVscode(tempDir string) error {
	logger.Infof("正在保存Vscode配置文件...")

	configs_base := filepath.Join(tempDir, "configs")
	configs_base_vscode_base := filepath.Join(tempDir, "configs", "vscode_config_dir")
	configs_base_vscode_base_APPDATA := filepath.Join(tempDir, "configs", "vscode_config_dir", "APPDATA")
	configs_base_vscode_base_USER := filepath.Join(tempDir, "configs", "vscode_config_dir", "USER")

	UsersFile := filepath.Join(CodeConfigDir, "User")
	WorkspacesFile := filepath.Join(CodeConfigDir, "Workspaces")
	ConfigsInUser := filepath.Join(CodeUserDir)
	dirs := []ConfigDirType{ConfigDirType{"APPDATA", UsersFile, CodeConfigDir},
		ConfigDirType{"APPDATA", WorkspacesFile, CodeConfigDir},
		ConfigDirType{"USER", ConfigsInUser, CodeUserDir}}

	err := os.MkdirAll(filepath.Join(configs_base), 0644)
	if err != nil {
		return err
	}
	err = os.MkdirAll(configs_base_vscode_base, 0644)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		// var workingDir string
		var extractPath string
		switch dir.Name {
		case "APPDATA":
			workingDir := filepath.Base(dir.Path)
			logger.Infof("dir.Path: %s", dir.Path)

			extractPath = filepath.Join(configs_base_vscode_base_APPDATA, "Code", workingDir)
			logger.Infof("extractPath: %s", extractPath)

			err = os.MkdirAll(extractPath, 0644)
			if err != nil {
				return err
			}
		case "USER":
			workingDir := filepath.Base(dir.Path)
			logger.Infof("dir.Path: %s", dir.Path)

			extractPath = filepath.Join(configs_base_vscode_base_USER, workingDir)
			logger.Infof("extractPath: %s", extractPath)

			err = os.MkdirAll(extractPath, 0644)
			if err != nil {
				return err
			}
		}

		if err != nil {
			return err
		}

		filepath.Walk(dir.Path, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(dir.Path, path) // 获取相对路径
			if err != nil {
				return err
			}

			dstPath := filepath.Join(extractPath, relPath) // 目标路径
			logger.Infof("正在处理文件: %s.  目录: (%s)", filepath.Base(dstPath), filepath.Dir(dstPath))

			if info.IsDir() {
				err = os.MkdirAll(dstPath, os.ModePerm) // 创建目录
				if err != nil {
					return err
				}
			} else {
				err = copyFile(dstPath, path) // 复制文件
				if err != nil {
					return err
				}
			}
			return nil
		})
	}

	return nil
}

// 获取系统信息到manifest 中并转换为 byte array
func convertManifestToJson() ([]byte, error) {
	var jsonData []byte // 获取系统信息

	hostname, err := os.Hostname()
	if err != nil {
		logger.Infof("获取主机名失败: %v", err)
		return jsonData, err
	}

	// 获取当前用户名
	// 注意：os.UserHomeDir() 不能直接获取用户名，这里使用另一种方式
	username := getWinUserName()

	// 创建manifest.json文件
	var manifestContent Manifest = Manifest{
		Timestamp: time.Now().Format(time.RFC3339),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Hostname:  hostname,
		Username:  username,
	}

	jsonData, err = json.MarshalIndent(manifestContent, "", "  ")
	if err != nil {
		return jsonData, err
	}

	return jsonData, nil
}

// createOrbitZipInMemory creates the orbit zip file in memory and returns the bytes
func createOrbitZipInMemory(tempDir string) ([]byte, error) {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	logger.Info("---  正在将文件写入 orbit包")
	err := filepath.Walk(tempDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relpath, err := filepath.Rel(tempDir, path)

		if relpath == "." {
			return nil
		}

		if err != nil {
			return err
		}

		// 如果是目录，创建目录条目并返回
		if info.IsDir() {
			_, err := zipWriter.Create(relpath + "/")
			if err != nil {
				return err
			}
			return nil
		}

		// 如果是文件，创建文件条目并复制内容
		zipFile, err := zipWriter.Create(relpath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFile, file)
		return err
	})

	if err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func createBackup() error {
	// @param tempDir C:\Users\mmili\AppData\Local\Temp\test_orbit-backup
	tempDir, err := os.MkdirTemp("", "test_orbit-backup")
	logger.Infof("成功创建临时目录：%v", tempDir)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	//保存vscode配置文件
	if err := saveVscode(tempDir); err != nil {
		return err
	}

	//获取系统信息写入进manifest.json
	jsonData, err := convertManifestToJson()
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(tempDir, "manifest.json")
	if err := os.WriteFile(manifestPath, jsonData, 0644); err != nil {
		return err
	}
	logger.Info("创建manifest.json文件")

	//保存已安装软件列表到software-list.json
	if err := saveSoftwareList(tempDir); err != nil {
		logger.Warnf("保存软件列表失败: %v", err)
		// Continue with backup even if software list fails
	}

	// Create zip in memory
	zipData, err := createOrbitZipInMemory(tempDir)
	if err != nil {
		return err
	}

	// 使用配置管理器获取加密配置
	configManager := GetConfigManager()
	var useEncryption bool
	var encryptionPublicKeyPath string

	if configManager != nil && configManager.IsConfigLoaded() {
		encryptionConfig := configManager.GetEncryptionConfig()
		useEncryption = encryptionConfig.Enabled
		encryptionPublicKeyPath = encryptionConfig.PublicKeyPath
	} else {
		// 回退到原来的逻辑
		useEncryption = publicKeyPath != ""
		encryptionPublicKeyPath = publicKeyPath
	}

	// Handle encryption if enabled
	if useEncryption && encryptionPublicKeyPath != "" {
		logger.Infof("使用公钥加密备份文件: %s", encryptionPublicKeyPath)

		// Load public key
		publicKey, err := LoadPublicKey(encryptionPublicKeyPath)
		if err != nil {
			return fmt.Errorf("加载公钥失败: %v", err)
		}

		// Encrypt the backup data
		encryptedSymmetricKey, encryptedData, err := EncryptBackup(zipData, publicKey)
		if err != nil {
			return fmt.Errorf("加密备份失败: %v", err)
		}

		// Create encrypted orbit file
		if err := CreateEncryptedOrbitFile(encryptedSymmetricKey, encryptedData); err != nil {
			return fmt.Errorf("创建加密orbit文件失败: %v", err)
		}

		logger.Info("备份已成功加密并保存为 backup.orbit")
	} else {
		// Save unencrypted backup
		if err := os.WriteFile("backup.orbit", zipData, 0644); err != nil {
			return err
		}
		logger.Info("备份已成功保存为 backup.orbit")
	}

	// 更新系统配置中的备份计数
	if configManager != nil && configManager.IsConfigLoaded() {
		err := configManager.UpdateSystemConfig(func(systemConfig *SystemConfig) {
			systemConfig.BackupCount++
			systemConfig.LastBackupTime = time.Now().Format("2006-01-02 15:04:05")
		})
		if err != nil {
			logger.Warnf("更新系统配置失败: %v", err)
		}
	}

	return nil
}

var save = &cobra.Command{
	Use:   "save",
	Short: "Create a backup of software configurations and installed software list",
	Long: `Create a compressed backup file (.orbit) containing:
	- manifest.json with timestamp and system information
	- software-list.json with installed software
	- configs/ folder with configuration files

Encryption is supported using a user-defined public key.`,
	Args: cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if err := createBackup(); err != nil {
			logger.Errorf("---  %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	save.Flags().StringVarP(&publicKeyPath, "public-key", "k", "", "Path to public key file for encryption (PEM format)")
	rootCmd.AddCommand(save)
}
