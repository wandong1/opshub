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

	// Performance breakdown metrics
	DNSLookupTime       float64 // DNS resolution time (ms)
	TCPConnectTime      float64 // TCP connection time (ms)
	TLSHandshakeTime    float64 // TLS handshake time (ms)
	TTFB                float64 // Time to first byte (ms)
	ContentTransferTime float64 // Content transfer time (ms)

	// TLS/Certificate information
	TLSVersion      string // TLS version (e.g., "TLS 1.3")
	TLSCipherSuite  string // Cipher suite name
	SSLCertNotAfter int64  // Certificate expiry timestamp

	// HTTP details
	RedirectCount       int     // Number of redirects
	RedirectTime        float64 // Total redirect time (ms)
	FinalURL            string  // Final URL after redirects
	ResponseHeaderBytes int     // Response header size
	ResponseBodyBytes   int     // Response body size

	// Assertion statistics
	AssertionPassCount int     // Number of passed assertions
	AssertionFailCount int     // Number of failed assertions
	AssertionEvalTime  float64 // Assertion evaluation time (ms)
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
