package agent

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	"github.com/ydcloud-dy/opshub/pkg/collector"
)

// AgentExecutor 通过Agent执行命令，实现 collector.CommandExecutor 接口
type AgentExecutor struct {
	hub    *AgentHub
	stream *AgentStream
}

// Execute 通过Agent gRPC执行命令
func (e *AgentExecutor) Execute(cmd string) (string, error) {
	requestID := uuid.New().String()
	if err := e.stream.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_CmdRequest{
			CmdRequest: &pb.CommandRequest{
				RequestId: requestID,
				Command:   cmd,
				Timeout:   30,
			},
		},
	}); err != nil {
		return "", fmt.Errorf("发送命令失败: %w", err)
	}

	resp, err := e.hub.WaitResponse(e.stream, requestID, 35*time.Second)
	if err != nil {
		return "", fmt.Errorf("等待Agent响应失败: %w", err)
	}

	cmdResult, ok := resp.(*pb.CommandResult)
	if !ok {
		return "", fmt.Errorf("Agent返回了非预期的响应类型")
	}

	output := cmdResult.Stdout
	if cmdResult.Stderr != "" {
		if output != "" {
			output += "\n"
		}
		output += cmdResult.Stderr
	}

	if cmdResult.ExitCode != 0 {
		return output, fmt.Errorf("退出码: %d", cmdResult.ExitCode)
	}
	return output, nil
}

// AgentCommandFactory Agent命令执行工厂
type AgentCommandFactory struct {
	hub *AgentHub
}

// NewAgentCommandFactory 创建Agent命令执行工厂
func NewAgentCommandFactory(hub *AgentHub) *AgentCommandFactory {
	return &AgentCommandFactory{hub: hub}
}

// IsOnline 检查Agent是否在线
func (f *AgentCommandFactory) IsOnline(hostID uint) bool {
	return f.hub.IsOnline(hostID)
}

// NewExecutor 创建Agent命令执行器
func (f *AgentCommandFactory) NewExecutor(hostID uint) (collector.CommandExecutor, error) {
	as, ok := f.hub.GetByHostID(hostID)
	if !ok {
		return nil, fmt.Errorf("Agent不在线")
	}
	return &AgentExecutor{hub: f.hub, stream: as}, nil
}
