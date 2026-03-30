package prober

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// wsAction mirrors inspection.WsAction (JSON-encoded in ProbeRequest.Body).
type wsAction struct {
	Type        string `json:"type"`        // "send" or "receive"
	Message     string `json:"message"`
	MessageType int    `json:"msgType"`     // 1=text, 2=binary
	ReadTimeout int    `json:"readTimeout"` // seconds
	ReceiveMode string `json:"receiveMode"` // "" / "single" = one message; "stream" = collect all until timeout
}

// wsActionResult mirrors inspection.WsActionResult.
type wsActionResult struct {
	Success      bool    `json:"success"`
	ResponseBody string  `json:"body"`
	Latency      float64 `json:"latency"`
	Error        string  `json:"error"`
}

// WebSocketProber WebSocket 拨测器
type WebSocketProber struct{}

func (p *WebSocketProber) Probe(ctx context.Context, req *pb.ProbeRequest) *pb.ProbeResult {
	result := &pb.ProbeResult{}
	start := time.Now()

	// 构建 WebSocket URL
	wsURL := req.Url
	if wsURL == "" {
		wsURL = req.Target
		if req.Port > 0 && req.Port != 80 && req.Port != 443 {
			wsURL = fmt.Sprintf("%s:%d", wsURL, req.Port)
		}
		wsURL = "ws://" + wsURL
	}

	// 添加查询参数
	if len(req.Params) > 0 {
		params := url.Values{}
		for k, v := range req.Params {
			params.Set(k, v)
		}
		wsURL += "?" + params.Encode()
	}

	// 创建 dialer
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: req.SkipVerify,
		},
		HandshakeTimeout: time.Duration(req.Timeout) * time.Second,
	}

	// 设置代理
	if req.ProxyUrl != "" {
		proxyURL, err := url.Parse(req.ProxyUrl)
		if err == nil {
			dialer.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// 设置请求头
	headers := http.Header{}
	for k, v := range req.Headers {
		headers.Set(k, v)
	}

	// 连接 WebSocket
	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("WebSocket dial failed: %v", err)
		result.Latency = float64(time.Since(start).Milliseconds())
		if resp != nil {
			result.HttpStatusCode = int32(resp.StatusCode)
		}
		return result
	}
	defer conn.Close()

	result.HttpStatusCode = int32(resp.StatusCode)
	result.FinalUrl = wsURL

	// Multi-step mode: Body carries a JSON-encoded []wsAction
	if req.Body != "" {
		var actions []wsAction
		if err := json.Unmarshal([]byte(req.Body), &actions); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("WebSocket multi-step: failed to parse actions: %v", err)
			result.Latency = float64(time.Since(start).Milliseconds())
			return result
		}
		stepResults := make([]wsActionResult, 0, len(actions))
		for _, action := range actions {
			ar := wsActionResult{}
			actionStart := time.Now()
			switch action.Type {
			case "send":
				msgType := websocket.TextMessage
				if action.MessageType == 2 {
					msgType = websocket.BinaryMessage
				}
				if writeErr := conn.WriteMessage(msgType, []byte(action.Message)); writeErr != nil {
					ar.Error = fmt.Sprintf("send failed: %v", writeErr)
				} else {
					ar.Success = true
				}
			case "receive":
				timeout := action.ReadTimeout
				if timeout <= 0 {
					timeout = 5
				}
				deadline := time.Now().Add(time.Duration(timeout) * time.Second)
				conn.SetReadDeadline(deadline)
				if action.ReceiveMode == "stream" {
					// Collect all messages until read deadline
					var msgs []string
					var totalLen int
					const maxTotal = 65536
					for {
						_, msg, readErr := conn.ReadMessage()
						if readErr != nil {
							// Deadline exceeded = normal end of stream
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
					ar.Success = true
					encoded, _ := json.Marshal(msgs)
					ar.ResponseBody = string(encoded)
				} else {
					_, msg, readErr := conn.ReadMessage()
					if readErr != nil {
						ar.Error = fmt.Sprintf("receive failed: %v", readErr)
					} else {
						if len(msg) > 4096 {
							msg = msg[:4096]
						}
						ar.Success = true
						ar.ResponseBody = string(msg)
					}
				}
			}
			ar.Latency = float64(time.Since(actionStart).Milliseconds())
			stepResults = append(stepResults, ar)
		}
		// Encode step results into ResponseBody
		encoded, _ := json.Marshal(stepResults)
		result.ResponseBody = encoded
		// Fill response headers
		result.ResponseHeaders = make(map[string]string)
		for k, v := range resp.Header {
			if len(v) > 0 {
				result.ResponseHeaders[k] = v[0]
			}
		}
		result.Latency = float64(time.Since(start).Milliseconds())
		result.HttpResponseTime = result.Latency
		result.Success = true
		result.AssertionSuccess = true
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return result
	}

	// 如果需要发送消息
	if req.WsMessage != "" {
		messageType := websocket.TextMessage
		if req.WsMessageType == 2 {
			messageType = websocket.BinaryMessage
		}

		err = conn.WriteMessage(messageType, []byte(req.WsMessage))
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("WebSocket write failed: %v", err)
			result.Latency = float64(time.Since(start).Milliseconds())
			return result
		}

		// 如果需要读取响应
		if req.WsReadTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(time.Duration(req.WsReadTimeout) * time.Second))
			_, message, err := conn.ReadMessage()
			if err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("WebSocket read failed: %v", err)
				result.Latency = float64(time.Since(start).Milliseconds())
				return result
			}

			// 截取前 4KB
			if len(message) > 4096 {
				message = message[:4096]
			}
			result.ResponseBody = message // 直接使用 bytes
		}
	}

	// 填充响应头
	result.ResponseHeaders = make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			result.ResponseHeaders[k] = v[0]
		}
	}

	result.Latency = float64(time.Since(start).Milliseconds())
	result.HttpResponseTime = result.Latency
	result.Success = true

	// 评估断言
	if len(req.Assertions) > 0 {
		p.evaluateAssertions(req.Assertions, result)
	} else {
		result.AssertionSuccess = true
	}

	result.Success = result.Success && result.AssertionSuccess

	// 优雅关闭
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	return result
}

func (p *WebSocketProber) evaluateAssertions(assertions []*pb.ProbeAssertion, result *pb.ProbeResult) {
	assertStart := time.Now()
	result.AssertionResults = make([]*pb.ProbeAssertionResult, 0, len(assertions))
	result.AssertionSuccess = true

	for _, assertion := range assertions {
		ar := &pb.ProbeAssertionResult{
			Name: assertion.Name,
		}

		// 提取实际值
		var actual string
		switch assertion.Source {
		case "status":
			actual = fmt.Sprintf("%d", result.HttpStatusCode)
		case "body":
			actual = string(result.ResponseBody) // 转换为 string 用于断言
		case "header":
			if val, ok := result.ResponseHeaders[assertion.Path]; ok {
				actual = val
			}
		}

		ar.Actual = actual

		// 评估条件
		ar.Success = evaluateCondition(actual, assertion.Condition, assertion.Value)
		if ar.Success {
			result.AssertionPassCount++
		} else {
			result.AssertionFailCount++
			result.AssertionSuccess = false
		}

		result.AssertionResults = append(result.AssertionResults, ar)
	}

	result.AssertionEvalTime = float64(time.Since(assertStart).Milliseconds())
}

func evaluateCondition(actual, condition, expected string) bool {
	switch condition {
	case "==":
		return actual == expected
	case "!=":
		return actual != expected
	case "contains":
		return len(actual) > 0 && len(expected) > 0 && contains(actual, expected)
	case "notcontains":
		return !contains(actual, expected)
	default:
		return false
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
