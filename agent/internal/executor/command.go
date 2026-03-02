package executor

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// CommandExecutor 命令执行器
type CommandExecutor struct{}

// NewCommandExecutor 创建命令执行器
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{}
}

// Execute 执行命令
func (e *CommandExecutor) Execute(requestID, command string, timeout int32) *pb.AgentMessage {
	if timeout <= 0 {
		timeout = 60
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	errMsg := ""

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			errMsg = err.Error()
			exitCode = -1
		}
	}

	return &pb.AgentMessage{
		Payload: &pb.AgentMessage_CmdResult{
			CmdResult: &pb.CommandResult{
				RequestId: requestID,
				ExitCode:  int32(exitCode),
				Stdout:    stdout.String(),
				Stderr:    stderr.String(),
				Error:     errMsg,
			},
		},
	}
}
