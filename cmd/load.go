package cmd

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
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
	configDir := filepath.Join(configPath, "configs")

	// 检查configs目录是否存在
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		logger.Errorf("configs目录不存在: %s", configDir)
		return nil, err
	}

	// 这里可以添加具体的配置文件读取逻辑
	// 例如读取JSON、YAML等格式的配置文件

	config := &Config{
		BackupConfigPath: configDir,
		TargetPath:       configPath,
	}

	logger.Infof("配置文件目录: %s", configDir)

	// 可以添加更多配置读取逻辑
	// 例如：读取具体的配置文件内容

	return config, nil
}

func loadFunc() error {
	targetExtractPath := "F:/_Default/GoLang/Proj/backup_orbit_test"

	logger.Infof("正在搜索 .orbit 文件...")

	// 搜索.orbit文件
	orbitFilePath, err := findOrbitFile(getCurrentDir())
	if err != nil {
		logger.Errorf("查找.orbit文件失败: %v", err)
		return err
	}

	logger.Infof("发现 .orbit 文件: %s", orbitFilePath)

	// 解压.orbit文件
	if err := extractOrbitFile(orbitFilePath, targetExtractPath); err != nil {
		logger.Errorf("解压.orbit文件失败: %v", err)
		return err
	}

	logger.Infof(".orbit 文件已成功解压到: %s", targetExtractPath)

	// 读取配置文件
	_, err = readConfigFile(targetExtractPath)
	if err != nil {
		logger.Errorf("读取配置文件失败: %v", err)
		return err
	}
	// logger.Infof("config: ", config)

	logger.Infof("配置加载完成")
	return nil
}

var load = &cobra.Command{
	Use:   "load [name.orbit]",
	Short: "Load configuration from an .orbit file",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger.Infof("开始启动 load 程序..., 参数为: ", args)

		if err := loadFunc(); err != nil {
			logger.Errorf("load程序执行失败, %v", err)
			return
		}

		logger.Infof("load 程序执行完毕.")
	},
}

func init() {
	rootCmd.AddCommand(load)
}
