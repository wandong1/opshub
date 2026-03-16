package inspection_mgmt

import (
	"context"
	"fmt"
	"time"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/internal/server/agent"
	"github.com/ydcloud-dy/opshub/pkg/agentproto"
	sshclient "github.com/ydcloud-dy/opshub/pkg/ssh"

	"github.com/google/uuid"
)

// CommandExecutor 命令执行器
type CommandExecutor struct {
	agentHub       *agent.AgentHub
	credentialRepo assetbiz.CredentialRepo
}

// NewCommandExecutor 创建命令执行器
func NewCommandExecutor(agentHub *agent.AgentHub, credentialRepo assetbiz.CredentialRepo) *CommandExecutor {
	return &CommandExecutor{
		agentHub:       agentHub,
		credentialRepo: credentialRepo,
	}
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	Output   string
	Error    error
	Duration float64 // 秒
}

// Execute 执行命令
func (e *CommandExecutor) Execute(ctx context.Context, host *assetbiz.Host, command string, executionMode string, timeout int) *ExecuteResult {
	startTime := time.Now()

	// 根据执行模式选择执行方式
	var output string
	var err error

	switch executionMode {
	case "agent":
		// 仅使用 Agent
		if e.agentHub == nil || !e.agentHub.IsOnline(host.ID) {
			err = fmt.Errorf("Agent 未在线")
		} else {
			output, err = e.executeViaAgent(ctx, host.ID, command, timeout)
		}
	case "ssh":
		// 仅使用 SSH
		output, err = e.executeViaSSH(ctx, host, command, timeout)
	case "auto":
		fallthrough
	default:
		// 自动选择：优先 Agent，回退 SSH
		if e.agentHub != nil && e.agentHub.IsOnline(host.ID) {
			output, err = e.executeViaAgent(ctx, host.ID, command, timeout)
		} else {
			output, err = e.executeViaSSH(ctx, host, command, timeout)
		}
	}

	duration := time.Since(startTime).Seconds()

	return &ExecuteResult{
		Output:   output,
		Error:    err,
		Duration: duration,
	}
}

// executeViaAgent 通过 Agent 执行命令
func (e *CommandExecutor) executeViaAgent(ctx context.Context, hostID uint, command string, timeout int) (string, error) {
	as, ok := e.agentHub.GetByHostID(hostID)
	if !ok {
		return "", fmt.Errorf("Agent 连接不存在")
	}

	requestID := uuid.New().String()

	// 发送命令请求
	cmdReq := &agentproto.CommandRequest{
		RequestId: requestID,
		Command:   command,
	}

	msg := &agentproto.ServerMessage{
		Payload: &agentproto.ServerMessage_CmdRequest{
			CmdRequest: cmdReq,
		},
	}

	if err := as.Send(msg); err != nil {
		return "", fmt.Errorf("发送命令失败: %v", err)
	}

	// 等待响应
	timeoutDuration := time.Duration(timeout) * time.Second
	resp, err := e.agentHub.WaitResponse(as, requestID, timeoutDuration)
	if err != nil {
		return "", fmt.Errorf("等待响应超时: %v", err)
	}

	// 类型断言获取命令结果
	cmdResult, ok := resp.(*agentproto.CommandResult)
	if !ok {
		return "", fmt.Errorf("响应格式错误")
	}

	if cmdResult.ExitCode != 0 {
		return cmdResult.Stdout + cmdResult.Stderr, fmt.Errorf("命令执行失败，退出码: %d", cmdResult.ExitCode)
	}

	return cmdResult.Stdout, nil
}

// executeViaSSH 通过 SSH 执行命令
func (e *CommandExecutor) executeViaSSH(ctx context.Context, host *assetbiz.Host, command string, timeout int) (string, error) {
	// 检查主机是否配置了凭证
	if host.CredentialID == 0 {
		return "", fmt.Errorf("主机未配置 SSH 凭证")
	}

	// 获取解密后的凭证信息
	credential, err := e.credentialRepo.GetByIDDecrypted(ctx, host.CredentialID)
	if err != nil {
		return "", fmt.Errorf("获取凭证信息失败: %w", err)
	}

	// 创建 SSH 客户端
	client, err := e.createSSHClient(host, credential)
	if err != nil {
		return "", fmt.Errorf("创建 SSH 客户端失败: %w", err)
	}
	defer client.Close()

	// 执行命令
	output, err := client.Execute(command)
	if err != nil {
		return output, fmt.Errorf("SSH 命令执行失败: %w", err)
	}

	return output, nil
}

// createSSHClient 创建SSH客户端
func (e *CommandExecutor) createSSHClient(host *assetbiz.Host, credential *assetbiz.Credential) (*sshclient.Client, error) {
	var privateKey []byte

	// 检查凭证信息是否完整
	if credential.Type == "password" && credential.Password == "" {
		return nil, fmt.Errorf("凭证类型为密码认证，但未填写密码")
	}
	if credential.Type == "key" && credential.PrivateKey == "" {
		return nil, fmt.Errorf("凭证类型为密钥认证，但未填写私钥")
	}

	// 如果是密钥认证，需要解密私钥
	if credential.Type == "key" && credential.PrivateKey != "" {
		privateKey = []byte(credential.PrivateKey)
	}

	client, err := sshclient.NewClient(
		host.IP,
		host.Port,
		host.SSHUser,
		credential.Password,
		privateKey,
		credential.Passphrase,
	)
	if err != nil {
		return nil, fmt.Errorf("创建SSH客户端失败: %w", err)
	}

	return client, nil
}

