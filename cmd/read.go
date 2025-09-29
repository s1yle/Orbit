package cmd

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
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

func readFromOrbitFile(filePath string) error {
	logger.Infof("正在读取 .orbit 文件: %s ", filePath)

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		logger.Infof("错误: 无法获取文件信息: %v ", err)
		return err
	}

	// 显示文件基本信息
	logger.Infof("文件大小: %d 字节 (%.2f KB) ", fileInfo.Size(), float64(fileInfo.Size())/1024)
	logger.Infof("修改时间: %s ", fileInfo.ModTime().Format("2006-01-02 15:04:05"))
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
	configCount := 0

	logger.Infof("文件列表:")
	for _, file := range r.File {
		// logger.Infof("  %d. %s", i+1, file.Name)

		if file.FileInfo().IsDir() {
			// logger.Infof(" (目录)")
		} else {
			fileSize := int64(file.UncompressedSize64)
			totalSize += fileSize
			// logger.Infof(" (%d 字节)", fileSize)
		}
		// fmt.Println()

		// 检查特殊文件
		if file.Name == "manifest.json" {
			hasManifest = true
		} else if file.Name == "software-list.json" {
			hasSoftwareList = true
		} else if strings.HasPrefix(file.Name, "configs/") && !file.FileInfo().IsDir() {
			configCount++
		}
	}

	logger.Infof("----------------------------------------")
	logger.Infof("总未压缩大小: %d 字节 (%.2f KB)", totalSize, float64(totalSize)/1024)
	logger.Infof("包含 manifest.json: %v", hasManifest)
	logger.Infof("包含 software-list.json: %v", hasSoftwareList)
	logger.Infof("配置文件数量: %d", configCount)

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

				// content, err := io.ReadAll(rc)
				// if err != nil {
				// 	logger.Infof("  错误: 无法读取 manifest.json 内容: %v ", err)
				// 	break
				// }

				readManifest(file)
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
