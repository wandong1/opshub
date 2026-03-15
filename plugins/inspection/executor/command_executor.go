package executor

import (
	"context"
	"fmt"
	"time"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/internal/server/agent"
	"github.com/ydcloud-dy/opshub/pkg/agentproto"

	"github.com/google/uuid"
)

// CommandExecutor 命令执行器
type CommandExecutor struct {
	agentHub *agent.AgentHub
}

// NewCommandExecutor 创建命令执行器
func NewCommandExecutor(agentHub *agent.AgentHub) *CommandExecutor {
	return &CommandExecutor{
		agentHub: agentHub,
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
	// 需要通过 CredentialRepo 获取凭证信息
	// 由于这里没有直接访问凭证的方式，我们需要使用 SSHUser 和 CredentialID
	// 实际使用时需要注入 CredentialRepo 来获取解密后的凭证

	// 临时方案：返回错误提示需要通过其他方式获取凭证
	return "", fmt.Errorf("SSH 执行需要凭证信息，请使用 Agent 方式或通过 HostUseCase 执行")
}

