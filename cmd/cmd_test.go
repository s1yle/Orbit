package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestLoadFunc(t *testing.T) {
	// 创建根命令
	rootCmd := &cobra.Command{
		Use: "orbit",
	}

	// 创建 save 子命令
	saveCmd := &cobra.Command{
		Use:   "save",
		Short: "Save data",
		RunE: func(cmd *cobra.Command, args []string) error {
			// log.Println("\n[TEST] --------- 正在测试 'orbit save' 指令")
			return createBackup()
		},
	}

	// 创建 read 子命令
	readCmd := &cobra.Command{
		Use:   "read [file]",
		Short: "Read data from file",
		Args:  cobra.ExactArgs(1), // 确保有一个参数
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			// log.Println("\n[TEST] --------- 正在测试 'orbit read backup.orbit' 指令")
			return readFromOrbitFile(filePath)
		},
	}

	// 创建 load 子命令
	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "Load data",
		RunE: func(cmd *cobra.Command, args []string) error {
			// log.Println("\n[TEST] --------- 正在测试 'orbit load' 指令")
			return loadFunc("./backup.orbit")
		},
	}

	// 添加子命令到根命令
	rootCmd.AddCommand(saveCmd, readCmd, loadCmd)

	// 定义测试用例
	tests := []struct {
		name    string
		args    []string // 命令行参数
		wantErr bool     // 是否期望错误
	}{
		{
			name:    "orbit save command",
			args:    []string{"save"},
			wantErr: false,
		},
		{
			name:    "orbit read command with valid file",
			args:    []string{"read", "backup.orbit"},
			wantErr: false,
		},
		{
			name:    "orbit load command",
			args:    []string{"load"},
			wantErr: false,
		},
		// {
		// 	name:    "orbit read command without file (should fail)",
		// 	args:    []string{"read"},
		// 	wantErr: true, // 缺少参数，应该报错
		// },
	}

	// 执行测试
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置命令行参数
			rootCmd.SetArgs(tt.args)

			cmmd := append([]string{"orbit "}, tt.args...)
			command := strings.Join(cmmd, " ")
			PrintBoxedText("正在测试指令: "+command, SimpleStyle)

			// 执行命令
			err := rootCmd.Execute()

			// 检查错误是否符合预期
			if (err != nil) != tt.wantErr {
				t.Errorf("Command execution error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
