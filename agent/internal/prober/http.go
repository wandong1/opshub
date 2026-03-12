package prober

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// HTTPProber HTTP/HTTPS 拨测器
type HTTPProber struct{}

func (p *HTTPProber) Probe(ctx context.Context, req *pb.ProbeRequest) *pb.ProbeResult {
	result := &pb.ProbeResult{}
	start := time.Now()

	// 构建请求
	reqURL := req.Url
	if reqURL == "" {
		reqURL = req.Target
		if req.Port > 0 && req.Port != 80 && req.Port != 443 {
			reqURL = reqURL + ":" + string(rune(req.Port))
		}
		if req.ProbeType == "https" {
			reqURL = "https://" + reqURL
		} else {
			reqURL = "http://" + reqURL
		}
	}

	// 添加查询参数
	if len(req.Params) > 0 {
		params := url.Values{}
		for k, v := range req.Params {
			params.Set(k, v)
		}
		sep := "?"
		if strings.Contains(reqURL, "?") {
			sep = "&"
		}
		reqURL += sep + params.Encode()
	}

	// 创建请求
	method := req.Method
	if method == "" {
		method = "GET"
	}

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		result.Success = false
		result.Error = "build request: " + err.Error()
		result.Latency = float64(time.Since(start).Milliseconds())
		return result
	}

	// 设置 Headers
	if req.ContentType != "" {
		httpReq.Header.Set("Content-Type", req.ContentType)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 性能追踪
	var dnsStart, connectStart, tlsStart time.Time
	var dnsTime, connectTime, tlsTime, ttfb float64

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				dnsTime = float64(time.Since(dnsStart).Milliseconds())
			}
		},
		ConnectStart: func(_, _ string) {
			connectStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			if !connectStart.IsZero() {
				connectTime = float64(time.Since(connectStart).Milliseconds())
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			if !tlsStart.IsZero() {
				tlsTime = float64(time.Since(tlsStart).Milliseconds())
			}
		},
		GotFirstResponseByte: func() {
			ttfb = float64(time.Since(start).Milliseconds())
		},
	}

	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), trace))

	// 创建 HTTP 客户端
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: req.SkipVerify,
		},
	}

	// 设置代理
	if req.ProxyUrl != "" {
		proxyURL, err := url.Parse(req.ProxyUrl)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// 重定向计数器
	redirectCount := 0

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(req.Timeout) * time.Second,
		// 支持自动跟随重定向（最多10次）
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectCount = len(via)
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// 执行请求
	resp, err := client.Do(httpReq)
	if err != nil {
		result.Success = false
		result.Error = "request failed: " + err.Error()
		result.Latency = float64(time.Since(start).Milliseconds())
		return result
	}
	defer resp.Body.Close()

	// 读取响应体
	// 默认限制 4KB，可通过 max_response_body_size 配置，0 表示不限制
	maxSize := int64(4 * 1024) // 默认 4KB
	if req.MaxResponseBodySize > 0 {
		maxSize = int64(req.MaxResponseBodySize)
	} else if req.MaxResponseBodySize == 0 && req.Url != "" {
		// 如果明确设置为 0 且提供了完整 URL（代理场景），则不限制
		maxSize = 10 * 1024 * 1024 // 10MB 上限，防止内存溢出
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		result.Success = false
		result.Error = "read response: " + err.Error()
		result.Latency = float64(time.Since(start).Milliseconds())
		return result
	}

	// 填充结果
	result.HttpStatusCode = int32(resp.StatusCode)
	result.ResponseBody = bodyBytes // 直接使用 bytes，支持二进制内容
	result.ResponseHeaders = make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			result.ResponseHeaders[k] = v[0]
		}
	}

	result.HttpContentLength = resp.ContentLength
	result.ResponseBodyBytes = int32(len(bodyBytes))
	result.FinalUrl = resp.Request.URL.String()
	result.RedirectCount = int32(redirectCount)

	// 性能指标
	result.DnsLookupTime = dnsTime
	result.TcpConnectTimeHttp = connectTime
	result.TlsHandshakeTime = tlsTime
	result.Ttfb = ttfb

	totalTime := float64(time.Since(start).Milliseconds())
	result.HttpResponseTime = totalTime
	result.Latency = totalTime
	result.ContentTransferTime = totalTime - ttfb

	// TLS 信息
	if resp.TLS != nil {
		result.TlsVersion = tlsVersionString(resp.TLS.Version)
		result.TlsCipherSuite = tls.CipherSuiteName(resp.TLS.CipherSuite)
		if len(resp.TLS.PeerCertificates) > 0 {
			result.SslCertNotAfter = resp.TLS.PeerCertificates[0].NotAfter.Unix()
		}
	}

	// 判断成功
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 400

	// 评估断言
	if len(req.Assertions) > 0 {
		p.evaluateAssertions(req.Assertions, result)
	} else {
		result.AssertionSuccess = true
	}

	// 最终成功 = HTTP 成功 + 断言成功
	result.Success = result.Success && result.AssertionSuccess

	return result
}

func (p *HTTPProber) evaluateAssertions(assertions []*pb.ProbeAssertion, result *pb.ProbeResult) {
	assertStart := time.Now()
	result.AssertionResults = make([]*pb.ProbeAssertionResult, 0, len(assertions))
	result.AssertionSuccess = true

	for _, assertion := range assertions {
		ar := &pb.ProbeAssertionResult{
			Name: assertion.Name,
		}

		// 提取实际值
		actual, err := p.extractValue(assertion, result)
		if err != nil {
			ar.Success = false
			ar.Error = err.Error()
			result.AssertionResults = append(result.AssertionResults, ar)
			result.AssertionFailCount++
			result.AssertionSuccess = false
			continue
		}

		ar.Actual = actual

		// 评估条件
		ar.Success = p.evaluateCondition(actual, assertion.Condition, assertion.Value)
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

func (p *HTTPProber) extractValue(assertion *pb.ProbeAssertion, result *pb.ProbeResult) (string, error) {
	switch assertion.Source {
	case "status":
		return string(rune(result.HttpStatusCode)), nil
	case "body":
		// 转换为 string 用于断言
		return string(result.ResponseBody), nil
	case "header":
		if val, ok := result.ResponseHeaders[assertion.Path]; ok {
			return val, nil
		}
		return "", nil
	default:
		return "", nil
	}
}

func (p *HTTPProber) evaluateCondition(actual, condition, expected string) bool {
	switch condition {
	case "==":
		return actual == expected
	case "!=":
		return actual != expected
	case "contains":
		return strings.Contains(actual, expected)
	case "notcontains":
		return !strings.Contains(actual, expected)
	default:
		return false
	}
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
