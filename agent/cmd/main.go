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
	"github.com/ydcloud-dy/opshub/agent/internal/logger"
	"github.com/ydcloud-dy/opshub/agent/internal/prober"
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

		// 初始化日志
		logFile := cfg.LogFile
		if logFile == "" {
			logFile = "/var/log/srehub-agent/agent.log"
		}
		if err := logger.Init(logFile, cfg.LogMaxSize, cfg.LogMaxBackups, cfg.LogLevel); err != nil {
			fmt.Printf("初始化日志失败: %v\n", err)
			os.Exit(1)
		}

		logger.Info("SREHub Agent 启动中...")
		logger.Info("AgentID: %s", cfg.AgentID)
		logger.Info("Server: %s", cfg.ServerAddr)
		logger.Info("日志文件: %s (最大: %dMB, 保留: %d个备份)", logFile, cfg.LogMaxSize, cfg.LogMaxBackups)

		grpcClient := client.NewGRPCClient(cfg)

		// 初始化处理器
		ptyMgr := terminal.NewPTYManager(grpcClient)
		fileMgr := filemanager.NewFileManager()
		cmdExec := executor.NewCommandExecutor()
		probeMgr := prober.NewManager()

		grpcClient.SetHandlers(ptyMgr, fileMgr, cmdExec)
		grpcClient.SetProbeHandler(probeMgr)

		logger.Info("拨测功能已启用")

		// 启动
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-quit
			logger.Info("收到退出信号，正在关闭Agent...")
			cancel()
		}()

		if err := grpcClient.Run(ctx); err != nil {
			if ctx.Err() == nil {
				logger.Error("Agent运行错误: %v", err)
				os.Exit(1)
			}
		}
		logger.Info("Agent已关闭")
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
