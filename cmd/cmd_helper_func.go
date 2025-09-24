package cmd

import (
	"fmt"
	"io"
	"os"
	"os/user"
)

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
