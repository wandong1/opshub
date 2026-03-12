package probers

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

const (
	maxResponseBodyRead  = 10 * 1024 * 1024 // 10MB read limit
	maxResponseBodyStore = 2 * 1024 * 1024  // 2MB storage limit
)

// HTTPProber implements AppProber for HTTP/HTTPS probing.
type HTTPProber struct{}

func (p *HTTPProber) ProbeApp(config *AppProbeConfig) *AppResult {
	result := &AppResult{}
	start := time.Now()

	// Build request
	// Clean URL: remove newlines and other control characters
	reqURL := strings.TrimSpace(config.URL)
	reqURL = strings.ReplaceAll(reqURL, "\n", "")
	reqURL = strings.ReplaceAll(reqURL, "\r", "")
	if len(config.Params) > 0 {
		params := url.Values{}
		for k, v := range config.Params {
			params.Set(k, v)
		}
		sep := "?"
		if strings.Contains(reqURL, "?") {
			sep = "&"
		}
		reqURL += sep + params.Encode()
	}

	var bodyReader io.Reader
	if config.Body != "" {
		bodyReader = strings.NewReader(config.Body)
	}

	req, err := http.NewRequest(config.Method, reqURL, bodyReader)
	if err != nil {
		result.Error = "build request: " + err.Error()
		result.Latency = ms(start)
		return result
	}

	// Set headers
	if config.ContentType != "" {
		req.Header.Set("Content-Type", config.ContentType)
	}
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	// Build transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipVerify},
	}
	if config.ProxyURL != "" {
		proxyURL, err := url.Parse(config.ProxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			result.RedirectCount = len(via)
			return nil
		},
	}

	// Setup httptrace for performance breakdown
	var dnsStart, connectStart, tlsStart, firstByteTime time.Time
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				result.DNSLookupTime = ms(dnsStart)
			}
		},
		ConnectStart: func(network, addr string) {
			connectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			if !connectStart.IsZero() {
				result.TCPConnectTime = ms(connectStart)
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			if !tlsStart.IsZero() {
				result.TLSHandshakeTime = ms(tlsStart)
			}
			if err == nil {
				result.TLSVersion = tlsVersionString(state.Version)
				result.TLSCipherSuite = tls.CipherSuiteName(state.CipherSuite)
				if len(state.PeerCertificates) > 0 {
					result.SSLCertNotAfter = state.PeerCertificates[0].NotAfter.Unix()
				}
			}
		},
		GotFirstResponseByte: func() {
			firstByteTime = time.Now()
		},
	}

	ctx := httptrace.WithClientTrace(context.Background(), trace)
	req = req.WithContext(ctx)

	// Execute request
	httpStart := time.Now()
	redirectStart := httpStart
	resp, err := client.Do(req)
	result.HTTPResponseTime = ms(httpStart)

	if result.RedirectCount > 0 {
		result.RedirectTime = ms(redirectStart)
	}

	if err != nil {
		result.Error = "request failed: " + err.Error()
		result.Latency = ms(start)
		return result
	}
	defer resp.Body.Close()

	result.HTTPStatusCode = resp.StatusCode
	result.FinalURL = resp.Request.URL.String()

	// Calculate TTFB
	if !firstByteTime.IsZero() {
		result.TTFB = float64(firstByteTime.Sub(httpStart).Microseconds()) / 1000.0
	}

	// Read body (limited)
	contentStart := time.Now()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyRead))
	bodyStr := string(bodyBytes)
	result.HTTPContentLength = int64(len(bodyBytes))
	result.ResponseBodyBytes = len(bodyBytes)

	// Calculate content transfer time
	if !firstByteTime.IsZero() {
		result.ContentTransferTime = ms(contentStart)
	}

	// Calculate response header size (approximate)
	headerSize := 0
	for k, vals := range resp.Header {
		for _, v := range vals {
			headerSize += len(k) + len(v) + 4 // ": " + "\r\n"
		}
	}
	result.ResponseHeaderBytes = headerSize

	// Store truncated body
	if len(bodyStr) > maxResponseBodyStore {
		result.ResponseBody = bodyStr[:maxResponseBodyStore]
	} else {
		result.ResponseBody = bodyStr
	}

	// Collect response headers
	result.ResponseHeaders = make(map[string]string)
	for k := range resp.Header {
		result.ResponseHeaders[k] = resp.Header.Get(k)
	}

	// Success: 2xx/3xx
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 400

	// Evaluate assertions
	assertionStart := time.Now()
	if len(config.Assertions) > 0 {
		assertionResults := EvaluateAssertions(config.Assertions, bodyStr, resp.Header)
		result.AssertionResults = assertionResults
		result.AssertionSuccess = true
		result.AssertionEvalTime = ms(assertionStart)

		passCount := 0
		failCount := 0
		for _, ar := range assertionResults {
			if ar.Success {
				passCount++
			} else {
				failCount++
				result.AssertionSuccess = false
				result.Success = false
			}
		}
		result.AssertionPassCount = passCount
		result.AssertionFailCount = failCount
	} else {
		result.AssertionSuccess = true
	}

	result.Latency = ms(start)
	return result
}

func ms(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000.0
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}
