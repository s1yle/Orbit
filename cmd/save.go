package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type Manifest struct {
	Timestamp string `json:"timestamp"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
}

func findFileByPath(path string) (*os.File, error) {

	srcFile, err := os.Open(path)
	if err != nil {
		return srcFile, err
	}

	return srcFile, err
}

func saveVscode(tempDir string) error {
	fmt.Println("[info] --- 正在保存Vscode配置文件...")

	codeConfigs := filepath.Join(os.Getenv("APPDATA"), "Code", "User", "settings.json")

	srcFile, err := findFileByPath(codeConfigs)
	if err != nil {
		return err
	}
	fmt.Println("[info] --- 找到配置文件：%v", srcFile.Name())

	// tempDir/configs/
	configsDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		return err
	}

	// tempDir/configs/vscode-settings.json
	destFile, err := os.Create(filepath.Join(configsDir, "vscode-settings.json"))
	if err != nil {
		return err
	}
	defer destFile.Close()
	fmt.Println("[info] --- 成功创建配置文件：%v", destFile.Name())

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}

	// fmt.Println("[info] --- configs.json的路径为：", codeConfigs, destFile)
	return err
}

func createBackup() error {

	tempDir, err := os.MkdirTemp("", "test_orbit-backup")
	fmt.Println("[info] --- 成功创建临时目录：%v", tempDir)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := saveVscode(tempDir); err != nil {
		return err
	}

	backupFile, err := os.Create("backup.orbit")
	if err != nil {
		return err
	}
	defer backupFile.Close()

	zipWriter := zip.NewWriter(backupFile)
	defer zipWriter.Close()

	err = filepath.Walk(tempDir, func(path string, info fs.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relpath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

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

	return err
}

var save = &cobra.Command{
	Use:   "save",
	Short: "Create a backup of software configurations and installed software list",
	Long: `Create a compressed backup file (.orbit) containing:
	- manifest.json with timestamp and system information
	- software-list.json with installed software
	- configs/ folder with configuration files`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := createBackup(); err != nil {
			fmt.Printf("[error] --- 发生错误，保存vscode配置文件失败, %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(save)
}
