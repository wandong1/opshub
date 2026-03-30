package inspection_mgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	inspectionbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"github.com/ydcloud-dy/opshub/internal/biz/inspection/probers"
	inspectionmgmtbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection_mgmt"
)

// ProbeExecutor 拨测执行器，用于在巡检项中执行拨测配置
type ProbeExecutor struct {
	probeConfigRepo inspectionbiz.ProbeConfigRepo
}

// NewProbeExecutor 创建拨测执行器
func NewProbeExecutor(probeConfigRepo inspectionbiz.ProbeConfigRepo) *ProbeExecutor {
	return &ProbeExecutor{
		probeConfigRepo: probeConfigRepo,
	}
}

// Execute 执行拨测配置并返回巡检兼容的结果
func (e *ProbeExecutor) Execute(ctx context.Context, probeConfigID uint, timeout int) *inspectionmgmtbiz.ExecuteResult {
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

	// 覆盖超时设置
	if timeout > 0 {
		config.Timeout = timeout
	}

	// 根据拨测类型执行
	var output string
	var execErr error

	switch config.Type {
	case "ping":
		prober := &probers.PingProber{}
		result := prober.Probe(config.Target, 0, config.Timeout, config.Count, config.PacketSize)
		output = e.formatPingResult(result)
		if !result.Success {
			execErr = fmt.Errorf(result.Error)
		}

	case "tcp":
		prober := &probers.TCPProber{}
		result := prober.Probe(config.Target, config.Port, config.Timeout, 0, 0)
		output = e.formatTCPResult(result)
		if !result.Success {
			execErr = fmt.Errorf(result.Error)
		}

	case "udp":
		prober := &probers.UDPProber{}
		result := prober.Probe(config.Target, config.Port, config.Timeout, 0, 0)
		output = e.formatUDPResult(result)
		if !result.Success {
			execErr = fmt.Errorf(result.Error)
		}

	case "http", "https":
		prober := &probers.HTTPProber{}
		appConfig := e.buildAppConfig(config)
		result := prober.ProbeApp(appConfig)
		output = e.formatHTTPResult(result)
		if !result.Success {
			execErr = fmt.Errorf(result.Error)
		}

	case "websocket":
		prober := &probers.WebSocketProber{}
		appConfig := e.buildAppConfig(config)
		result := prober.ProbeApp(appConfig)
		output = e.formatWebSocketResult(result)
		if !result.Success {
			execErr = fmt.Errorf(result.Error)
		}

	case "workflow":
		// workflow 类型执行
		wfResult := inspectionbiz.ExecuteWorkflowProbe(ctx, config, nil, nil, false)
		output = e.formatWorkflowResult(wfResult)
		if !wfResult.Success {
			execErr = fmt.Errorf(wfResult.Error)
		}

	default:
		return &inspectionmgmtbiz.ExecuteResult{
			Output:   "",
			Error:    fmt.Errorf("不支持的拨测类型: %s", config.Type),
			Duration: time.Since(startTime).Seconds(),
		}
	}

	fmt.Printf("[ProbeExecutor] Probe execution completed: success=%v, duration=%.2fs\n", execErr == nil, time.Since(startTime).Seconds())

	return &inspectionmgmtbiz.ExecuteResult{
		Output:   output,
		Error:    execErr,
		Duration: time.Since(startTime).Seconds(),
	}
}

// buildAppConfig 构建应用层拨测配置
func (e *ProbeExecutor) buildAppConfig(config *inspectionbiz.ProbeConfig) *probers.AppProbeConfig {
	return &probers.AppProbeConfig{
		URL:         config.Target,
		Method:      config.Method,
		Timeout:     config.Timeout,
		SkipVerify:  true,
		WSMessage:   config.WSMessage,
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
