package prober

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WsSessionManager 管理 WebSocket 会话
type WsSessionManager struct {
	sessions map[string]*websocket.Conn
	mu       sync.RWMutex
}

// NewWsSessionManager 创建会话管理器
func NewWsSessionManager() *WsSessionManager {
	return &WsSessionManager{
		sessions: make(map[string]*websocket.Conn),
	}
}

// OpenSession 打开 WebSocket 连接
func (m *WsSessionManager) OpenSession(sessionID, wsURL string, headers, params map[string]string, timeout int32, skipVerify bool, proxyURL string) (int, map[string]string, error) {
	// 添加查询参数
	if len(params) > 0 {
		urlParams := url.Values{}
		for k, v := range params {
			urlParams.Set(k, v)
		}
		if len(urlParams) > 0 {
			sep := "?"
			if len(wsURL) > 0 && wsURL[len(wsURL)-1] != '?' {
				sep = "?"
			}
			wsURL += sep + urlParams.Encode()
		}
	}

	// 创建 dialer
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipVerify,
		},
		HandshakeTimeout: time.Duration(timeout) * time.Second,
	}

	// 设置代理
	if proxyURL != "" {
		pURL, err := url.Parse(proxyURL)
		if err == nil {
			dialer.Proxy = http.ProxyURL(pURL)
		}
	}

	// 设置请求头
	httpHeaders := http.Header{}
	for k, v := range headers {
		httpHeaders.Set(k, v)
	}

	// 连接 WebSocket
	conn, resp, err := dialer.Dial(wsURL, httpHeaders)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return statusCode, nil, fmt.Errorf("WebSocket dial failed: %v", err)
	}

	// 保存连接
	m.mu.Lock()
	m.sessions[sessionID] = conn
	m.mu.Unlock()

	// 提取响应头
	respHeaders := make(map[string]string)
	if resp != nil {
		for k, v := range resp.Header {
			if len(v) > 0 {
				respHeaders[k] = v[0]
			}
		}
	}

	statusCode := 101
	if resp != nil {
		statusCode = resp.StatusCode
	}

	return statusCode, respHeaders, nil
}

// SendMessage 发送消息
func (m *WsSessionManager) SendMessage(sessionID string, messageType int, message string) error {
	m.mu.RLock()
	conn, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	msgType := websocket.TextMessage
	if messageType == 2 {
		msgType = websocket.BinaryMessage
	}

	return conn.WriteMessage(msgType, []byte(message))
}

// ReceiveMessage 接收消息
func (m *WsSessionManager) ReceiveMessage(sessionID string, readTimeout int32, receiveMode string) (string, error) {
	m.mu.RLock()
	conn, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}

	timeout := readTimeout
	if timeout <= 0 {
		timeout = 5
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	conn.SetReadDeadline(deadline)

	if receiveMode == "stream" {
		// 收集所有消息直到超时
		var msgs []string
		var totalLen int
		const maxTotal = 65536
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				// 超时是正常的流结束
				break
			}
			if totalLen+len(msg) > maxTotal {
				msg = msg[:maxTotal-totalLen]
				msgs = append(msgs, string(msg))
				break
			}
			totalLen += len(msg)
			msgs = append(msgs, string(msg))
		}
		// 返回 JSON 数组格式
		result := "["
		for i, m := range msgs {
			if i > 0 {
				result += ","
			}
			result += fmt.Sprintf("%q", m)
		}
		result += "]"
		return result, nil
	}

	// 单条消息模式
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("receive failed: %v", err)
	}

	// 截取前 4KB
	if len(msg) > 4096 {
		msg = msg[:4096]
	}

	return string(msg), nil
}

// CloseSession 关闭会话
func (m *WsSessionManager) CloseSession(sessionID string) error {
	m.mu.Lock()
	conn, exists := m.sessions[sessionID]
	if exists {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// 发送关闭消息
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return conn.Close()
}

// CloseAll 关闭所有会话
func (m *WsSessionManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for sessionID, conn := range m.sessions {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn.Close()
		delete(m.sessions, sessionID)
	}
}
