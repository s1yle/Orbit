package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	privateKeyPath string
)

type Config struct {
	BackupConfigPath string `json:"backup_config_path"`
	TargetPath       string `json:"target_path"`
}

// findOrbitFile 在当前目录及父目录中搜索.orbit文件
func findOrbitFile(startDir string) (string, error) {
	// 首先在当前目录搜索
	orbitFiles, err := filepath.Glob(filepath.Join(startDir, "*.orbit"))
	if err != nil {
		return "", err
	}

	if len(orbitFiles) > 0 {
		// 返回找到的第一个.orbit文件
		return orbitFiles[0], nil
	}

	// 如果没有找到，向父目录搜索
	parentDir := filepath.Dir(startDir)
	if parentDir == startDir {
		// 已经到达根目录
		logger.Errorf("未找到.orbit文件")
		return "", err
	}

	return findOrbitFile(parentDir)
}

// extractFile 解压单个文件
func extractFile(zipFile *zip.File, targetPath string) error {
	// logger.Infof("解压单个文件中 -- 文件名: %v, 解压目录: %v ", zipFile.Name, targetPath)
	// 如果是目录，创建目录
	if zipFile.FileInfo().IsDir() {
		return os.MkdirAll(targetPath, zipFile.Mode())
	}

	// 创建文件所在目录
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// 创建目标文件
	targetFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipFile.Mode())
	if err != nil {
		return err
	}
	defer targetFile.Close()

	// 打开zip中的文件
	zipFileReader, err := zipFile.Open()
	if err != nil {
		return err
	}
	defer zipFileReader.Close()

	// 复制文件内容
	_, err = io.Copy(targetFile, zipFileReader)
	return err
}

// extractOrbitFile 解压.orbit压缩文件
func extractOrbitFile(orbitFilePath, targetPath string) error {
	logger.Infoln(targetPath)
	// 打开zip文件
	reader, err := zip.OpenReader(orbitFilePath)
	if err != nil {
		logger.Errorf("无法打开.orbit文件: %v", err)
		return err
	}
	defer reader.Close()

	// 创建目标目录
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		logger.Errorf("创建目标目录失败: %v", err)
		return err
	}

	// 遍历zip文件中的每个文件/目录
	for _, file := range reader.File {
		// logger.Infof("文件: %v ", file.Name)
		filePath := filepath.Join(targetPath, file.Name)

		if err := extractFile(file, filePath); err != nil {
			logger.Errorf("解压文件 %s 失败: %v", file.Name, err)
			return err
		}
	}

	return nil
}

// readConfigFile 读取配置文件
func readConfigFile(configPath string) (*Config, error) {

	// 检查configs目录是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Errorf("configs目录不存在: %s", configPath)
		return nil, err
	}

	// 这里可以添加具体的配置文件读取逻辑
	// 例如读取JSON、YAML等格式的配置文件

	config := &Config{
		BackupConfigPath: configPath,
		TargetPath:       configPath,
	}

	logger.Infof("配置文件目录: %s", configPath)

	// 可以添加更多配置读取逻辑
	// 例如：读取具体的配置文件内容

	return config, nil
}

// 读取.orbit 文件的结构
func readStruct(targetZipFile string, targetDirName string) (*ConfigDirType, error) {
	if targetDirName == "" {
		return nil, fmt.Errorf("targetDirName 不能为空")
	}

	logger.Infof("func<readStruct> -> 正在读取结构信息(%v)...", targetDirName)
	logger.Infoln("func<readStruct> -> 目标文件: ", targetZipFile)

	// 打开zip文件
	r, err := zip.OpenReader(targetZipFile)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var configDir ConfigDirType
	for _, f := range r.File {
		// logger.Infof("func<readStruct> 发现文件: %s", f.Name)
		if strings.HasPrefix(f.Name, targetDirName) || f.Name == targetDirName || filepath.Base(f.Name) == targetDirName {

			logger.Infof("func<readStruct> 找到目标目录: %s", f.Name)
			logger.Infof("文件信息: %+v", f.FileInfo())

			// 找到目标目录，读取信息
			configDir.Name = filepath.Base(targetDirName)
			configDir.Path = f.Name
			switch filepath.Base(f.Name) {
			case "APPDATA":
				configDir.OriginalPath = CodeConfigDir
			case "USER":
				configDir.OriginalPath = CodeUserDir
				logger.Infof("cdt: %v", configDir)
			default:
				configDir.OriginalPath = "Unknown"
			}

			return &configDir, err
		}
	}

	return &configDir, err
}

func extractSpecificDir(zipFile, targetDirInZip, destDir string) error {
	// 打开ZIP文件
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	// 遍历ZIP中的文件/目录
	for _, f := range r.File {
		// 检查文件路径是否位于目标目录内
		if strings.HasPrefix(f.Name, targetDirInZip) || f.Name == targetDirInZip {
			// 构建目标路径
			fpath := filepath.Join(destDir, strings.TrimPrefix(f.Name, targetDirInZip))
			logger.Info("正在解压的文件: ", fpath)
			logger.Info("destDir: ", destDir)
			logger.Info("strings.TrimPrefix(f.Name, targetDirInZip): ", strings.TrimPrefix(f.Name, targetDirInZip))

			// 如果是目录，则创建目录
			if f.FileInfo().IsDir() {
				if err := os.MkdirAll(fpath, f.Mode()); err != nil {
					return err
				}
				continue
			}

			// 确保文件所在目录存在
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return err
			}

			// 创建目标文件
			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer outFile.Close()

			// 打开ZIP中的源文件
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			// 复制文件内容
			_, err = io.Copy(outFile, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func handleByConfigDirType(cdt *ConfigDirType, orbitFilePath string) error {
	targetExtractPath := "./bacup_orbit/extractPath"

	// 处理不同的配置目录
	switch cdt.Name {
	case "APPDATA":
		// 处理 APPDATA 目录
		logger.Infof("配置目录 %s 对应系统目录: %s", cdt.Name, cdt.OriginalPath)
	case "USER":
		// 处理 USER 目录
		logger.Infof("配置目录 %s 对应系统目录: %s", cdt.Name, cdt.OriginalPath)

	default:
		logger.Infof("未知配置目录: %s", cdt.Name)
	}

	// 解压.orbit文件
	targetExtractPath = filepath.Dir(cdt.OriginalPath)
	logger.Infof("正在将 .orbit 文件解压到: %s, 指定zip路径：%s", targetExtractPath, filepath.Join(cdt.Path))
	if err := extractSpecificDir(orbitFilePath, filepath.Join(cdt.Path), targetExtractPath); err != nil {
		logger.Errorf("解压.orbit文件失败: %v", err)
		return err
	}
	logger.Infof(".orbit 文件已成功解压到: %s", targetExtractPath)

	// 读取配置文件
	_, err := readConfigFile(targetExtractPath)
	if err != nil {
		logger.Errorf("读取配置文件失败: %v", err)
		return err
	}
	// logger.Infof("config: ", config)

	return err

}

// loadDecryptedOrbitFile handles loading and decrypting an encrypted orbit file
func loadDecryptedOrbitFile(orbitFilePath, privateKeyPath string) error {
	logger.Infof("正在加载加密的 .orbit 文件: %s", orbitFilePath)

	// Load private key
	privateKey, err := LoadPrivateKey(privateKeyPath)
	if err != nil {
		return fmt.Errorf("加载私钥失败: %v", err)
	}

	// Read encrypted orbit file
	encryptedSymmetricKey, encryptedData, err := ReadEncryptedOrbitFile(orbitFilePath)
	if err != nil {
		return fmt.Errorf("读取加密orbit文件失败: %v", err)
	}

	// Decrypt backup data
	decryptedData, err := DecryptBackup(encryptedSymmetricKey, encryptedData, privateKey)
	if err != nil {
		return fmt.Errorf("解密备份数据失败: %v", err)
	}

	// Create temporary file for decrypted data
	tempFile, err := os.CreateTemp("", "orbit_decrypted_*.orbit")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write decrypted data to temporary file
	if _, err := tempFile.Write(decryptedData); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}

	// Use the temporary file for loading
	orbitFilePath = tempFile.Name()

	logger.Info("备份数据解密成功")

	DirPaths := []string{"APPDATA", "USER"}
	for _, dir := range DirPaths {

		// 读取结构信息
		cdt, err := readStruct(orbitFilePath, dir)
		if err != nil {
			logger.Errorf("读取结构信息失败: %v", err)
			return err
		}

		// 处理配置目录
		if err := handleByConfigDirType(cdt, orbitFilePath); err != nil {
			logger.Errorf("处理配置目录失败: %v", err)
			return err
		}
	}

	logger.Infof("配置加载完成")
	return nil
}

func loadFunc(orbitFilePath string) error {
	// 如果没有提供文件路径，搜索当前目录及父目录
	if orbitFilePath == "" {
		logger.Infof("正在搜索 .orbit 文件...")

		// 搜索.orbit文件
		orbitFilePath, err := findOrbitFile(getCurrentDir())
		if err != nil {
			logger.Errorf("查找.orbit文件失败: %v", err)
			return err
		}
		logger.Infof("发现 .orbit 文件: %s", orbitFilePath)
	}

	// Check if file is encrypted
	fileData, err := os.ReadFile(orbitFilePath)
	if err != nil {
		return fmt.Errorf("读取orbit文件失败: %v", err)
	}

	// Check if it's an encrypted file
	isEncrypted := len(fileData) >= len(EncryptedVerStr) &&
		string(fileData[:len(EncryptedVerStr)]) == EncryptedVerStr

	if isEncrypted {
		if privateKeyPath == "" {
			return fmt.Errorf("检测到加密的orbit文件，但未提供私钥。请使用 --private-key 参数指定私钥文件")
		}
		return loadDecryptedOrbitFile(orbitFilePath, privateKeyPath)
	}

	// Original unencrypted loading logic
	DirPaths := []string{"APPDATA", "USER"}
	for _, dir := range DirPaths {

		// 读取结构信息
		cdt, err := readStruct(orbitFilePath, dir)
		if err != nil {
			logger.Errorf("读取结构信息失败: %v", err)
			return err
		}

		// 处理配置目录
		if err := handleByConfigDirType(cdt, orbitFilePath); err != nil {
			logger.Errorf("处理配置目录失败: %v", err)
			return err
		}
	}

	logger.Infof("配置加载完成")
	return nil
}

var load = &cobra.Command{
	Use:   "load [name.orbit]",
	Short: "Load configuration from an .orbit file",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger.Infof("开始启动 load 程序..., 参数为: ", args)

		if err := loadFunc(args[0]); err != nil {
			logger.Errorf("load程序执行失败, %v", err)
			return
		}

		logger.Infof("load 程序执行完毕.")
	},
}

func init() {
	load.Flags().StringVarP(&privateKeyPath, "private-key", "k", "", "Path to private key file for decryption (PEM format)")
	rootCmd.AddCommand(load)
}
