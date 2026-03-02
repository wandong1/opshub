package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/ydcloud-dy/opshub/agent/internal/client"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// PTYManager 管理PTY会话
type PTYManager struct {
	sessions map[string]*PTYSession
	mu       sync.RWMutex
	grpc     *client.GRPCClient
}

// PTYSession 单个PTY会话
type PTYSession struct {
	cmd  *exec.Cmd
	ptmx *os.File
	done chan struct{}
}

// NewPTYManager 创建PTY管理器
func NewPTYManager(grpc *client.GRPCClient) *PTYManager {
	return &PTYManager{
		sessions: make(map[string]*PTYSession),
		grpc:     grpc,
	}
}

// Open 打开终端会话
func (m *PTYManager) Open(sessionID string, cols, rows uint32) error {
	cmd := exec.Command("/bin/bash", "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("启动PTY失败: %w", err)
	}

	pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})

	session := &PTYSession{cmd: cmd, ptmx: ptmx, done: make(chan struct{})}
	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	// 读取PTY输出并发送到服务端
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				m.grpc.SendMessage(&pb.AgentMessage{
					Payload: &pb.AgentMessage_TermOutput{
						TermOutput: &pb.TerminalOutput{
							SessionId: sessionID,
							Data:      data,
						},
					},
				})
			}
			if err != nil {
				break
			}
		}
		close(session.done)
	}()

	return nil
}

// Input 写入终端输入
func (m *PTYManager) Input(sessionID string, data []byte) error {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}
	_, err := session.ptmx.Write(data)
	return err
}

// Resize 调整终端大小
func (m *PTYManager) Resize(sessionID string, cols, rows uint32) error {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}
	return pty.Setsize(session.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// Close 关闭终端会话
func (m *PTYManager) Close(sessionID string) error {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}
	session.ptmx.Close()
	session.cmd.Process.Kill()
	session.cmd.Wait()
	return nil
}
