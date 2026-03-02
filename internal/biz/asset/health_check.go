package asset

import (
	"context"
	"fmt"
	"sync"
	"time"

	sshclient "github.com/ydcloud-dy/opshub/pkg/ssh"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/scheduler"
	"go.uber.org/zap"
)

// HostHealthExecutor 主机健康检查执行器
type HostHealthExecutor struct {
	hostRepo       HostRepo
	credentialRepo CredentialRepo
	agentFactory   AgentCommandFactory
}

// NewHostHealthExecutor 创建主机健康检查执行器
func NewHostHealthExecutor(hostRepo HostRepo, credentialRepo CredentialRepo) *HostHealthExecutor {
	return &HostHealthExecutor{
		hostRepo:       hostRepo,
		credentialRepo: credentialRepo,
	}
}

// SetAgentCommandFactory 设置Agent命令执行工厂
func (e *HostHealthExecutor) SetAgentCommandFactory(f AgentCommandFactory) {
	e.agentFactory = f
}

// Type 返回执行器类型
func (e *HostHealthExecutor) Type() string {
	return "host_health_check"
}

// Execute 执行健康检查
func (e *HostHealthExecutor) Execute(ctx context.Context, task scheduler.Task) error {
	hosts, err := e.hostRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("获取主机列表失败: %w", err)
	}
	if len(hosts) == 0 {
		return nil
	}

	appLogger.Info("开始主机健康检查", zap.Int("total", len(hosts)))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // 并发信号量

	for _, host := range hosts {
		wg.Add(1)
		sem <- struct{}{}
		go func(h *Host) {
			defer wg.Done()
			defer func() { <-sem }()
			e.checkHost(ctx, h)
		}(host)
	}

	wg.Wait()
	appLogger.Info("主机健康检查完成", zap.Int("total", len(hosts)))
	return nil
}

// checkHost 检测单台主机
func (e *HostHealthExecutor) checkHost(ctx context.Context, host *Host) {
	oldStatus := host.Status
	newStatus := 0 // 默认离线
	now := time.Now()

	// 1. Agent 在线 → 直接标记在线
	if e.agentFactory != nil && e.agentFactory.IsOnline(host.ID) {
		newStatus = 1
	} else if host.CredentialID > 0 {
		// 2. 有凭证 → SSH 连接测试
		if e.testSSH(ctx, host) {
			newStatus = 1
		}
	}
	// 3. 无凭证且无 Agent → status=0

	if newStatus == oldStatus {
		return
	}

	host.Status = newStatus
	if newStatus == 1 {
		host.LastSeen = &now
	}
	if err := e.hostRepo.Update(ctx, host); err != nil {
		appLogger.Error("更新主机状态失败",
			zap.Uint("hostID", host.ID),
			zap.String("ip", host.IP),
			zap.Error(err),
		)
	}
}

// testSSH 通过 SSH 测试主机连通性
func (e *HostHealthExecutor) testSSH(ctx context.Context, host *Host) bool {
	cred, err := e.credentialRepo.GetByIDDecrypted(ctx, host.CredentialID)
	if err != nil {
		return false
	}

	user := host.SSHUser
	if user == "" && cred.Username != "" {
		user = cred.Username
	}
	if user == "" {
		user = "root"
	}

	port := host.Port
	if port == 0 {
		port = 22
	}

	var privateKey []byte
	if cred.PrivateKey != "" {
		privateKey = []byte(cred.PrivateKey)
	}

	client, err := sshclient.NewClient(host.IP, port, user, cred.Password, privateKey, cred.Passphrase)
	if err != nil {
		return false
	}
	defer client.Close()

	return client.TestConnection() == nil
}

// Ensure interface compliance at compile time.
var _ scheduler.TaskExecutor = (*HostHealthExecutor)(nil)
