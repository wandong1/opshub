package inspection_mgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	inspectionbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"github.com/ydcloud-dy/opshub/internal/biz/inspection/probers"
	inspectionmgmtbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection_mgmt"
)

// ProbeDetails 拨测详细信息（用于二次断言和前端展示）
type ProbeDetails struct {
	// 基础信息
	ProbeType string `json:"probe_type"` // ping/tcp/udp/http/https/websocket/workflow
	Target    string `json:"target"`     // 解析后的目标地址
	Port      string `json:"port,omitempty"`

	// 性能指标
	LatencyMs float64 `json:"latency_ms"` // 响应时间（毫秒）
	Success   bool    `json:"success"`    // 拨测是否成功

	// HTTP/HTTPS 专属
	StatusCode       int    `json:"status_code,omitempty"`        // HTTP 状态码
	Method           string `json:"method,omitempty"`             // HTTP 方法
	URL              string `json:"url,omitempty"`                // 完整 URL
	RequestHeaders   string `json:"request_headers,omitempty"`   // 请求头（JSON）
	RequestBody      string `json:"request_body,omitempty"`      // 请求体
	ResponseBody     string `json:"response_body,omitempty"`     // 响应体
	ContentLength    int64  `json:"content_length,omitempty"`    // 响应体大小

	// 断言结果
	AssertionResults []AssertionResult `json:"assertion_results,omitempty"` // 原始断言详情
	AssertionPass    int               `json:"assertion_pass"`              // 断言通过数量
	AssertionFail    int               `json:"assertion_fail"`              // 断言失败数量

	// HTTP 性能分解
	DNSLookupTime       float64 `json:"dns_lookup_time,omitempty"`       // DNS 查询时间（毫秒）
	TCPConnectTime      float64 `json:"tcp_connect_time,omitempty"`      // TCP 连接时间（毫秒）
	TLSHandshakeTime    float64 `json:"tls_handshake_time,omitempty"`    // TLS 握手时间（毫秒）
	TTFB                float64 `json:"ttfb,omitempty"`                  // 首字节时间（毫秒）
	ContentTransferTime float64 `json:"content_transfer_time,omitempty"` // 内容传输时间（毫秒）

	// Ping 专属
	PacketLoss     float64 `json:"packet_loss,omitempty"`      // 丢包率（%）
	PingRttAvg     float64 `json:"ping_rtt_avg,omitempty"`     // 平均 RTT（毫秒）
	PingRttMin     float64 `json:"ping_rtt_min,omitempty"`     // 最小 RTT（毫秒）
	PingRttMax     float64 `json:"ping_rtt_max,omitempty"`     // 最大 RTT（毫秒）
	PingPacketsSent int    `json:"ping_packets_sent,omitempty"` // 发送包数
	PingPacketsRecv int    `json:"ping_packets_recv,omitempty"` // 接收包数

	// TCP/UDP 专属
	TCPConnectTimeMs float64 `json:"tcp_connect_time_ms,omitempty"` // TCP 连接时间（毫秒）
	UDPWriteTimeMs   float64 `json:"udp_write_time_ms,omitempty"`   // UDP 写入时间（毫秒）
	UDPReadTimeMs    float64 `json:"udp_read_time_ms,omitempty"`    // UDP 读取时间（毫秒）

	// Workflow 专属
	TotalSteps  int                  `json:"total_steps,omitempty"`  // 总步骤数
	StepResults []map[string]interface{} `json:"step_results,omitempty"` // 步骤结果

	// 错误信息
	Error string `json:"error,omitempty"` // 错误信息
}

// AssertionResult 断言结果
type AssertionResult struct {
	Name    string `json:"name"`              // 断言名称
	Success bool   `json:"success"`           // 是否通过
	Actual  string `json:"actual"`            // 实际值
	Error   string `json:"error,omitempty"`   // 错误信息
}


// ProbeExecutor 拨测执行器，用于在巡检项中执行拨测配置
type ProbeExecutor struct {
	probeConfigRepo  inspectionbiz.ProbeConfigRepo
	variableResolver *inspectionbiz.VariableResolver
}

// NewProbeExecutor 创建拨测执行器
func NewProbeExecutor(probeConfigRepo inspectionbiz.ProbeConfigRepo, variableResolver *inspectionbiz.VariableResolver) *ProbeExecutor {
	return &ProbeExecutor{
		probeConfigRepo:  probeConfigRepo,
		variableResolver: variableResolver,
	}
}

// Execute 执行拨测配置并返回巡检兼容的结果
//
// 参数：
//   - probeConfigID: 拨测配置 ID
//   - timeout: 超时时间（秒），0 表示使用配置中的默认值
//   - extraVars: 外部变量（包含任务调度变量和巡检组变量），用于覆盖拨测配置中的变量
//
// 变量优先级（从高到低）：
// 1. extraVars（任务调度变量 + 巡检组变量）- 最高优先级
// 2. 拨测配置变量（ProbeVariable 表，按分组过滤）
// 3. 全局环境变量（ProbeVariable 表）
// 4. 系统预置变量 - 最低优先级
//
// 返回：ExecuteResult 包含执行结果、输出、错误和耗时
func (e *ProbeExecutor) Execute(ctx context.Context, probeConfigID uint, timeout int, extraVars map[string]string) *inspectionmgmtbiz.ExecuteResult {
	startTime := time.Now()

	// 获取拨测配置
	config, err := e.probeConfigRepo.GetByID(ctx, probeConfigID)
	if err != nil {
		return &inspectionmgmtbiz.ExecuteResult{
			Output:   "",
			Error:    fmt.Errorf("获取拨测配置失败: %v", err),
			Duration: time.Since(startTime).Seconds(),
		}
	}

	fmt.Printf("[ProbeExecutor] Executing probe config: id=%d, type=%s, target=%s\n", config.ID, config.Type, config.Target)

	// 解析配置中的变量（使用 extraVars 作为最高优先级）
	if e.variableResolver != nil && extraVars != nil {
		resolvedConfig, err := e.variableResolver.ResolveConfigWithExtra(ctx, config, extraVars)
		if err != nil {
			fmt.Printf("[ProbeExecutor] Failed to resolve variables: %v\n", err)
		} else {
			config = resolvedConfig
			fmt.Printf("[ProbeExecutor] Config after variable resolution: target=%s, url=%s\n", config.Target, config.URL)
		}
	}

	// 覆盖超时设置
	if timeout > 0 {
		config.Timeout = timeout
	}

	// 根据拨测类型执行
	var output string
	var execErr error
	var probeDetails *ProbeDetails

	switch config.Type {
	case "ping":
		prober := &probers.PingProber{}
		result := prober.Probe(config.Target, 0, config.Timeout, config.Count, config.PacketSize)

		// 构建 ProbeDetails
		probeDetails = &ProbeDetails{
			ProbeType:       "ping",
			Target:          config.Target,
			LatencyMs:       result.Latency,
			Success:         result.Success,
			PacketLoss:      result.PacketLoss,
			PingRttAvg:      result.PingRttAvg,
			PingRttMin:      result.PingRttMin,
			PingRttMax:      result.PingRttMax,
			PingPacketsSent: result.PingPacketsSent,
			PingPacketsRecv: result.PingPacketsRecv,
		}
		if result.Error != "" {
			probeDetails.Error = result.Error
			execErr = fmt.Errorf(result.Error)
		}

		output = e.formatPingResult(result)

	case "tcp":
		prober := &probers.TCPProber{}
		// 将 Port 字符串转换为整数
		port := 0
		if config.Port != "" {
			if p, err := strconv.Atoi(config.Port); err == nil {
				port = p
			}
		}
		result := prober.Probe(config.Target, port, config.Timeout, 0, 0)

		// 构建 ProbeDetails
		probeDetails = &ProbeDetails{
			ProbeType:        "tcp",
			Target:           config.Target,
			Port:             config.Port,
			LatencyMs:        result.Latency,
			Success:          result.Success,
			TCPConnectTimeMs: result.TCPConnectTime,
		}
		if result.Error != "" {
			probeDetails.Error = result.Error
			execErr = fmt.Errorf(result.Error)
		}

		output = e.formatTCPResult(result)

	case "udp":
		prober := &probers.UDPProber{}
		// 将 Port 字符串转换为整数
		port := 0
		if config.Port != "" {
			if p, err := strconv.Atoi(config.Port); err == nil {
				port = p
			}
		}
		result := prober.Probe(config.Target, port, config.Timeout, 0, 0)

		// 构建 ProbeDetails
		probeDetails = &ProbeDetails{
			ProbeType:      "udp",
			Target:         config.Target,
			Port:           config.Port,
			LatencyMs:      result.Latency,
			Success:        result.Success,
			UDPWriteTimeMs: result.UDPWriteTime,
			UDPReadTimeMs:  result.UDPReadTime,
		}
		if result.Error != "" {
			probeDetails.Error = result.Error
			execErr = fmt.Errorf(result.Error)
		}

		output = e.formatUDPResult(result)

	case "http", "https":
		prober := &probers.HTTPProber{}
		appConfig := e.buildAppConfig(config)
		result := prober.ProbeApp(appConfig)

		// 构建断言结果
		var assertionResults []AssertionResult
		assertionPass := 0
		assertionFail := 0
		if result.AssertionResults != nil {
			for _, ar := range result.AssertionResults {
				assertionResults = append(assertionResults, AssertionResult{
					Name:    ar.Name,
					Success: ar.Success,
					Actual:  ar.Actual,
					Error:   ar.Error,
				})
				if ar.Success {
					assertionPass++
				} else {
					assertionFail++
				}
			}
		}

		// 构建 ProbeDetails
		probeDetails = &ProbeDetails{
			ProbeType:           config.Type,
			Target:              config.Target,
			LatencyMs:           result.Latency,
			Success:             result.Success,
			StatusCode:          result.HTTPStatusCode,
			Method:              config.Method,
			URL:                 config.URL,
			RequestHeaders:      config.Headers,
			RequestBody:         config.Body,
			ResponseBody:        result.ResponseBody,
			ContentLength:       result.HTTPContentLength,
			AssertionResults:    assertionResults,
			AssertionPass:       assertionPass,
			AssertionFail:       assertionFail,
			DNSLookupTime:       result.DNSLookupTime,
			TCPConnectTime:      result.TCPConnectTime,
			TLSHandshakeTime:    result.TLSHandshakeTime,
			TTFB:                result.TTFB,
			ContentTransferTime: result.ContentTransferTime,
		}
		if result.Error != "" {
			probeDetails.Error = result.Error
			execErr = fmt.Errorf(result.Error)
		}

		output = e.formatHTTPResult(result)

	case "websocket":
		prober := &probers.WebSocketProber{}
		appConfig := e.buildAppConfig(config)
		result := prober.ProbeApp(appConfig)

		// 构建 ProbeDetails
		probeDetails = &ProbeDetails{
			ProbeType:     "websocket",
			Target:        config.Target,
			URL:           config.URL,
			LatencyMs:     result.Latency,
			Success:       result.Success,
			RequestBody:   config.WSMessage,
			ResponseBody:  result.ResponseBody,
		}
		if result.Error != "" {
			probeDetails.Error = result.Error
			execErr = fmt.Errorf(result.Error)
		}

		output = e.formatWebSocketResult(result)

	case "workflow":
		// workflow 类型执行
		wfResult := inspectionbiz.ExecuteWorkflowProbe(ctx, config, nil, nil, false)

		// 转换 StepResults 为 map 格式
		var stepResults []map[string]interface{}
		for _, step := range wfResult.StepResults {
			stepMap := map[string]interface{}{
				"step_name": step.StepName,
				"success":   step.Success,
				"latency":   step.Latency,
				"error":     step.Error,
			}
			stepResults = append(stepResults, stepMap)
		}

		// 构建 ProbeDetails
		probeDetails = &ProbeDetails{
			ProbeType:   "workflow",
			Target:      config.Target,
			LatencyMs:   wfResult.TotalLatency,
			Success:     wfResult.Success,
			TotalSteps:  len(wfResult.StepResults),
			StepResults: stepResults,
		}
		if wfResult.Error != "" {
			probeDetails.Error = wfResult.Error
			execErr = fmt.Errorf(wfResult.Error)
		}

		output = e.formatWorkflowResult(wfResult)

	default:
		return &inspectionmgmtbiz.ExecuteResult{
			Output:   "",
			Error:    fmt.Errorf("不支持的拨测类型: %s", config.Type),
			Duration: time.Since(startTime).Seconds(),
		}
	}

	fmt.Printf("[ProbeExecutor] Probe execution completed: success=%v, duration=%.2fs\n", execErr == nil, time.Since(startTime).Seconds())

	// 将 ProbeDetails 序列化为 JSON 作为 Output
	// 这样断言验证器可以解析 ProbeDetails 进行二次断言
	var finalOutput string
	if probeDetails != nil {
		detailsJSON, err := json.Marshal(probeDetails)
		if err == nil {
			finalOutput = string(detailsJSON)
		} else {
			// 如果序列化失败，使用原始 output
			finalOutput = output
		}
	} else {
		finalOutput = output
	}

	return &inspectionmgmtbiz.ExecuteResult{
		Output:       finalOutput,
		Error:        execErr,
		Duration:     time.Since(startTime).Seconds(),
		ProbeDetails: probeDetails,
	}
}

// buildAppConfig 构建应用层拨测配置
func (e *ProbeExecutor) buildAppConfig(config *inspectionbiz.ProbeConfig) *probers.AppProbeConfig {
	// 解析 Headers（JSON 字符串 -> map）
	headers := make(map[string]string)
	if config.Headers != "" {
		_ = json.Unmarshal([]byte(config.Headers), &headers)
	}

	// 解析 Params（JSON 字符串 -> map）
	params := make(map[string]string)
	if config.Params != "" {
		_ = json.Unmarshal([]byte(config.Params), &params)
	}

	// 处理 SkipVerify（*bool -> bool）
	skipVerify := true
	if config.SkipVerify != nil {
		skipVerify = *config.SkipVerify
	}

	return &probers.AppProbeConfig{
		URL:           config.URL,
		Method:        config.Method,
		Headers:       headers,
		Params:        params,
		Body:          config.Body,
		Timeout:       config.Timeout,
		SkipVerify:    skipVerify,
		WSMessage:     config.WSMessage,
		WSReadTimeout: config.Timeout,
	}
}

// formatPingResult 格式化 Ping 结果
func (e *ProbeExecutor) formatPingResult(result *probers.Result) string {
	data := map[string]interface{}{
		"success":       result.Success,
		"latency_ms":    result.Latency,
		"packet_loss":   result.PacketLoss,
		"rtt_avg_ms":    result.PingRttAvg,
		"rtt_min_ms":    result.PingRttMin,
		"rtt_max_ms":    result.PingRttMax,
		"packets_sent":  result.PingPacketsSent,
		"packets_recv":  result.PingPacketsRecv,
	}
	if result.Error != "" {
		data["error"] = result.Error
	}
	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}

// formatTCPResult 格式化 TCP 结果
func (e *ProbeExecutor) formatTCPResult(result *probers.Result) string {
	data := map[string]interface{}{
		"success":          result.Success,
		"latency_ms":       result.Latency,
		"connect_time_ms":  result.TCPConnectTime,
	}
	if result.Error != "" {
		data["error"] = result.Error
	}
	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}

// formatUDPResult 格式化 UDP 结果
func (e *ProbeExecutor) formatUDPResult(result *probers.Result) string {
	data := map[string]interface{}{
		"success":        result.Success,
		"latency_ms":     result.Latency,
		"write_time_ms":  result.UDPWriteTime,
		"read_time_ms":   result.UDPReadTime,
	}
	if result.Error != "" {
		data["error"] = result.Error
	}
	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}

// formatHTTPResult 格式化 HTTP 结果
func (e *ProbeExecutor) formatHTTPResult(result *probers.AppResult) string {
	data := map[string]interface{}{
		"success":           result.Success,
		"latency_ms":        result.Latency,
		"status_code":       result.HTTPStatusCode,
		"response_time_ms":  result.HTTPResponseTime,
		"content_length":    result.HTTPContentLength,
	}
	if result.Error != "" {
		data["error"] = result.Error
	}
	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}

// formatWebSocketResult 格式化 WebSocket 结果
func (e *ProbeExecutor) formatWebSocketResult(result *probers.AppResult) string {
	data := map[string]interface{}{
		"success":     result.Success,
		"latency_ms":  result.Latency,
	}
	if result.Error != "" {
		data["error"] = result.Error
	}
	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}

// formatWorkflowResult 格式化 Workflow 结果
func (e *ProbeExecutor) formatWorkflowResult(result *inspectionbiz.WorkflowResult) string {
	data := map[string]interface{}{
		"success":       result.Success,
		"latency_ms":    result.TotalLatency,
		"total_steps":   len(result.StepResults),
		"step_results":  result.StepResults,
	}
	if result.Error != "" {
		data["error"] = result.Error
	}
	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}
