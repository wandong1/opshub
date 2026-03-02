package probers

import (
	"crypto/tls"
	"io"
	"net/http"
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
	reqURL := config.URL
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
	}

	// Execute request
	httpStart := time.Now()
	resp, err := client.Do(req)
	result.HTTPResponseTime = ms(httpStart)

	if err != nil {
		result.Error = "request failed: " + err.Error()
		result.Latency = ms(start)
		return result
	}
	defer resp.Body.Close()

	result.HTTPStatusCode = resp.StatusCode

	// Read body (limited)
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyRead))
	bodyStr := string(bodyBytes)
	result.HTTPContentLength = int64(len(bodyBytes))

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
	if len(config.Assertions) > 0 {
		assertionResults := EvaluateAssertions(config.Assertions, bodyStr, resp.Header)
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

	result.Latency = ms(start)
	return result
}

func ms(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000.0
}
