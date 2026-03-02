package probers

import (
	"fmt"
	"net/http"
)

// AppProbeConfig holds the configuration for an application-level probe.
type AppProbeConfig struct {
	URL           string
	Method        string
	Body          string
	ContentType   string
	ProxyURL      string
	Headers       map[string]string
	Params        map[string]string
	Timeout       int
	Assertions    []Assertion
	SkipVerify    bool // true=skip TLS verification (default)
	WSMessage     string // WebSocket message to send
	WSMessageType int    // websocket.TextMessage=1, BinaryMessage=2
	WSReadTimeout int    // seconds to wait for response, 0=no receive
}

// AppResult holds the outcome of an application-level probe.
type AppResult struct {
	Success           bool
	Latency           float64 // total ms including assertion evaluation
	Error             string
	HTTPStatusCode    int
	HTTPResponseTime  float64 // real HTTP response time ms
	HTTPContentLength int64
	AssertionSuccess  bool
	AssertionResults  []AssertionResult
	ResponseBody      string            // truncated to 4KB
	ResponseHeaders   map[string]string
}

// AppProber defines the interface for application-level probing.
type AppProber interface {
	ProbeApp(config *AppProbeConfig) *AppResult
}

// GetAppProber returns the appropriate local AppProber for the given type.
func GetAppProber(probeType string) (AppProber, error) {
	switch probeType {
	case "http", "https":
		return &HTTPProber{}, nil
	case "websocket":
		return &WebSocketProber{}, nil
	default:
		return nil, fmt.Errorf("unknown app probe type: %s", probeType)
	}
}

// ToHeaders converts AppResult.ResponseHeaders to http.Header for assertion evaluation.
func toHTTPHeader(headers map[string]string) http.Header {
	h := make(http.Header)
	for k, v := range headers {
		h.Set(k, v)
	}
	return h
}
