package inspection_mgmt

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
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

// ScriptExecuteRequest 脚本执行请求
type ScriptExecuteRequest struct {
	ScriptType    string // shell/python/binary
	ScriptContent string // 脚本内容
	ScriptFile    string // 脚本文件路径（如果是上传的文件）
	ScriptArgs    string // 脚本参数
}

// Execute 执行命令
func (e *CommandExecutor) Execute(ctx context.Context, host *assetbiz.Host, command string, executionMode string, timeout int) *ExecuteResult {
	fmt.Printf("[CommandExecutor] Execute command - hostID: %d, hostName: %s, command: %s, executionMode: %s\n",
		host.ID, host.Name, command, executionMode)
	startTime := time.Now()

	// 根据执行模式选择执行方式
	var output string
	var err error

	switch executionMode {
	case "agent":
		// 仅使用 Agent
		if e.agentHub == nil || !e.agentHub.IsOnline(host.ID) {
			err = fmt.Errorf("Agent 未在线")
			fmt.Printf("[CommandExecutor] Agent not online - hostID: %d\n", host.ID)
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
			fmt.Printf("[CommandExecutor] Using Agent - hostID: %d\n", host.ID)
			output, err = e.executeViaAgent(ctx, host.ID, command, timeout)
		} else {
			fmt.Printf("[CommandExecutor] Using SSH - hostID: %d\n", host.ID)
			output, err = e.executeViaSSH(ctx, host, command, timeout)
		}
	}

	duration := time.Since(startTime).Seconds()
	fmt.Printf("[CommandExecutor] Execute completed - hostID: %d, duration: %.2fs, hasError: %v\n",
		host.ID, duration, err != nil)

	return &ExecuteResult{
		Output:   output,
		Error:    err,
		Duration: duration,
	}
}

// ExecuteScript 执行脚本
func (e *CommandExecutor) ExecuteScript(ctx context.Context, host *assetbiz.Host, req *ScriptExecuteRequest, executionMode string, timeout int) *ExecuteResult {
	fmt.Printf("[CommandExecutor] ExecuteScript - hostID: %d, hostName: %s, scriptType: %s, executionMode: %s\n",
		host.ID, host.Name, req.ScriptType, executionMode)
	startTime := time.Now()

	var output string
	var err error

	// 根据执行模式选择执行方式
	switch executionMode {
	case "agent":
		if e.agentHub == nil || !e.agentHub.IsOnline(host.ID) {
			err = fmt.Errorf("Agent 未在线")
			fmt.Printf("[CommandExecutor] Agent not online for script execution - hostID: %d\n", host.ID)
		} else {
			output, err = e.executeScriptViaAgent(ctx, host.ID, req, timeout)
		}
	case "ssh":
		output, err = e.executeScriptViaSSH(ctx, host, req, timeout)
	case "auto":
		fallthrough
	default:
		if e.agentHub != nil && e.agentHub.IsOnline(host.ID) {
			fmt.Printf("[CommandExecutor] Using Agent for script - hostID: %d\n", host.ID)
			output, err = e.executeScriptViaAgent(ctx, host.ID, req, timeout)
		} else {
			fmt.Printf("[CommandExecutor] Using SSH for script - hostID: %d\n", host.ID)
			output, err = e.executeScriptViaSSH(ctx, host, req, timeout)
		}
	}

	duration := time.Since(startTime).Seconds()
	fmt.Printf("[CommandExecutor] ExecuteScript completed - hostID: %d, duration: %.2fs, hasError: %v\n",
		host.ID, duration, err != nil)

	return &ExecuteResult{
		Output:   output,
		Error:    err,
		Duration: duration,
	}
}

// executeScriptViaSSH 通过 SSH 执行脚本
func (e *CommandExecutor) executeScriptViaSSH(ctx context.Context, host *assetbiz.Host, req *ScriptExecuteRequest, timeout int) (string, error) {
	fmt.Printf("[CommandExecutor] executeScriptViaSSH - hostID: %d, scriptType: %s\n", host.ID, req.ScriptType)

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

	// 生成脚本文件名（基于内容的 MD5）
	scriptContent := req.ScriptContent
	if scriptContent == "" && req.ScriptFile != "" {
		// TODO: 如果是上传的文件，需要读取文件内容
		return "", fmt.Errorf("暂不支持上传的脚本文件")
	}

	hash := md5.Sum([]byte(scriptContent))
	scriptHash := hex.EncodeToString(hash[:])

	// 确定脚本文件扩展名和执行器
	var scriptExt, executor string
	switch req.ScriptType {
	case "shell":
		scriptExt = ".sh"
		executor = "/bin/bash"
	case "python":
		scriptExt = ".py"
		executor = "/usr/bin/python3"
	case "binary":
		scriptExt = ""
		executor = ""
	default:
		return "", fmt.Errorf("不支持的脚本类型: %s", req.ScriptType)
	}

	// 脚本存储目录
	scriptDir := "$HOME/.opshub_scripts"
	scriptName := fmt.Sprintf("script_%s%s", scriptHash[:16], scriptExt)
	scriptPath := filepath.Join(scriptDir, scriptName)

	fmt.Printf("[CommandExecutor] Script path: %s\n", scriptPath)

	// 获取用户 HOME 目录的绝对路径
	homeCmd := "echo $HOME"
	homeDir, err := client.Execute(homeCmd)
	if err != nil {
		return "", fmt.Errorf("获取 HOME 目录失败: %w", err)
	}
	homeDir = strings.TrimSpace(homeDir)
	absScriptDir := filepath.Join(homeDir, ".opshub_scripts")
	absScriptPath := filepath.Join(absScriptDir, scriptName)

	// 检查脚本是否已存在（使用绝对路径）
	checkCmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'not_exists'", absScriptPath)
	checkResult, _ := client.Execute(checkCmd)

	if strings.TrimSpace(checkResult) != "exists" {
		fmt.Printf("[CommandExecutor] Script not exists, uploading...\n")

		fmt.Printf("[CommandExecutor] Absolute script dir: %s, script path: %s\n", absScriptDir, absScriptPath)

		// 创建脚本目录
		mkdirCmd := fmt.Sprintf("mkdir -p %s", absScriptDir)
		if _, err := client.Execute(mkdirCmd); err != nil {
			return "", fmt.Errorf("创建脚本目录失败: %w", err)
		}

		// 上传脚本内容
		if err := client.UploadFromReader(bytes.NewReader([]byte(scriptContent)), absScriptPath); err != nil {
			return "", fmt.Errorf("上传脚本失败: %w", err)
		}

		// 赋予执行权限
		chmodCmd := fmt.Sprintf("chmod +x %s", absScriptPath)
		if _, err := client.Execute(chmodCmd); err != nil {
			return "", fmt.Errorf("设置脚本权限失败: %w", err)
		}

		fmt.Printf("[CommandExecutor] Script uploaded successfully\n")
	} else {
		fmt.Printf("[CommandExecutor] Script already exists, skipping upload\n")
	}

	// 执行脚本（使用绝对路径）
	var execCmd string
	if req.ScriptType == "binary" {
		if req.ScriptArgs != "" {
			execCmd = fmt.Sprintf("%s %s", absScriptPath, req.ScriptArgs)
		} else {
			execCmd = absScriptPath
		}
	} else {
		if req.ScriptArgs != "" {
			execCmd = fmt.Sprintf("%s %s %s", executor, absScriptPath, req.ScriptArgs)
		} else {
			execCmd = fmt.Sprintf("%s %s", executor, absScriptPath)
		}
	}

	fmt.Printf("[CommandExecutor] Executing: %s\n", execCmd)
	output, err := client.Execute(execCmd)
	if err != nil {
		return output, fmt.Errorf("脚本执行失败: %w", err)
	}

	return output, nil
}

// executeScriptViaAgent 通过 Agent 执行脚本
func (e *CommandExecutor) executeScriptViaAgent(ctx context.Context, hostID uint, req *ScriptExecuteRequest, timeout int) (string, error) {
	fmt.Printf("[CommandExecutor] executeScriptViaAgent - hostID: %d, scriptType: %s\n", hostID, req.ScriptType)

	as, ok := e.agentHub.GetByHostID(hostID)
	if !ok {
		return "", fmt.Errorf("Agent 连接不存在")
	}

	// 生成脚本文件名（基于内容的 MD5）
	scriptContent := req.ScriptContent
	if scriptContent == "" && req.ScriptFile != "" {
		// TODO: 如果是上传的文件，需要读取文件内容
		return "", fmt.Errorf("暂不支持上传的脚本文件")
	}

	hash := md5.Sum([]byte(scriptContent))
	scriptHash := hex.EncodeToString(hash[:])

	// 确定脚本文件扩展名和执行器
	var scriptExt, executor string
	switch req.ScriptType {
	case "shell":
		scriptExt = ".sh"
		executor = "/bin/bash"
	case "python":
		scriptExt = ".py"
		executor = "/usr/bin/python3"
	case "binary":
		scriptExt = ""
		executor = ""
	default:
		return "", fmt.Errorf("不支持的脚本类型: %s", req.ScriptType)
	}

	// 获取 HOME 目录的绝对路径
	homeCmd := "echo $HOME"
	homeDir, err := e.executeViaAgent(ctx, hostID, homeCmd, 10)
	if err != nil {
		return "", fmt.Errorf("获取 HOME 目录失败: %w", err)
	}
	homeDir = strings.TrimSpace(homeDir)

	// 脚本存储目录（使用绝对路径）
	scriptName := fmt.Sprintf("script_%s%s", scriptHash[:16], scriptExt)
	absScriptDir := filepath.Join(homeDir, ".opshub_scripts")
	absScriptPath := filepath.Join(absScriptDir, scriptName)

	fmt.Printf("[CommandExecutor] Absolute script dir: %s, script path: %s\n", absScriptDir, absScriptPath)

	// 检查脚本是否已存在（使用绝对路径）
	checkCmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'not_exists'", absScriptPath)
	checkResult, _ := e.executeViaAgent(ctx, hostID, checkCmd, 10)

	if strings.TrimSpace(checkResult) != "exists" {
		fmt.Printf("[CommandExecutor] Script not exists, uploading via Agent...\n")

		// 创建脚本目录（使用绝对路径）
		mkdirCmd := fmt.Sprintf("mkdir -p %s", absScriptDir)
		if _, err := e.executeViaAgent(ctx, hostID, mkdirCmd, 10); err != nil {
			return "", fmt.Errorf("创建脚本目录失败: %w", err)
		}

		// 通过 Agent 上传文件
		requestID := uuid.New().String()
		fileReq := &agentproto.FileRequest{
			RequestId: requestID,
			Action:    "upload",
			Path:      absScriptPath,
			Data:      []byte(scriptContent),
		}

		msg := &agentproto.ServerMessage{
			Payload: &agentproto.ServerMessage_FileRequest{
				FileRequest: fileReq,
			},
		}

		if err := as.Send(msg); err != nil {
			return "", fmt.Errorf("发送文件上传请求失败: %v", err)
		}

		// 等待上传完成
		timeoutDuration := time.Duration(timeout) * time.Second
		_, err := e.agentHub.WaitResponse(as, requestID, timeoutDuration)
		if err != nil {
			return "", fmt.Errorf("文件上传失败: %v", err)
		}

		// 赋予执行权限（使用绝对路径）
		chmodCmd := fmt.Sprintf("chmod +x %s", absScriptPath)
		if _, err := e.executeViaAgent(ctx, hostID, chmodCmd, 10); err != nil {
			return "", fmt.Errorf("设置脚本权限失败: %w", err)
		}

		fmt.Printf("[CommandExecutor] Script uploaded successfully via Agent\n")
	} else {
		fmt.Printf("[CommandExecutor] Script already exists, skipping upload\n")
	}

	// 执行脚本（使用绝对路径）
	var execCmd string
	if req.ScriptType == "binary" {
		if req.ScriptArgs != "" {
			execCmd = fmt.Sprintf("%s %s", absScriptPath, req.ScriptArgs)
		} else {
			execCmd = absScriptPath
		}
	} else {
		if req.ScriptArgs != "" {
			execCmd = fmt.Sprintf("%s %s %s", executor, absScriptPath, req.ScriptArgs)
		} else {
			execCmd = fmt.Sprintf("%s %s", executor, absScriptPath)
		}
	}

	fmt.Printf("[CommandExecutor] Executing via Agent: %s\n", execCmd)
	output, err := e.executeViaAgent(ctx, hostID, execCmd, timeout)
	if err != nil {
		return output, fmt.Errorf("脚本执行失败: %w", err)
	}

	return output, nil
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
