package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/ydcloud-dy/opshub/agent/internal/client"
	"github.com/ydcloud-dy/opshub/agent/internal/config"
	"github.com/ydcloud-dy/opshub/agent/internal/executor"
	"github.com/ydcloud-dy/opshub/agent/internal/filemanager"
	"github.com/ydcloud-dy/opshub/agent/internal/terminal"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "srehub-agent",
	Short: "SREHub Agent - 运维管理Agent",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "启动Agent",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load(configFile)
		if err != nil {
			fmt.Printf("加载配置失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("SREHub Agent 启动中...\n")
		fmt.Printf("AgentID: %s\n", cfg.AgentID)
		fmt.Printf("Server:  %s\n", cfg.ServerAddr)

		grpcClient := client.NewGRPCClient(cfg)

		// 初始化处理器
		ptyMgr := terminal.NewPTYManager(grpcClient)
		fileMgr := filemanager.NewFileManager()
		cmdExec := executor.NewCommandExecutor()
		grpcClient.SetHandlers(ptyMgr, fileMgr, cmdExec)

		// 启动
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-quit
			fmt.Println("\n正在关闭Agent...")
			cancel()
		}()

		if err := grpcClient.Run(ctx); err != nil {
			if ctx.Err() == nil {
				fmt.Printf("Agent运行错误: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Println("Agent已关闭")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("srehub-agent v1.0.0")
	},
}

func init() {
	runCmd.Flags().StringVarP(&configFile, "config", "c", "agent.yaml", "配置文件路径")
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
