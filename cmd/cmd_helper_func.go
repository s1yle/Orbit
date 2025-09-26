package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
)

func findFileWithWalk(rootDir, targetFileName string) (*os.File, error) {
	var foundFile *os.File

	fmt.Printf("在当前路径(%v)查找文件：%v \n", rootDir, targetFileName)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil //跳过目录
		}

		// fmt.Println("rootDir, ", rootDir, " --- ", d.Name())

		fileName := filepath.Base(path)
		if fileName == targetFileName { //检查是否是目标文件
			foundFile_stat, err := os.Stat(path)
			if err != nil {
				fmt.Printf("无法获取文件(%v)信息: %v \n", foundFile_stat.Name(), err)
				return err
			}

			f, err := os.OpenFile(path, 0644, os.ModePerm)
			if err != nil {
				fmt.Printf("无法打开文件(%v): %v \n", foundFile_stat.Name(), err)
				return err
			}
			defer f.Close()

			fmt.Println("找到文件: ",
				foundFile_stat.Name(), " 大小: ",
				foundFile_stat.Size(), " 字节",
				foundFile_stat.Mode().Type(), " 类型")

			foundFile = f
		}

		return err
	})

	if err != nil {
		return foundFile, err
	}

	return foundFile, err
}

func copyFile(dst, src string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Println("出现错误，", err)
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		fmt.Println("出现错误，", err)
		return err
	}
	defer dstFile.Close()

	io.Copy(dstFile, srcFile) // 复制文件内容

	return nil
}

func findFileByPath(path string) (*os.File, error) {

	srcFile, err := os.Open(path)
	if err != nil {
		return srcFile, err
	}

	return srcFile, err
}

func getWinUserName() (string, error) {

	user, err := user.Current()
	if err != nil {
		return "unknown", err
	}

	return user.Name, nil
}

func getWinUserName2() (string, error) {
	var username string

	_, err := os.UserHomeDir()
	if err != nil {
		username = "unknown" // 如果获取失败，使用默认值
	} else {
		// 这是一个简单的方法，实际上可能需要更复杂的逻辑来提取用户名
		// 例如在Unix系统上，可以从环境变量 $USER 获取
		username = os.Getenv("USER")
		if username == "" {
			username = "unknown"
		}
	}

	return username, err
}
