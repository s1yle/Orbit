package cmd

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func readManifest(file *zip.File) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// 解析 JSON 内容
	content, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	var manifest Manifest
	err = json.Unmarshal(content, &manifest)
	if err != nil {
		return err
	}

	logger.Infof("Manifest 内容:  ")
	logger.Infof("  保存的时间: %s ", manifest.Timestamp)
	logger.Infof("  系统:        %s ", manifest.OS)
	logger.Infof("  架构:      %s ", manifest.Arch)
	logger.Infof("  主机名:  %s ", manifest.Hostname)
	logger.Infof("  用户名:  %s ", manifest.Username)

	return nil
}

func readVscodeConfigDir(dirPath string) error {
	var err error
	logger.Infof("正在读取 vscode 配置目录, 路径: %v", dirPath)

	logger.Infof("成功读取 vscode配置目录...")
	return err
}

func readSoftwareList(file *zip.File) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// 解析 JSON 内容
	content, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	var softwareList SoftwareList
	err = json.Unmarshal(content, &softwareList)
	if err != nil {
		return err
	}

	logger.Infof("Software-list 内容:  ")
	logger.Infof("  保存的时间: %s ", softwareList.Timestamp)
	logger.Infof("  app 数量:  %v 个", softwareList.TotalCount)

	return err
}

func readFromOrbitFile(filePath string) error {
	logger.Infof("正在读取 .orbit 文件: %s ", filePath)

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		logger.Infof("错误: 无法获取文件信息: %v ", err)
		return err
	}

	// 显示文件基本信息
	logger.Infof("文件大小: %v MB (%.2f KB) ", fileInfo.Size()/1024/1024, float64(fileInfo.Size())/1024)
	logger.Infof("最后修改时间: %s ", fileInfo.ModTime().Format("2006-01-02 15:04:05"))
	logger.Infof("----------------------------------------")

	// 打开zip文件
	r, err := zip.OpenReader(filePath)
	if err != nil {
		logger.Infof("错误: 无法打开 .orbit 文件: %v ", err)
		return err
	}
	defer r.Close()

	// 显示zip文件基本信息
	logger.Infof("包含文件数量: %d ", len(r.File))
	logger.Infof("----------------------------------------")

	// 遍历所有文件并显示信息
	totalSize := int64(0)
	hasManifest := false
	hasSoftwareList := false
	hasVscodeConfig := false
	hasVscodeAPPDATAConfig := false
	hasVscodeUSERConfig := false

	logger.Infof("目录列表:")
	for _, file := range r.File {
		if file.FileInfo().IsDir() {
			// logger.Infof("  [目录] %s", file.FileInfo().Name())
			if strings.HasPrefix(file.FileInfo().Name(), filepath.Join("configs", "vscode_config_dir")) && !hasVscodeConfig {
				logger.Infof("读取到 vscode 配置目录, %v", file.FileInfo().Name())
				hasVscodeConfig = true
				continue
			}

			if strings.HasPrefix(file.FileInfo().Name(),
				filepath.Join("configs", "vscode_config_dir", "APPDATA")) && !hasVscodeAPPDATAConfig {
				logger.Infof("读取到 vscode APPDATA 配置目录, %v", file.FileInfo().Name())
				hasVscodeAPPDATAConfig = true
				continue
			}

			if strings.HasPrefix(file.FileInfo().Name(),
				filepath.Join("configs", "vscode_config_dir", "USER")) && !hasVscodeUSERConfig {
				logger.Infof("读取到 vscode USER 配置目录, %v", file.FileInfo().Name())
				hasVscodeUSERConfig = true
				continue
			}

		} else {
			fileSize := int64(file.UncompressedSize64)
			totalSize += fileSize
		}

		// 检查特殊文件
		if file.Name == "manifest.json" {
			hasManifest = true
		} else if file.Name == "software-list.json" {
			hasSoftwareList = true
		}
	}

	logger.Infof("----------------------------------------")
	logger.Infof("总未压缩大小:  %v MB (%.2f KB)", totalSize/1024/1024, float64(totalSize)/1024)
	logger.Infof("包含 manifest.json: %v", hasManifest)
	logger.Infof("包含 software-list.json: %v", hasSoftwareList)

	// 如果存在manifest.json，读取并显示其内容
	if hasManifest {
		logger.Infof("----------------------------------------")

		for _, file := range r.File {
			if file.Name == "manifest.json" {
				rc, err := file.Open()
				if err != nil {
					logger.Errorf("  错误: 无法读取 manifest.json: %v ", err)
					break
				}
				defer rc.Close()

				readManifest(file)
				break
			}
		}
	}

	if hasSoftwareList {
		logger.Infof("----------------------------------------")
		logger.Infof("读取 software-list.json 内容...")

		for _, file := range r.File {
			if file.Name == "software-list.json" {
				rc, err := file.Open()
				if err != nil {
					logger.Errorf("  错误: 无法读取 software-list.json: %v ", err)
					break
				}
				defer rc.Close()

				readSoftwareList(file)
				break
			}
		}
	}

	return nil
}

var readOrbitFile = &cobra.Command{
	Use:   "read [name.orbit]",
	Short: "Read configuration from an .orbit file",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 参数检查
		if len(args) < 1 {
			logger.Errorf("Please provide the .orbit file name.")
			return
		}

		orbitFile := args[0]

		logger.Infof("Reading .orbit file: %s ", orbitFile)
		err := readFromOrbitFile(orbitFile)
		if err != nil {
			logger.Errorf("failed to read .orbit file: %v", err)
			return
		}

		logger.Infof("Read operation completed.")
	},
}

func init() {
	rootCmd.AddCommand(readOrbitFile)
}
