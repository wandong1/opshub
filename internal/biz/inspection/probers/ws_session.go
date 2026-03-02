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

// WSSession manages a WebSocket connection lifecycle for workflow steps.
type WSSession struct {
	conn    *websocket.Conn
	resp    *http.Response
	headers map[string]string
}

// NewWSSession dials a WebSocket connection and returns a session.
func NewWSSession(wsURL string, reqHeaders map[string]string, params map[string]string, timeout int, skipVerify bool, proxyURL string) (*WSSession, error) {
	if timeout <= 0 {
		timeout = 10
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: time.Duration(timeout) * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: skipVerify},
	}
	if proxyURL != "" {
		if pu, err := url.Parse(proxyURL); err == nil {
			dialer.Proxy = http.ProxyURL(pu)
		}
	}

	header := http.Header{}
	for k, v := range reqHeaders {
		header.Set(k, v)
	}

	// Append query params to URL
	dialURL := wsURL
	if len(params) > 0 {
		qp := url.Values{}
		for k, v := range params {
			qp.Set(k, v)
		}
		sep := "?"
		if strings.Contains(dialURL, "?") {
			sep = "&"
		}
		dialURL += sep + qp.Encode()
	}

	conn, resp, err := dialer.Dial(dialURL, header)
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	if resp != nil {
		for k := range resp.Header {
			headers[k] = resp.Header.Get(k)
		}
	}

	return &WSSession{conn: conn, resp: resp, headers: headers}, nil
}

// Send writes a message to the WebSocket connection.
func (s *WSSession) Send(messageType int, data string) error {
	if messageType == 0 {
		messageType = websocket.TextMessage
	}
	return s.conn.WriteMessage(messageType, []byte(data))
}

// Receive reads one message from the WebSocket connection with a timeout.
func (s *WSSession) Receive(timeout int) (int, string, error) {
	if timeout > 0 {
		s.conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	}
	msgType, data, err := s.conn.ReadMessage()
	if err != nil {
		return 0, "", err
	}
	return msgType, string(data), nil
}

// ReceiveAll reads messages in a loop until timeout, returning all collected messages as a JSON array string.
// A timeout error after receiving ≥1 message is treated as normal completion.
func (s *WSSession) ReceiveAll(timeout int) (string, error) {
	if timeout <= 0 {
		timeout = 5
	}
	s.conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))

	var messages []string
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			if len(messages) > 0 {
				// Timeout or close after collecting messages — normal end
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					break
				}
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					break
				}
				// Other error, still return what we have
				break
			}
			return "", err
		}
		messages = append(messages, string(data))
	}

	b, _ := json.Marshal(messages)
	return string(b), nil
}

// Close gracefully closes the WebSocket connection.
func (s *WSSession) Close() error {
	s.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return s.conn.Close()
}

// UpgradeHeaders returns the upgrade response headers.
func (s *WSSession) UpgradeHeaders() map[string]string {
	return s.headers
}

// StatusCode returns the HTTP status code from the upgrade response.
func (s *WSSession) StatusCode() int {
	if s.resp != nil {
		return s.resp.StatusCode
	}
	return 0
}

// RawHeader returns the raw http.Header from the upgrade response.
func (s *WSSession) RawHeader() http.Header {
	if s.resp != nil {
		return s.resp.Header
	}
	return http.Header{}
}
