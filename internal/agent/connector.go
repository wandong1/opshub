package agent

import "context"

// AgentConnector 定义Agent连接器接口，供终端、文件、命令执行使用
type AgentConnector interface {
	// IsOnline 检查Agent是否在线
	IsOnline(hostID uint) bool
	// OpenTerminal 打开终端会话
	OpenTerminal(ctx context.Context, hostID uint, sessionID string, cols, rows uint32) error
	// SendTerminalInput 发送终端输入
	SendTerminalInput(hostID uint, sessionID string, data []byte) error
	// ResizeTerminal 调整终端大小
	ResizeTerminal(hostID uint, sessionID string, cols, rows uint32) error
	// CloseTerminal 关闭终端会话
	CloseTerminal(hostID uint, sessionID string) error
	// ListFiles 列出文件
	ListFiles(ctx context.Context, hostID uint, path string) ([]FileEntry, error)
	// UploadFile 上传文件
	UploadFile(ctx context.Context, hostID uint, path string, filename string, data []byte) error
	// DownloadFile 下载文件
	DownloadFile(ctx context.Context, hostID uint, path string) ([]byte, error)
	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, hostID uint, path string) error
	// ExecuteCommand 执行命令
	ExecuteCommand(ctx context.Context, hostID uint, command string, timeout int) (*CommandOutput, error)
}

// FileEntry 文件条目
type FileEntry struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime int64  `json:"modTime"`
	IsDir   bool   `json:"isDir"`
}

// CommandOutput 命令执行结果
type CommandOutput struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error,omitempty"`
}
