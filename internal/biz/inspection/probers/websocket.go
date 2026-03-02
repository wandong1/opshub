package probers

import (
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketProber implements AppProber for WebSocket probing.
type WebSocketProber struct{}

func (p *WebSocketProber) ProbeApp(config *AppProbeConfig) *AppResult {
	result := &AppResult{}
	start := time.Now()

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: time.Duration(timeout) * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: config.SkipVerify},
	}

	if config.ProxyURL != "" {
		proxyURL, err := url.Parse(config.ProxyURL)
		if err == nil {
			dialer.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// Build request headers
	reqHeader := http.Header{}
	for k, v := range config.Headers {
		reqHeader.Set(k, v)
	}

	// Append query params to URL
	dialURL := config.URL
	if len(config.Params) > 0 {
		params := url.Values{}
		for k, v := range config.Params {
			params.Set(k, v)
		}
		sep := "?"
		if strings.Contains(dialURL, "?") {
			sep = "&"
		}
		dialURL += sep + params.Encode()
	}

	// Dial
	httpStart := time.Now()
	conn, resp, err := dialer.Dial(dialURL, reqHeader)
	result.HTTPResponseTime = ms(httpStart)

	if err != nil {
		result.Error = "websocket dial: " + err.Error()
		if resp != nil {
			result.HTTPStatusCode = resp.StatusCode
		}
		result.Latency = ms(start)
		return result
	}
	defer conn.Close()

	result.Success = true
	result.HTTPStatusCode = resp.StatusCode

	// Collect upgrade response headers
	result.ResponseHeaders = make(map[string]string)
	for k := range resp.Header {
		result.ResponseHeaders[k] = resp.Header.Get(k)
	}

	// Send message if configured
	if config.WSMessage != "" {
		msgType := config.WSMessageType
		if msgType == 0 {
			msgType = websocket.TextMessage
		}
		if err := conn.WriteMessage(msgType, []byte(config.WSMessage)); err != nil {
			result.Error = "websocket send: " + err.Error()
			result.Success = false
			result.Latency = ms(start)
			return result
		}
	}

	// Receive response if configured
	if config.WSReadTimeout > 0 || (config.WSMessage != "" && config.WSReadTimeout == 0) {
		readTimeout := config.WSReadTimeout
		if readTimeout <= 0 {
			readTimeout = 5
		}
		conn.SetReadDeadline(time.Now().Add(time.Duration(readTimeout) * time.Second))

		// Loop-receive: collect all messages until timeout
		var messages []string
		for {
			_, msgData, err := conn.ReadMessage()
			if err != nil {
				if len(messages) == 0 {
					// No messages received at all
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						result.Error = "websocket receive: timeout with no messages"
					} else {
						result.Error = "websocket receive: " + err.Error()
					}
					result.Success = false
					result.Latency = ms(start)
					return result
				}
				// Got at least one message — timeout/close is normal end
				break
			}
			messages = append(messages, string(msgData))
		}

		var body string
		if len(messages) == 1 {
			// Single message: keep as-is for backward compatibility
			body = messages[0]
		} else {
			b, _ := json.Marshal(messages)
			body = string(b)
		}
		if len(body) > maxResponseBodyStore {
			result.ResponseBody = body[:maxResponseBodyStore]
		} else {
			result.ResponseBody = body
		}
		result.HTTPContentLength = int64(len(body))
	}

	// Evaluate assertions on response body + upgrade headers
	if len(config.Assertions) > 0 {
		assertionResults := EvaluateAssertions(config.Assertions, result.ResponseBody, resp.Header)
		result.AssertionResults = assertionResults
		result.AssertionSuccess = true
		for _, ar := range assertionResults {
			if !ar.Success {
				result.AssertionSuccess = false
				result.Success = false
				break
			}
		}
	} else {
		result.AssertionSuccess = true
	}

	// Graceful close
	conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	result.Latency = ms(start)
	return result
}
