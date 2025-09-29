package cmd

import (
	"archive/zip"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

type DirName struct {
	Path string
	Type string
}

func saveVscode(tempDir string) error {
	logger.Infof("正在保存Vscode配置文件...")
	logger.Infof("CodeConfigDir: %v", CodeConfigDir)

	configs_base := filepath.Join(tempDir, "configs")
	configs_base_vscode_base := filepath.Join(tempDir, "configs", "vscode_config_dir")
	configs_base_vscode_base_APPDATA := filepath.Join(tempDir, "configs", "vscode_config_dir", "APPDATA")
	configs_base_vscode_base_USER := filepath.Join(tempDir, "configs", "vscode_config_dir", "USER")

	UsersFile := filepath.Join(CodeConfigDir, "User")
	WorkspacesFile := filepath.Join(CodeConfigDir, "Workspaces")
	ConfigsInUser := filepath.Join(CodeUserDir)
	dirs := []DirName{DirName{UsersFile, "APPDATA"}, DirName{WorkspacesFile, "APPDATA"}, DirName{ConfigsInUser, "USER"}}
	// fmt.Println("dirs: ", dirs)
	err := os.MkdirAll(filepath.Join(configs_base), 0644)
	if err != nil {
		return err
	}
	err = os.MkdirAll(configs_base_vscode_base, 0644)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		var workingDir string
		switch dir.Type {
		case "APPDATA":
			workingDir := filepath.Base(dir.Path)
			logger.Infof("正在保存 -> %v (%v)", workingDir, dir.Type)
			err = os.MkdirAll(filepath.Join(configs_base_vscode_base_APPDATA, workingDir), 0644)
		case "USER":
			workingDir := filepath.Base(dir.Path)
			err = os.MkdirAll(filepath.Join(configs_base_vscode_base_USER, workingDir), 0644)
		}

		if err != nil {
			return err
		}

		filepath.Walk(dir.Path, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// fmt.Printf("[info] --- 当前目录: %v \n", filepath.Base(dir))
			relPath, err := filepath.Rel(dir.Path, path) // 获取相对路径
			// logger.Infof("dir.Path: %v", dir.Path)       //path: C:\Users\mmili985\AppData\Roaming\Code\User\globalStorage\state.vscdb
			// logger.Infof("relPath: %v", relPath)         //relPath: extensions\vscodeshift.mui-snippets-1.0.1\package.json
			if err != nil {
				return err
			}
			dstPath := filepath.Join(filepath.Join(configs_base_vscode_base, dir.Type, workingDir), filepath.Base(dir.Path), relPath) // 目标路径
			// logger.Infof("dstPath: %v", dstPath)
			//C:\Users\mmili985\AppData\Local\Temp\test_orbit-backup3878169596\configs\vscode_config_dir\USER\extensions\heisthepirate.mui-snippets-updated-1.0.0\node_modules\atob

			if info.IsDir() {
				err = os.MkdirAll(dstPath, os.ModePerm) // 创建目录
				if err != nil {
					return err
				}
				// fmt.Println("dir: ", dstPath)
			} else {
				err = copyFile(dstPath, path) // 复制文件
				if err != nil {
					return err
				}
				// fmt.Println(path)
			}
			return nil
		})
	}

	return nil
}

// 获取系统信息到manifest 中并转换为 byte array
func convertMeniToJson() ([]byte, error) {
	var jsonData []byte // 获取系统信息

	hostname, err := os.Hostname()
	if err != nil {
		logger.Infof("获取主机名失败: %v", err)
		return jsonData, err
	}

	// 获取当前用户名
	// 注意：os.UserHomeDir() 不能直接获取用户名，这里使用另一种方式
	username, err := getWinUserName()
	if err != nil {
		return jsonData, err
	}

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

// @param tempDir C:\Users\mmili\AppData\Local\Temp\test_orbit-backup
func createOrbitZip(tempDir string) error {

	backupFile, err := os.Create("backup.orbit")
	if err != nil {
		return err
	}
	defer backupFile.Close()

	zipWriter := zip.NewWriter(backupFile)
	defer zipWriter.Close()

	logger.Info("---  正在将文件写入 orbit包")
	err = filepath.Walk(tempDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// fmt.Println(path)

		relpath, err := filepath.Rel(tempDir, path)

		if relpath == "." {
			return nil
		}

		if err != nil {
			return err
		}

		// 如果是目录，创建目录条目并返回
		if info.IsDir() {
			// fmt.Println(relpath)
			// 为目录创建条目（以斜杠结尾）
			_, err := zipWriter.Create(relpath + "/")
			if err != nil {
				return err
			}

			// dirname := info.Name()

			// switch dirname {
			// case "globalStorage":
			// 	logger.Infof("正在保存 -> 扩展的全局配置数据(%v)", dirname)
			// case "History":
			// 	logger.Infof("正在保存 -> 文件编辑历史(%v)", dirname)
			// case "snippets":
			// 	logger.Infof("正在保存 -> 用户自定义代码片段(%v)", dirname)
			// case "workspaceStorage":
			// 	logger.Infof("正在保存 -> 工作区特定配置(%v)", dirname)
			// }
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

	return err
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
	jsonData, err := convertMeniToJson()
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(tempDir, "manifest.json")
	if err := os.WriteFile(manifestPath, jsonData, 0644); err != nil {
		return err
	}
	logger.Info("创建manifest.json文件")

	//压缩成.orbit格式
	err = createOrbitZip(tempDir)
	if err != nil {
		return err
	}

	return err
}

var save = &cobra.Command{
	Use:   "save",
	Short: "Create a backup of software configurations and installed software list",
	Long: `Create a compressed backup file (.orbit) containing:
	- manifest.json with timestamp and system information
	- software-list.json with installed software
	- configs/ folder with configuration files`,
	Args: cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if err := createBackup(); err != nil {
			logger.Errorf("---  %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(save)
}
