package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

func loadFunc() error {
	fmt.Println("[info] --- 正在加载 .orbit 文件...")

	//检查当前目录是否有.orbit文件，如果有 就从当前目录加载
	// var orbitFile string

	_, curDir, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("无法获取当前目录, %v", curDir)
	}
	fmt.Println("当前目录: ", CurrentDir)

	return nil
}

var load = &cobra.Command{
	Use:   "load [name.orbit]",
	Short: "Load configuration from an .orbit file",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[info] --- 开始启动 load 程序...")

		if err := loadFunc(); err != nil {
			fmt.Printf("[error] ---  %v", err)
			os.Exit(1)
		}

		fmt.Println("[info] --- load 程序执行完毕.")
	},
}

func init() {
	rootCmd.AddCommand(load)
}
