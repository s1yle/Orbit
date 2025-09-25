package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func readManifest(file *zip.File) error {
	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("无法打开 manifest.json: %v", err)
	}
	defer rc.Close()

	// 解析 JSON 内容
	content, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("无法读取 manifest.json 内容: %v", err)
	}

	var manifest Manifest
	err = json.Unmarshal(content, &manifest)
	if err != nil {
		return fmt.Errorf("无法解析 manifest.json 内容: %v", err)
	}

	fmt.Printf("Manifest 内容: \n")
	fmt.Printf("  保存的时间: %s\n", manifest.Timestamp)
	fmt.Printf("  系统:        %s\n", manifest.OS)
	fmt.Printf("  架构:      %s\n", manifest.Arch)
	fmt.Printf("  主机名:  %s\n", manifest.Hostname)
	fmt.Printf("  用户名:  %s\n", manifest.Username)

	return nil
}

func readFromOrbitFile(filePath string) error {
	fmt.Printf("正在读取 .orbit 文件: %s\n", filePath)

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("错误: 无法获取文件信息: %v\n", err)
		return err
	}

	// 显示文件基本信息
	fmt.Printf("文件大小: %d 字节 (%.2f KB)\n", fileInfo.Size(), float64(fileInfo.Size())/1024)
	fmt.Printf("修改时间: %s\n", fileInfo.ModTime().Format("2006-01-02 15:04:05"))
	fmt.Println("----------------------------------------")

	// 打开zip文件
	r, err := zip.OpenReader(filePath)
	if err != nil {
		fmt.Printf("错误: 无法打开 .orbit 文件: %v\n", err)
		return err
	}
	defer r.Close()

	// 显示zip文件基本信息
	fmt.Printf("包含文件数量: %d\n", len(r.File))
	fmt.Println("----------------------------------------")

	// 遍历所有文件并显示信息
	totalSize := int64(0)
	hasManifest := false
	hasSoftwareList := false
	configCount := 0

	fmt.Println("文件列表:")
	for _, file := range r.File {
		// fmt.Printf("  %d. %s", i+1, file.Name)

		if file.FileInfo().IsDir() {
			// fmt.Printf(" (目录)")
		} else {
			fileSize := int64(file.UncompressedSize64)
			totalSize += fileSize
			// fmt.Printf(" (%d 字节)", fileSize)
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

	fmt.Println("----------------------------------------")
	fmt.Printf("总未压缩大小: %d 字节 (%.2f KB)\n", totalSize, float64(totalSize)/1024)
	fmt.Printf("包含 manifest.json: %v\n", hasManifest)
	fmt.Printf("包含 software-list.json: %v\n", hasSoftwareList)
	fmt.Printf("配置文件数量: %d\n", configCount)

	// 如果存在manifest.json，读取并显示其内容
	if hasManifest {
		fmt.Println("----------------------------------------")

		for _, file := range r.File {
			if file.Name == "manifest.json" {
				rc, err := file.Open()
				if err != nil {
					fmt.Printf("  错误: 无法读取 manifest.json: %v\n", err)
					break
				}
				defer rc.Close()

				// content, err := io.ReadAll(rc)
				// if err != nil {
				// 	fmt.Printf("  错误: 无法读取 manifest.json 内容: %v\n", err)
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
			fmt.Println("Please provide the .orbit file name.")
			return
		}

		orbitFile := args[0]

		fmt.Printf("Reading .orbit file: %s\n", orbitFile)
		err := readFromOrbitFile(orbitFile)
		if err != nil {
			fmt.Errorf("failed to read .orbit file: %v", err)
		}

		fmt.Println("Read operation completed.")
	},
}

func init() {
	rootCmd.AddCommand(readOrbitFile)
}
