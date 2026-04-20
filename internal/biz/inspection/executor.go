package inspection

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tidwall/gjson"
	"github.com/ydcloud-dy/opshub/internal/biz/inspection/probers"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/metrics"
	"github.com/ydcloud-dy/opshub/pkg/scheduler"
	"go.uber.org/zap"
)

// NetworkProbeExecutor implements scheduler.TaskExecutor for network probes.
type NetworkProbeExecutor struct {
	taskRepo         ProbeTaskRepo
	resultRepo       ProbeResultRepo
	pgwRepo          PushgatewayConfigRepo
	groupLookup      func(ctx context.Context, id uint) string // returns group name
	agentFactory     AgentCommandFactory
	variableResolver *VariableResolver
	redisCounter     *metrics.RedisCounter
	teleAIEnabled    bool // global TeleAI auth switch
}

// NewNetworkProbeExecutor creates a new executor.
func NewNetworkProbeExecutor(
	taskRepo ProbeTaskRepo,
	resultRepo ProbeResultRepo,
	pgwRepo PushgatewayConfigRepo,
	groupLookup func(ctx context.Context, id uint) string,
) *NetworkProbeExecutor {
	return &NetworkProbeExecutor{
		taskRepo:    taskRepo,
		resultRepo:  resultRepo,
		pgwRepo:     pgwRepo,
		groupLookup: groupLookup,
	}
}

// SetAgentCommandFactory injects Agent capability.
func (e *NetworkProbeExecutor) SetAgentCommandFactory(f AgentCommandFactory) {
	e.agentFactory = f
}

// SetVariableResolver injects variable resolver capability.
func (e *NetworkProbeExecutor) SetVariableResolver(r *VariableResolver) {
	e.variableResolver = r
}

// SetRedisCounter injects a Redis-backed counter for persistent metric counting.
func (e *NetworkProbeExecutor) SetRedisCounter(rc *metrics.RedisCounter) {
	e.redisCounter = rc
}

// SetTeleAIEnabled sets the global TeleAI Authorization auto-fill switch.
func (e *NetworkProbeExecutor) SetTeleAIEnabled(enabled bool) {
	e.teleAIEnabled = enabled
}

func (e *NetworkProbeExecutor) Type() string { return "network_probe" }

// Execute runs a probe task: load all associated configs -> probe concurrently -> save results -> push metrics.
func (e *NetworkProbeExecutor) Execute(ctx context.Context, task scheduler.Task) error {
	var payload struct {
		TaskID uint `json:"task_id"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	probeTask, err := e.taskRepo.GetByID(ctx, payload.TaskID)
	if err != nil {
		return fmt.Errorf("get probe task: %w", err)
	}

	configs, err := e.taskRepo.GetConfigsByTaskID(ctx, payload.TaskID)
	if err != nil {
		return fmt.Errorf("get probe configs: %w", err)
	}
	if len(configs) == 0 {
		return fmt.Errorf("no probe configs associated with task %d", payload.TaskID)
	}

	concurrency := probeTask.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var failCount int64

	for _, config := range configs {
		wg.Add(1)
		sem <- struct{}{}
		go func(cfg *ProbeConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			// Variable resolution
			resolvedCfg := cfg
			if e.variableResolver != nil {
				if rc, err := e.variableResolver.ResolveConfig(ctx, cfg); err != nil {
					appLogger.Error("resolve variables failed", zap.String("name", cfg.Name), zap.Error(err))
				} else {
					resolvedCfg = rc
				}
			}

			if resolvedCfg.Category == ProbeCategoryApplication {
				e.executeAndSaveAppProbe(ctx, probeTask, cfg, resolvedCfg, &failCount)
			} else if resolvedCfg.Category == ProbeCategoryWorkflow {
				e.executeAndSaveWorkflowProbe(ctx, probeTask, cfg, resolvedCfg, &failCount)
			} else {
				e.executeAndSaveNetworkProbe(ctx, probeTask, cfg, resolvedCfg, &failCount)
			}
		}(config)
	}
	wg.Wait()

	// Auto-cleanup: keep latest 30 * len(configs) results per task
	keepCount := 30 * len(configs)
	if err := e.resultRepo.CleanupByTaskID(ctx, probeTask.ID, keepCount); err != nil {
		appLogger.Error("cleanup probe results failed", zap.Error(err))
	}

	runResult := "success"
	if failCount > 0 {
		runResult = "fail"
	}
	if err := e.taskRepo.UpdateLastRun(ctx, probeTask.ID, runResult); err != nil {
		appLogger.Error("update last run failed", zap.Error(err))
	}

	return nil
}

// executeProbe runs a single probe, choosing local or agent mode based on config.
// Returns the probe result and the agent host ID used (0 for local).
func (e *NetworkProbeExecutor) executeProbe(cfg *ProbeConfig) (*probers.Result, uint) {
	if cfg.ExecMode == ExecModeAgent && e.agentFactory != nil {
		return e.executeViaAgent(cfg)
	}
	// Local execution
	prober, err := probers.GetProber(cfg.Type)
	if err != nil {
		return &probers.Result{Error: err.Error()}, 0
	}
	return prober.Probe(cfg.Target, cfg.Port, cfg.Timeout, cfg.Count, cfg.PacketSize), 0
}

// executeViaAgent picks a random online agent from cfg.AgentHostIDs and runs the probe.
func (e *NetworkProbeExecutor) executeViaAgent(cfg *ProbeConfig) (*probers.Result, uint) {
	hostIDs := parseHostIDs(cfg.AgentHostIDs)
	if len(hostIDs) == 0 {
		return &probers.Result{Error: "no agent host IDs configured"}, 0
	}

	// Filter online agents
	var onlineIDs []uint
	for _, id := range hostIDs {
		if e.agentFactory.IsOnline(id) {
			onlineIDs = append(onlineIDs, id)
		}
	}
	if len(onlineIDs) == 0 {
		return &probers.Result{Error: "no online agent available"}, 0
	}

	// Random pick
	hostID := onlineIDs[rand.Intn(len(onlineIDs))]
	executor, err := e.agentFactory.NewExecutor(hostID)
	if err != nil {
		return &probers.Result{Error: fmt.Sprintf("create agent executor: %v", err)}, hostID
	}

	prober, err := probers.GetAgentProber(cfg.Type, executor)
	if err != nil {
		return &probers.Result{Error: err.Error()}, hostID
	}
	return prober.Probe(cfg.Target, cfg.Port, cfg.Timeout, cfg.Count, cfg.PacketSize), hostID
}

// executeAndSaveNetworkProbe handles network/layer4 probe execution with retry and persistence.
func (e *NetworkProbeExecutor) executeAndSaveNetworkProbe(ctx context.Context, probeTask *ProbeTask, origCfg, resolvedCfg *ProbeConfig, failCount *int64) {
	result, agentHostID := e.executeProbe(resolvedCfg)
	retryAttempt := 0
	for !result.Success && retryAttempt < origCfg.RetryCount {
		retryAttempt++
		appLogger.Info("probe retry", zap.String("name", origCfg.Name), zap.Int("attempt", retryAttempt))
		result, agentHostID = e.executeProbe(resolvedCfg)
	}
	result.RetryAttempt = retryAttempt
	dbResult := &ProbeResult{
		ProbeTaskID:     probeTask.ID,
		ProbeConfigID:   origCfg.ID,
		Success:         result.Success,
		Latency:         result.Latency,
		PacketLoss:      result.PacketLoss,
		PingRttAvg:      result.PingRttAvg,
		PingRttMin:      result.PingRttMin,
		PingRttMax:      result.PingRttMax,
		PingStddev:      result.PingStddev,
		PingPacketsSent: result.PingPacketsSent,
		PingPacketsRecv: result.PingPacketsRecv,
		TCPConnectTime:  result.TCPConnectTime,
		UDPWriteTime:    result.UDPWriteTime,
		UDPReadTime:     result.UDPReadTime,
		ErrorMessage:    result.Error,
		AgentHostID:     agentHostID,
		RetryAttempt:    retryAttempt,
		TriggerType:     probeTask.TriggerType,
	}
	if err := e.resultRepo.Create(ctx, dbResult); err != nil {
		appLogger.Error("save probe result failed", zap.Error(err))
	}
	if !result.Success {
		atomic.AddInt64(failCount, 1)
	}
	if probeTask.PushgatewayID > 0 {
		e.pushMetrics(ctx, probeTask, origCfg, result)
	}
}

// executeAndSaveAppProbe handles application probe execution with retry and persistence.
func (e *NetworkProbeExecutor) executeAndSaveAppProbe(ctx context.Context, probeTask *ProbeTask, origCfg, resolvedCfg *ProbeConfig, failCount *int64) {
	appResult, agentHostID := e.executeAppProbe(resolvedCfg)
	retryAttempt := 0
	for !appResult.Success && retryAttempt < origCfg.RetryCount {
		retryAttempt++
		appLogger.Info("app probe retry", zap.String("name", origCfg.Name), zap.Int("attempt", retryAttempt))
		appResult, agentHostID = e.executeAppProbe(resolvedCfg)
	}
	assertionDetail := ""
	if len(appResult.AssertionResults) > 0 {
		if b, err := json.Marshal(appResult.AssertionResults); err == nil {
			assertionDetail = string(b)
		}
	}
	dbResult := &ProbeResult{
		ProbeTaskID:       probeTask.ID,
		ProbeConfigID:     origCfg.ID,
		Success:           appResult.Success,
		Latency:           appResult.Latency,
		HTTPStatusCode:    appResult.HTTPStatusCode,
		HTTPResponseTime:  appResult.HTTPResponseTime,
		HTTPContentLength: appResult.HTTPContentLength,
		AssertionSuccess:  appResult.AssertionSuccess,
		AssertionDetail:   assertionDetail,
		ErrorMessage:      appResult.Error,
		AgentHostID:       agentHostID,
		RetryAttempt:      retryAttempt,

		// Performance breakdown
		DNSLookupTime:       appResult.DNSLookupTime,
		HTTPTCPConnectTime:  appResult.TCPConnectTime,
		TLSHandshakeTime:    appResult.TLSHandshakeTime,
		TTFB:                appResult.TTFB,
		ContentTransferTime: appResult.ContentTransferTime,

		// TLS/Certificate
		TLSVersion:      appResult.TLSVersion,
		TLSCipherSuite:  appResult.TLSCipherSuite,
		SSLCertNotAfter: appResult.SSLCertNotAfter,

		// HTTP details
		RedirectCount:       appResult.RedirectCount,
		RedirectTime:        appResult.RedirectTime,
		FinalURL:            appResult.FinalURL,
		ResponseHeaderBytes: appResult.ResponseHeaderBytes,
		ResponseBodyBytes:   appResult.ResponseBodyBytes,

		// Assertion statistics
		AssertionPassCount: appResult.AssertionPassCount,
		AssertionFailCount: appResult.AssertionFailCount,
		AssertionEvalTime:  appResult.AssertionEvalTime,
		TriggerType:        probeTask.TriggerType,
	}
	if err := e.resultRepo.Create(ctx, dbResult); err != nil {
		appLogger.Error("save app probe result failed", zap.Error(err))
	}
	if !appResult.Success {
		atomic.AddInt64(failCount, 1)
	}
	if probeTask.PushgatewayID > 0 {
		e.pushAppMetrics(ctx, probeTask, origCfg, appResult, retryAttempt)
	}
}

// executeAppProbe runs a single application probe.
func (e *NetworkProbeExecutor) executeAppProbe(cfg *ProbeConfig) (*probers.AppResult, uint) {
	appCfg := buildAppProbeConfig(cfg)
	if e.teleAIEnabled && cfg.TeleAIEnabled {
		InjectTeleAIAuthHeader(true, cfg.TeleAIAppKey, cfg.TeleAIRegion, appCfg)
	}

	appLogger.Info("executeAppProbe called",
		zap.String("probe_name", cfg.Name),
		zap.String("exec_mode", cfg.ExecMode),
		zap.String("agent_host_ids", cfg.AgentHostIDs),
		zap.String("type", cfg.Type),
	)

	// Agent mode
	if cfg.ExecMode == ExecModeAgent {
		appLogger.Info("Agent mode detected for application probe",
			zap.String("probe_name", cfg.Name),
			zap.String("agent_host_ids", cfg.AgentHostIDs),
		)

		if e.agentFactory == nil {
			appLogger.Error("Agent factory not initialized",
				zap.String("probe_name", cfg.Name),
			)
			return &probers.AppResult{
				Error: "agent factory not initialized",
			}, 0
		}

		// Parse agent host IDs
		hostIDs := parseHostIDs(cfg.AgentHostIDs)
		appLogger.Info("Parsed agent host IDs",
			zap.String("probe_name", cfg.Name),
			zap.Uints("host_ids", hostIDs),
		)

		if len(hostIDs) == 0 {
			appLogger.Error("No agent host specified for agent mode",
				zap.String("probe_name", cfg.Name),
			)
			return &probers.AppResult{
				Error: "no agent host specified for agent mode",
			}, 0
		}

		// Filter online agents
		var onlineIDs []uint
		for _, id := range hostIDs {
			if e.agentFactory.IsOnline(id) {
				onlineIDs = append(onlineIDs, id)
			}
		}

		if len(onlineIDs) == 0 {
			appLogger.Error("No online agent available",
				zap.String("probe_name", cfg.Name),
			)
			return &probers.AppResult{
				Error: "no online agent available",
			}, 0
		}

		// Random pick an online agent
		hostID := onlineIDs[rand.Intn(len(onlineIDs))]
		appLogger.Info("Selected agent for application probe",
			zap.String("probe_name", cfg.Name),
			zap.Uint("host_id", hostID),
		)

		// Build ProbeRequest
		probeReq := BuildProbeRequest(cfg, appCfg)

		appLogger.Info("Sending probe request to agent",
			zap.String("probe_name", cfg.Name),
			zap.Uint("host_id", hostID),
			zap.String("request_id", probeReq.RequestId),
			zap.String("type", cfg.Type),
		)

		// Send probe request to agent
		pbResult, err := e.agentFactory.SendProbeRequest(hostID, probeReq)
		if err != nil {
			appLogger.Error("Failed to send probe request to agent",
				zap.String("probe_name", cfg.Name),
				zap.Uint("host_id", hostID),
				zap.Error(err),
			)
			return &probers.AppResult{
				Success: false,
				Error:   fmt.Sprintf("send probe request failed: %v", err),
			}, hostID
		}

		// Convert pb.ProbeResult to probers.AppResult
		appResult := ConvertProbeResultToAppResult(pbResult)

		appLogger.Info("Received probe result from agent",
			zap.String("probe_name", cfg.Name),
			zap.Uint("host_id", hostID),
			zap.Bool("success", appResult.Success),
			zap.Float64("latency", appResult.Latency),
		)

		return appResult, hostID
	}

	// Proxy mode: use local prober with proxy
	if cfg.ExecMode == ExecModeProxy {
		appLogger.Info("Proxy mode detected, executing locally with proxy",
			zap.String("probe_name", cfg.Name),
			zap.String("proxy_url", cfg.ProxyURL),
		)
		prober, err := probers.GetAppProber(cfg.Type)
		if err != nil {
			return &probers.AppResult{Error: err.Error()}, 0
		}
		return prober.ProbeApp(appCfg), 0
	}

	// Local execution
	appLogger.Info("Local mode detected, executing locally",
		zap.String("probe_name", cfg.Name),
	)
	prober, err := probers.GetAppProber(cfg.Type)
	if err != nil {
		return &probers.AppResult{Error: err.Error()}, 0
	}
	return prober.ProbeApp(appCfg), 0
}

// InjectTeleAIAuthHeader injects Authorization into appCfg.Headers when:
// - globally enabled
// - per-probe appKey/region non-empty
// - X-APP-ID header is present in the request
func InjectTeleAIAuthHeader(enabled bool, appKey, region string, appCfg *probers.AppProbeConfig) {
	if !enabled || appKey == "" || region == "" {
		return
	}
	appID := ""
	for k, v := range appCfg.Headers {
		if strings.EqualFold(k, "x-app-id") {
			appID = v
			break
		}
	}
	if appID == "" {
		return
	}
	if appCfg.Headers == nil {
		appCfg.Headers = make(map[string]string)
	}
	auth, err := GenTeleAIHeader(appID, appKey, region, appCfg.Method, appCfg.URL, appCfg.Headers, appCfg.Params)
	if err != nil {
		appLogger.Warn("TeleAI auth generation failed", zap.Error(err))
		return
	}
	appCfg.Headers["Authorization"] = auth
}

// buildAppProbeConfig converts a ProbeConfig to an AppProbeConfig.
func buildAppProbeConfig(cfg *ProbeConfig) *probers.AppProbeConfig {
	appCfg := &probers.AppProbeConfig{
		URL:           cfg.URL,
		Method:        cfg.Method,
		Body:          cfg.Body,
		ContentType:   cfg.ContentType,
		ProxyURL:      cfg.ProxyURL,
		Timeout:       cfg.Timeout,
		SkipVerify:    cfg.SkipVerify == nil || *cfg.SkipVerify,
		WSMessage:     cfg.WSMessage,
		WSMessageType: cfg.WSMessageType,
		WSReadTimeout: cfg.WSReadTimeout,
	}
	if appCfg.URL == "" {
		appCfg.URL = cfg.Target
	}
	if appCfg.Method == "" {
		appCfg.Method = "GET"
	}
	// Parse Headers JSON
	if cfg.Headers != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(cfg.Headers), &headers); err == nil {
			appCfg.Headers = headers
		}
	}
	// Parse Params JSON
	if cfg.Params != "" {
		var params map[string]string
		if err := json.Unmarshal([]byte(cfg.Params), &params); err == nil {
			appCfg.Params = params
		}
	}
	// Parse Assertions JSON
	if cfg.Assertions != "" {
		var assertions []probers.Assertion
		if err := json.Unmarshal([]byte(cfg.Assertions), &assertions); err == nil {
			appCfg.Assertions = assertions
		}
	}
	return appCfg
}

// parseHostIDs parses a comma-separated string or JSON array string of host IDs into a uint slice.
// Supports both formats: "1,2,3" (comma-separated) and "[1,2,3]" (JSON array)
func parseHostIDs(s string) []uint {
	if s == "" {
		return nil
	}
	// Try JSON array format first (from inspection_tasks.agent_host_ids)
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		var ids []uint
		if err := json.Unmarshal([]byte(s), &ids); err == nil {
			return ids
		}
	}
	// Fallback to comma-separated format (from probe_configs.agent_host_ids)
	parts := strings.Split(s, ",")
	ids := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if id, err := strconv.ParseUint(p, 10, 64); err == nil && id > 0 {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

// executeAndSaveWorkflowProbe handles workflow probe execution and persistence.
func (e *NetworkProbeExecutor) executeAndSaveWorkflowProbe(ctx context.Context, probeTask *ProbeTask, origCfg, resolvedCfg *ProbeConfig, failCount *int64) {
	// Bug 修复：使用 resolvedCfg 而不是 origCfg，确保任务级覆盖的 AgentHostIDs 生效
	wfResult := ExecuteWorkflowProbe(ctx, resolvedCfg, e.variableResolver, e.agentFactory, e.teleAIEnabled)
	detail := ""
	if b, err := json.Marshal(wfResult); err == nil {
		detail = string(b)
		// Truncate to avoid DB column overflow (mediumtext = 16MB)
		if len(detail) > 1048576 {
			detail = detail[:1048576] + "...(truncated)"
		}
	}
	dbResult := &ProbeResult{
		ProbeTaskID:   probeTask.ID,
		ProbeConfigID: origCfg.ID,
		Success:       wfResult.Success,
		Latency:       wfResult.TotalLatency,
		ErrorMessage:  wfResult.Error,
		Detail:        detail,
		TriggerType:   probeTask.TriggerType,
	}
	if err := e.resultRepo.Create(ctx, dbResult); err != nil {
		appLogger.Error("save workflow probe result failed", zap.Error(err))
	}
	if !wfResult.Success {
		atomic.AddInt64(failCount, 1)
	}
	if probeTask.PushgatewayID > 0 {
		e.pushWorkflowMetrics(ctx, probeTask, origCfg, wfResult)
	}
}

// ExecuteWorkflowProbe executes a workflow probe: parse definition, run steps sequentially, extract variables.
func ExecuteWorkflowProbe(ctx context.Context, cfg *ProbeConfig, resolver *VariableResolver, agentFactory AgentCommandFactory, teleAIEnabled bool) *WorkflowResult {
	var def WorkflowDefinition
	if err := json.Unmarshal([]byte(cfg.Body), &def); err != nil {
		return &WorkflowResult{Success: false, Error: fmt.Sprintf("parse workflow definition: %v", err)}
	}
	if len(def.Steps) == 0 {
		return &WorkflowResult{Success: false, Error: "workflow has no steps"}
	}

	allowedGroupIDs := parseGroupIDs(cfg.GroupIDs)
	// 添加 cfg.GroupID 到允许的分组列表
	if cfg.GroupID > 0 {
		allowedGroupIDs = append(allowedGroupIDs, cfg.GroupID)
	}

	// Initialize variable context with global variables (lowest priority)
	varCtx := make(map[string]string)

	// 1. 加载全局环境变量（最低优先级）
	if resolver != nil && resolver.variableRepo != nil {
		// 获取所有全局变量（不限制名称）
		globalVars, _, err := resolver.variableRepo.List(ctx, 1, 10000, "", "", "")
		if err == nil {
			for _, v := range globalVars {
				// 检查变量的分组权限
				if len(v.GroupIDs) == 0 || hasGroupAccess(v.GroupIDs, allowedGroupIDs) {
					varCtx[v.Name] = v.Value
				}
			}
		}
	}

	// 2. 加载业务流程定义中的内联变量（覆盖全局变量）
	for k, v := range def.Variables {
		varCtx[k] = v
	}

	result := &WorkflowResult{
		Success:     true,
		StepResults: make([]WorkflowStepResult, 0, len(def.Steps)),
	}
	var totalLatency float64

	// WebSocket session for cross-step WS operations
	var wsSession *probers.WSSession
	defer func() {
		if wsSession != nil {
			wsSession.Close()
		}
	}()

	for i := 0; i < len(def.Steps); i++ {
		step := def.Steps[i]
		stepType := step.StepType
		if stepType == "" {
			stepType = "http"
		}
		stepResult := WorkflowStepResult{
			StepName:  step.Name,
			StepIndex: i,
			StepType:  stepType,
		}

		// Delay before execution
		if step.Delay > 0 {
			select {
			case <-ctx.Done():
				stepResult.Error = "context cancelled"
				stepResult.Skipped = true
				result.StepResults = append(result.StepResults, stepResult)
				result.Success = false
				result.Error = "context cancelled"
				result.TotalLatency = totalLatency
				return result
			case <-timeAfter(step.Delay):
			}
		}

		// Resolve variables in step fields
		resolvedURL := step.URL
		resolvedBody := step.Body
		resolvedHeaders := step.Headers
		resolvedParams := step.Params
		resolvedWSMessage := step.WSMessage
		if resolver != nil {
			if v, err := resolver.ResolveText(ctx, step.URL, varCtx, allowedGroupIDs); err == nil {
				resolvedURL = v
			}
			if v, err := resolver.ResolveText(ctx, step.Body, varCtx, allowedGroupIDs); err == nil {
				resolvedBody = v
			}
			if v, err := resolver.ResolveMap(ctx, step.Headers, varCtx, allowedGroupIDs); err == nil {
				resolvedHeaders = v
			}
			if v, err := resolver.ResolveMap(ctx, step.Params, varCtx, allowedGroupIDs); err == nil {
				resolvedParams = v
			}
			if v, err := resolver.ResolveText(ctx, step.WSMessage, varCtx, allowedGroupIDs); err == nil {
				resolvedWSMessage = v
			}
		}

		stepSkipVerify := step.SkipVerify == nil || *step.SkipVerify
		start := time.Now()

		switch stepType {
		case "ws_connect":
			if wsSession != nil {
				wsSession.Close()
				wsSession = nil
			}
			// Inject TeleAI Authorization into WS handshake headers (step-level switch)
			if teleAIEnabled && step.TeleAIEnabled {
				wsConnCfg := &probers.AppProbeConfig{
					URL:     resolvedURL,
					Method:  "GET",
					Headers: resolvedHeaders,
					Params:  resolvedParams,
				}
				InjectTeleAIAuthHeader(true, step.TeleAIAppKey, step.TeleAIRegion, wsConnCfg)
				resolvedHeaders = wsConnCfg.Headers
			}
			// Agent mode: bundle entire WS flow (connect→all send/receive steps→disconnect)
			// into one ProbeRequest. All ws_send/ws_receive actions are encoded as a JSON
			// array in ProbeRequest.Body; Agent executes them on a single persistent
			// connection and returns per-action results in ProbeResult.ResponseBody.
			wsEffMode := step.ExecMode
			if wsEffMode == "" {
				wsEffMode = cfg.ExecMode
			}
			if wsEffMode == ExecModeAgent && agentFactory != nil {
				// Agent 模式：逐步执行 WebSocket 操作，支持跨步骤变量引用
				// 1. 选择在线 Agent
				agentHostIDs := parseHostIDs(cfg.AgentHostIDs)
				var pickedAgentID uint
				for _, id := range agentHostIDs {
					if agentFactory.IsOnline(id) {
						pickedAgentID = id
						break
					}
				}
				if pickedAgentID == 0 {
					stepResult.Error = "no online agent available for ws_connect"
					stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
					totalLatency += stepResult.Latency
					result.StepResults = append(result.StepResults, stepResult)
					result.Success = false
					if def.StopOnFailure {
						appendSkippedSteps(result, def.Steps, i+1)
						result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, stepResult.Error)
						result.TotalLatency = totalLatency
						return result
					}
					continue
				}

				// 2. 打开 WebSocket 会话
				sessionID := uuid.New().String()
				wsResult, err := agentFactory.SendWsSessionOpen(pickedAgentID, sessionID, resolvedURL, resolvedHeaders, resolvedParams, int32(step.Timeout), stepSkipVerify, step.ProxyURL)
				if err != nil || !wsResult.Success {
					errMsg := err.Error()
					if wsResult != nil && wsResult.Error != "" {
						errMsg = wsResult.Error
					}
					stepResult.Error = "ws_connect: " + errMsg
					stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
					if wsResult != nil {
						stepResult.Latency = wsResult.Latency
						stepResult.HTTPStatusCode = int(wsResult.StatusCode)
						stepResult.ResponseHeaders = wsResult.Headers
					}
					totalLatency += stepResult.Latency
					result.StepResults = append(result.StepResults, stepResult)
					result.Success = false
					if def.StopOnFailure {
						appendSkippedSteps(result, def.Steps, i+1)
						result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, stepResult.Error)
						result.TotalLatency = totalLatency
						return result
					}
					continue
				}

				// 3. ws_connect 成功
				stepResult.Success = true
				stepResult.HTTPStatusCode = int(wsResult.StatusCode)
				stepResult.ResponseHeaders = wsResult.Headers
				stepResult.Latency = wsResult.Latency
				totalLatency += stepResult.Latency

				// 提取变量
				if len(step.Extractions) > 0 {
					extracted := extractWorkflowVars(step.Extractions, &probers.AppResult{ResponseHeaders: wsResult.Headers}, varCtx)
					if len(extracted) > 0 {
						stepResult.ExtractedVars = extracted
					}
				}
				result.StepResults = append(result.StepResults, stepResult)

				// 4. 逐步执行后续的 ws_send/ws_receive 步骤，直到 ws_disconnect
				disconnectIdx := -1
				for j := i + 1; j < len(def.Steps); j++ {
					if def.Steps[j].StepType == "ws_disconnect" {
						disconnectIdx = j
						break
					}
				}

				// 执行 ws_send/ws_receive 步骤
				for j := i + 1; j < len(def.Steps) && (disconnectIdx < 0 || j < disconnectIdx); j++ {
					subStep := def.Steps[j]
					subType := subStep.StepType
					if subType != "ws_send" && subType != "ws_receive" {
						continue
					}

					subResult := WorkflowStepResult{
						StepName:  subStep.Name,
						StepIndex: j,
						StepType:  subType,
						Success:   true,
					}
					subStart := time.Now()

					if subType == "ws_send" {
						// 解析消息内容（此时 varCtx 已包含前面步骤提取的变量）
						msg := subStep.WSMessage
						if resolver != nil {
							if v, err := resolver.ResolveText(ctx, msg, varCtx, allowedGroupIDs); err == nil {
								msg = v
							}
						}
						msgType := subStep.WSMessageType
						if msgType == 0 {
							msgType = 1
						}

						// 发送消息
						actionID := uuid.New().String()
						actionResult, err := agentFactory.SendWsSessionAction(pickedAgentID, sessionID, actionID, "send", msg, int32(msgType), 0, "")
						if err != nil || !actionResult.Success {
							errMsg := err.Error()
							if actionResult != nil && actionResult.Error != "" {
								errMsg = actionResult.Error
							}
							subResult.Success = false
							subResult.Error = errMsg
							result.Success = false
						}
						if actionResult != nil {
							subResult.Latency = actionResult.Latency
						} else {
							subResult.Latency = float64(time.Since(subStart).Microseconds()) / 1000.0
						}
						totalLatency += subResult.Latency

					} else if subType == "ws_receive" {
						// 接收消息
						actionID := uuid.New().String()
						readTimeout := subStep.WSReadTimeout
						if readTimeout == 0 {
							readTimeout = 5
						}
						actionResult, err := agentFactory.SendWsSessionAction(pickedAgentID, sessionID, actionID, "receive", "", 0, int32(readTimeout), subStep.WSReceiveMode)
						if err != nil || !actionResult.Success {
							errMsg := err.Error()
							if actionResult != nil && actionResult.Error != "" {
								errMsg = actionResult.Error
							}
							subResult.Success = false
							subResult.Error = errMsg
							result.Success = false
						} else {
							subResult.ResponseBody = actionResult.ResponseBody

							// 评估断言
							if len(subStep.Assertions) > 0 {
								assertions := toProberAssertions(subStep.Assertions)
								assertionResults := probers.EvaluateAssertions(assertions, actionResult.ResponseBody, nil)
								for idx, assertRes := range assertionResults {
									subResult.AssertionResults = append(subResult.AssertionResults, WorkflowAssertionResult{
										Name: assertRes.Name, Success: assertRes.Success, Actual: assertRes.Actual, Error: assertRes.Error,
										Source: subStep.Assertions[idx].Source, Path: subStep.Assertions[idx].Path,
										Condition: subStep.Assertions[idx].Condition, Value: subStep.Assertions[idx].Value,
									})
									if !assertRes.Success {
										subResult.Success = false
										result.Success = false
									}
								}
							}

							// 提取变量（关键：此时提取的变量会立即更新到 varCtx，供后续步骤使用）
							if len(subStep.Extractions) > 0 {
								extracted := extractWorkflowVars(subStep.Extractions, &probers.AppResult{ResponseBody: actionResult.ResponseBody}, varCtx)
								if len(extracted) > 0 {
									subResult.ExtractedVars = extracted
								}
							}
						}
						if actionResult != nil {
							subResult.Latency = actionResult.Latency
						} else {
							subResult.Latency = float64(time.Since(subStart).Microseconds()) / 1000.0
						}
						totalLatency += subResult.Latency
					}

					result.StepResults = append(result.StepResults, subResult)

					// 如果失败且需要停止
					if !subResult.Success && def.StopOnFailure {
						// 关闭会话
						agentFactory.SendWsSessionClose(pickedAgentID, sessionID)
						if disconnectIdx >= 0 {
							appendSkippedSteps(result, def.Steps, disconnectIdx+1)
						} else {
							appendSkippedSteps(result, def.Steps, j+1)
						}
						result.Error = fmt.Sprintf("step %d (%s) failed: %s", j+1, subStep.Name, subResult.Error)
						result.TotalLatency = totalLatency
						return result
					}
				}

				// 5. 跳转到 ws_disconnect 步骤
				if disconnectIdx >= 0 {
					i = disconnectIdx - 1 // 循环会 i++，所以这里 -1
				}
				continue
			}
			sess, err := probers.NewWSSession(resolvedURL, resolvedHeaders, resolvedParams, step.Timeout, stepSkipVerify, step.ProxyURL)
			if err != nil {
				stepResult.Error = "ws_connect: " + err.Error()
				stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
				totalLatency += stepResult.Latency
				result.StepResults = append(result.StepResults, stepResult)
				result.Success = false
				if def.StopOnFailure {
					appendSkippedSteps(result, def.Steps, i+1)
					result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, err.Error())
					result.TotalLatency = totalLatency
					return result
				}
				continue
			}
			wsSession = sess
			stepResult.Success = true
			stepResult.HTTPStatusCode = sess.StatusCode()
			stepResult.ResponseHeaders = sess.UpgradeHeaders()
			stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
			totalLatency += stepResult.Latency
			// Assertions on upgrade headers
			if len(step.Assertions) > 0 {
				assertions := toProberAssertions(step.Assertions)
				assertionResults := probers.EvaluateAssertions(assertions, "", sess.RawHeader())
				for idx, ar := range assertionResults {
					stepResult.AssertionResults = append(stepResult.AssertionResults, WorkflowAssertionResult{
						Name: ar.Name, Success: ar.Success, Actual: ar.Actual, Error: ar.Error,
						Source: step.Assertions[idx].Source, Path: step.Assertions[idx].Path,
						Condition: step.Assertions[idx].Condition, Value: step.Assertions[idx].Value,
					})
					if !ar.Success {
						stepResult.Success = false
					}
				}
			}
			// Extract variables from headers
			if stepResult.Success || !def.StopOnFailure {
				extracted := extractWorkflowVars(step.Extractions, &probers.AppResult{ResponseHeaders: sess.UpgradeHeaders()}, varCtx)
				if len(extracted) > 0 {
					stepResult.ExtractedVars = extracted
				}
			}

		case "ws_send":
			if wsSession == nil {
				stepResult.Error = "ws_send: no active WebSocket connection"
				stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
				totalLatency += stepResult.Latency
				result.StepResults = append(result.StepResults, stepResult)
				result.Success = false
				if def.StopOnFailure {
					appendSkippedSteps(result, def.Steps, i+1)
					result.Error = fmt.Sprintf("step %d (%s) failed: no WS connection", i+1, step.Name)
					result.TotalLatency = totalLatency
					return result
				}
				continue
			}
			msgType := step.WSMessageType
			if msgType == 0 {
				msgType = 1
			}
			if err := wsSession.Send(msgType, resolvedWSMessage); err != nil {
				stepResult.Error = "ws_send: " + err.Error()
				stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
				totalLatency += stepResult.Latency
				result.StepResults = append(result.StepResults, stepResult)
				result.Success = false
				if def.StopOnFailure {
					appendSkippedSteps(result, def.Steps, i+1)
					result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, err.Error())
					result.TotalLatency = totalLatency
					return result
				}
				continue
			}
			stepResult.Success = true
			stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
			totalLatency += stepResult.Latency

		case "ws_receive":
			if wsSession == nil {
				stepResult.Error = "ws_receive: no active WebSocket connection"
				stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
				totalLatency += stepResult.Latency
				result.StepResults = append(result.StepResults, stepResult)
				result.Success = false
				if def.StopOnFailure {
					appendSkippedSteps(result, def.Steps, i+1)
					result.Error = fmt.Sprintf("step %d (%s) failed: no WS connection", i+1, step.Name)
					result.TotalLatency = totalLatency
					return result
				}
				continue
			}
			readTimeout := step.WSReadTimeout
			if readTimeout <= 0 {
				readTimeout = 5
			}

			receiveMode := step.WSReceiveMode
			if receiveMode == "stream" {
				// Stream mode: collect all messages until timeout
				msgData, err := wsSession.ReceiveAll(readTimeout)
				stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
				totalLatency += stepResult.Latency
				if err != nil {
					stepResult.Error = "ws_receive(stream): " + err.Error()
					result.StepResults = append(result.StepResults, stepResult)
					result.Success = false
					if def.StopOnFailure {
						appendSkippedSteps(result, def.Steps, i+1)
						result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, err.Error())
						result.TotalLatency = totalLatency
						return result
					}
					continue
				}
				stepResult.Success = true
				stepResult.ResponseBody = msgData
			} else {
				// Single mode (default): receive one message
				_, msgData, err := wsSession.Receive(readTimeout)
				stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
				totalLatency += stepResult.Latency
				if err != nil {
					stepResult.Error = "ws_receive: " + err.Error()
					result.StepResults = append(result.StepResults, stepResult)
					result.Success = false
					if def.StopOnFailure {
						appendSkippedSteps(result, def.Steps, i+1)
						result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, err.Error())
						result.TotalLatency = totalLatency
						return result
					}
					continue
				}
				stepResult.Success = true
				stepResult.ResponseBody = msgData
			}

			// Assertions on received message
			if len(step.Assertions) > 0 {
				assertions := toProberAssertions(step.Assertions)
				assertionResults := probers.EvaluateAssertions(assertions, stepResult.ResponseBody, nil)
				for idx, ar := range assertionResults {
					stepResult.AssertionResults = append(stepResult.AssertionResults, WorkflowAssertionResult{
						Name: ar.Name, Success: ar.Success, Actual: ar.Actual, Error: ar.Error,
						Source: step.Assertions[idx].Source, Path: step.Assertions[idx].Path,
						Condition: step.Assertions[idx].Condition, Value: step.Assertions[idx].Value,
					})
					if !ar.Success {
						stepResult.Success = false
					}
				}
			}
			// Extract variables from received message
			if stepResult.Success || !def.StopOnFailure {
				extracted := extractWorkflowVars(step.Extractions, &probers.AppResult{ResponseBody: stepResult.ResponseBody}, varCtx)
				if len(extracted) > 0 {
					stepResult.ExtractedVars = make(map[string]string)
					for k, v := range extracted {
						stepResult.ExtractedVars[k] = v
					}
				}
			}

		case "ws_disconnect":
			if wsSession != nil {
				wsSession.Close()
				wsSession = nil
			}
			stepResult.Success = true
			stepResult.Latency = float64(time.Since(start).Microseconds()) / 1000.0
			totalLatency += stepResult.Latency

		default: // "http" or empty — existing HTTP/WS logic
			appCfg := &probers.AppProbeConfig{
				URL:         resolvedURL,
				Method:      step.Method,
				Body:        resolvedBody,
				ContentType: step.ContentType,
				Headers:     resolvedHeaders,
				Params:      resolvedParams,
				Timeout:     step.Timeout,
				SkipVerify:  stepSkipVerify,
			}
			if appCfg.Method == "" {
				appCfg.Method = "GET"
			}
			if appCfg.Timeout == 0 {
				appCfg.Timeout = cfg.Timeout
			}
				// Use step-level TeleAI config if enabled for this step, falling back to disabled
				if teleAIEnabled && step.TeleAIEnabled {
					InjectTeleAIAuthHeader(true, step.TeleAIAppKey, step.TeleAIRegion, appCfg)
				}
			for _, a := range step.Assertions {
				appCfg.Assertions = append(appCfg.Assertions, probers.Assertion{
					Name: a.Name, Source: a.Source, Path: a.Path,
					Condition: a.Condition, Value: a.Value,
				})
			}
			if step.ProxyURL != "" {
				appCfg.ProxyURL = step.ProxyURL
			}

			probeType := "http"
			if strings.HasPrefix(strings.ToLower(resolvedURL), "wss://") || strings.HasPrefix(strings.ToLower(resolvedURL), "ws://") {
				probeType = "websocket"
			}

			appLogger.Info("Workflow step execution",
				zap.String("workflow_name", cfg.Name),
				zap.String("step_name", step.Name),
				zap.Int("step_index", i),
				zap.String("step_exec_mode", step.ExecMode),
				zap.String("probe_type", probeType),
				zap.String("url", resolvedURL),
			)

			// Determine effective exec mode: step-level overrides cfg-level; empty step.ExecMode inherits cfg.ExecMode
			effectiveExecMode := step.ExecMode
			if effectiveExecMode == "" {
				effectiveExecMode = cfg.ExecMode
			}
			// Check step.ExecMode for agent execution
			var appResult *probers.AppResult
			if effectiveExecMode == ExecModeAgent && agentFactory != nil {
				appLogger.Info("Workflow step using agent mode",
					zap.String("workflow_name", cfg.Name),
					zap.String("step_name", step.Name),
					zap.String("agent_host_ids", cfg.AgentHostIDs),
				)

				// Parse agent host IDs from config
				hostIDs := parseHostIDs(cfg.AgentHostIDs)
				appLogger.Info("Parsed agent host IDs for workflow step",
					zap.String("workflow_name", cfg.Name),
					zap.String("step_name", step.Name),
					zap.Uints("host_ids", hostIDs),
				)

				if len(hostIDs) == 0 {
					appLogger.Error("No agent host specified for workflow step agent mode",
						zap.String("workflow_name", cfg.Name),
						zap.String("step_name", step.Name),
					)
					stepResult.Error = "no agent host specified for agent mode"
					stepResult.Success = false
					result.StepResults = append(result.StepResults, stepResult)
					result.Success = false
					if def.StopOnFailure {
						appendSkippedSteps(result, def.Steps, i+1)
						result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, stepResult.Error)
						result.TotalLatency = totalLatency
						return result
					}
					continue
				}

				// Filter online agents
				var onlineIDs []uint
				for _, id := range hostIDs {
					if agentFactory.IsOnline(id) {
						onlineIDs = append(onlineIDs, id)
					}
				}
				if len(onlineIDs) == 0 {
					stepResult.Error = "no online agent available for workflow step"
					stepResult.Success = false
					result.StepResults = append(result.StepResults, stepResult)
					result.Success = false
					if def.StopOnFailure {
						appendSkippedSteps(result, def.Steps, i+1)
						result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, stepResult.Error)
						result.TotalLatency = totalLatency
						return result
					}
					continue
				}
				// Random pick an online agent
				pickedHostID := onlineIDs[rand.Intn(len(onlineIDs))]
				// Build a temporary ProbeConfig for this step
				stepCfg := &ProbeConfig{
					Type:      probeType,
					Target:    resolvedURL,
					Timeout:   appCfg.Timeout,
					SkipVerify: cfg.SkipVerify,
					WSMessage:     cfg.WSMessage,
					WSMessageType: cfg.WSMessageType,
					WSReadTimeout: cfg.WSReadTimeout,
				}
				probeReq := BuildProbeRequest(stepCfg, appCfg)
				pbResult, err := agentFactory.SendProbeRequest(pickedHostID, probeReq)
				var stepAppResult *probers.AppResult
				if err != nil {
					stepAppResult = &probers.AppResult{Error: err.Error()}
				} else {
					stepAppResult = ConvertProbeResultToAppResult(pbResult)
				}
				appResult = stepAppResult
				appLogger.Info("Workflow step agent execution done",
					zap.String("workflow_name", cfg.Name),
					zap.String("step_name", step.Name),
					zap.Uint("host_id", pickedHostID),
					zap.Bool("success", appResult.Success),
				)
			} else {
				appLogger.Info("Workflow step using local/proxy mode",
					zap.String("workflow_name", cfg.Name),
					zap.String("step_name", step.Name),
					zap.String("exec_mode", step.ExecMode),
				)

				// Local or proxy execution
				prober, err := probers.GetAppProber(probeType)
				if err != nil {
					stepResult.Error = err.Error()
					result.StepResults = append(result.StepResults, stepResult)
					result.Success = false
					if def.StopOnFailure {
						appendSkippedSteps(result, def.Steps, i+1)
						result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, err.Error())
						result.TotalLatency = totalLatency
						return result
					}
					continue
				}

				appResult = prober.ProbeApp(appCfg)
			}

			totalLatency += appResult.Latency

			stepResult.HTTPStatusCode = appResult.HTTPStatusCode
			stepResult.HTTPResponseTime = appResult.HTTPResponseTime
			stepResult.Latency = appResult.Latency
			stepResult.ResponseBody = appResult.ResponseBody
			stepResult.ResponseHeaders = appResult.ResponseHeaders
			stepResult.Success = appResult.Success
			// 需求二：填充请求信息用于前端展示
			stepResult.URL = resolvedURL
			stepResult.Method = appCfg.Method
			stepResult.RequestHeaders = resolvedHeaders
			stepResult.RequestParams = resolvedParams
			stepResult.RequestBody = resolvedBody

			for idx, ar := range appResult.AssertionResults {
				stepResult.AssertionResults = append(stepResult.AssertionResults, WorkflowAssertionResult{
					Name: ar.Name, Success: ar.Success, Actual: ar.Actual, Error: ar.Error,
					Source: step.Assertions[idx].Source, Path: step.Assertions[idx].Path,
					Condition: step.Assertions[idx].Condition, Value: step.Assertions[idx].Value,
				})
			}
			if appResult.Error != "" {
				stepResult.Error = appResult.Error
			}

			// Extract variables from response
			if appResult.Success || !def.StopOnFailure {
				extracted := extractWorkflowVars(step.Extractions, appResult, varCtx)
				if len(extracted) > 0 {
					stepResult.ExtractedVars = extracted
				}
			}
		}

		result.StepResults = append(result.StepResults, stepResult)

		if !stepResult.Success {
			result.Success = false
			if def.StopOnFailure {
				appendSkippedSteps(result, def.Steps, i+1)
				result.Error = fmt.Sprintf("step %d (%s) failed", i+1, step.Name)
				result.TotalLatency = totalLatency
				return result
			}
		}
	}

	result.TotalLatency = totalLatency
	return result
}

// toProberAssertions converts workflow assertions to prober assertions.
func toProberAssertions(assertions []WorkflowStepAssertion) []probers.Assertion {
	result := make([]probers.Assertion, 0, len(assertions))
	for _, a := range assertions {
		result = append(result, probers.Assertion{
			Name: a.Name, Source: a.Source, Path: a.Path,
			Condition: a.Condition, Value: a.Value,
		})
	}
	return result
}

// extractWorkflowVars extracts variables from a step result and updates varCtx.
func extractWorkflowVars(extractions []StepExtraction, appResult *probers.AppResult, varCtx map[string]string) map[string]string {
	extracted := make(map[string]string)
	for _, ext := range extractions {
		val, err := extractStepVariable(ext, appResult)
		if err != nil {
			appLogger.Warn("workflow variable extraction failed",
				zap.String("var", ext.Name), zap.Error(err))
			continue
		}
		extracted[ext.Name] = val
		varCtx[ext.Name] = val
	}
	return extracted
}

// extractStepVariable extracts a variable value from the step's response.
func extractStepVariable(ext StepExtraction, appResult *probers.AppResult) (string, error) {
	switch ext.Source {
	case "response":
		return appResult.ResponseBody, nil
	case "body":
		path := ext.Path
		if strings.HasPrefix(path, "$.") {
			path = path[2:]
		} else if strings.HasPrefix(path, "$") {
			path = path[1:]
		}
		r := gjson.Get(appResult.ResponseBody, path)
		if !r.Exists() {
			return "", fmt.Errorf("path %s not found in response body", ext.Path)
		}
		return r.String(), nil
	case "header":
		if appResult.ResponseHeaders == nil {
			return "", fmt.Errorf("no response headers")
		}
		val, ok := appResult.ResponseHeaders[ext.Path]
		if !ok {
			// Try case-insensitive
			for k, v := range appResult.ResponseHeaders {
				if strings.EqualFold(k, ext.Path) {
					return v, nil
				}
			}
			return "", fmt.Errorf("header %s not found", ext.Path)
		}
		return val, nil
	default:
		return "", fmt.Errorf("unknown extraction source: %s", ext.Source)
	}
}

// appendSkippedSteps marks remaining steps as skipped.
func appendSkippedSteps(result *WorkflowResult, steps []WorkflowStep, fromIndex int) {
	for j := fromIndex; j < len(steps); j++ {
		result.StepResults = append(result.StepResults, WorkflowStepResult{
			StepName:  steps[j].Name,
			StepIndex: j,
			Skipped:   true,
		})
	}
}

// timeAfter returns a channel that fires after d seconds. Extracted for testability.
var timeAfter = func(d int) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		<-time.After(time.Duration(d) * time.Second)
		close(ch)
	}()
	return ch
}

func (e *NetworkProbeExecutor) pushMetrics(ctx context.Context, task *ProbeTask, config *ProbeConfig, result *probers.Result) {
	pgw, err := e.pgwRepo.GetByID(ctx, task.PushgatewayID)
	if err != nil {
		appLogger.Error("pushMetrics: get pushgateway config failed",
			zap.Uint("pgwID", task.PushgatewayID),
			zap.Error(err))
		return
	}
	if pgw.Status != 1 {
		appLogger.Warn("pushMetrics: pushgateway disabled",
			zap.Uint("pgwID", task.PushgatewayID),
			zap.String("pgwURL", pgw.URL))
		return
	}

	pusher := metrics.NewPusher(pgw.URL, pgw.Username, pgw.Password)

	groupName := ""
	if e.groupLookup != nil && task.GroupID > 0 {
		groupName = e.groupLookup(ctx, task.GroupID)
	}
	owner := task.TriggerType // TriggerType is reused; Owner comes from InspectionTask, not ProbeTask
	_ = owner
	scheduleMode := task.TriggerType
	if scheduleMode == "" {
		scheduleMode = "scheduled"
	}

	// 通用标签（用于 Redis Counter key）
	baseLabels := map[string]string{
		"task_id":        fmt.Sprintf("%d", task.ID),
		"task_name":      task.Name,
		"task_type":      config.Type,
		"business_group": groupName,
		"schedule_mode":  scheduleMode,
	}

	// metric label 不含 task_id（已在 grouping 中，避免冲突）
	allLabels := prometheus.Labels{
		"task_name":      task.Name,
		"task_type":      config.Type,
		"business_group": groupName,
		"schedule_mode":  scheduleMode,
		"target":         config.Target,
		"probe_name":     config.Name,
	}

	// 解析自定义 tags
	if config.Tags != "" {
		for _, tag := range strings.Split(config.Tags, ",") {
			parts := strings.SplitN(strings.TrimSpace(tag), "=", 2)
			if len(parts) == 2 {
				allLabels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	labelNames := make([]string, 0, len(allLabels))
	labelValues := make([]string, 0, len(allLabels))
	for k, v := range allLabels {
		labelNames = append(labelNames, k)
		labelValues = append(labelValues, v)
	}

	samples := make([]metrics.MetricSample, 0, 32)
	addGauge := func(name, help string, value float64) {
		samples = append(samples, metrics.MetricSample{
			Name: name, Help: help,
			LabelNames: labelNames, LabelValues: labelValues,
			Value: value,
		})
	}
	addCounter := func(name, help string) {
		if e.redisCounter == nil {
			return
		}
		val := e.redisCounter.Inc(ctx, name, baseLabels)
		samples = append(samples, metrics.MetricSample{
			Name: name, Help: help,
			LabelNames: labelNames, LabelValues: labelValues,
			Value: val,
		})
	}

	// 通用 Counter 指标
	addCounter("srehub_inspect_task_exec_total", "Total task executions")
	if result.Success {
		addCounter("srehub_inspect_task_success_total", "Total task successes")
	} else {
		addCounter("srehub_inspect_task_fail_total", "Total task failures")
	}
	if result.RetryAttempt > 0 {
		for i := 0; i < result.RetryAttempt; i++ {
			if e.redisCounter != nil {
				e.redisCounter.Inc(ctx, "srehub_inspect_task_retry_total", baseLabels)
			}
		}
		if e.redisCounter != nil {
			val := e.redisCounter.Get(ctx, "srehub_inspect_task_retry_total", baseLabels)
			samples = append(samples, metrics.MetricSample{
				Name: "srehub_inspect_task_retry_total", Help: "Total task retries",
				LabelNames: labelNames, LabelValues: labelValues,
				Value: val,
			})
		}
	}

	// 通用 Gauge 指标
	successVal := 0.0
	if result.Success {
		successVal = 1.0
	}
	addGauge("srehub_inspect_task_exec_duration_seconds", "Task execution duration in seconds", result.Latency/1000.0)

	// 可用性（成功/总执行）
	if e.redisCounter != nil {
		total := e.redisCounter.Get(ctx, "srehub_inspect_task_exec_total", baseLabels)
		success := e.redisCounter.Get(ctx, "srehub_inspect_task_success_total", baseLabels)
		if total > 0 {
			addGauge("srehub_inspect_task_availability", "Task availability ratio (success/total)", success/total)
		}
	}

	// 类型专属指标
	switch config.Type {
	case "ping":
		addGauge("srehub_inspect_ping_avg_rtt_seconds", "Ping average RTT seconds", result.PingRttAvg/1000.0)
		addGauge("srehub_inspect_ping_min_rtt_seconds", "Ping min RTT seconds", result.PingRttMin/1000.0)
		addGauge("srehub_inspect_ping_max_rtt_seconds", "Ping max RTT seconds", result.PingRttMax/1000.0)
		addGauge("srehub_inspect_ping_jitter_seconds", "Ping RTT jitter (stddev) seconds", result.PingStddev/1000.0)
		addGauge("srehub_inspect_ping_loss_ratio", "Ping packet loss ratio", result.PacketLoss)
		if e.redisCounter != nil {
			pingSentLabels := map[string]string{"task_id": baseLabels["task_id"], "task_name": baseLabels["task_name"], "task_type": baseLabels["task_type"], "business_group": baseLabels["business_group"], "schedule_mode": baseLabels["schedule_mode"]}
			// note: pingSentLabels used only for Redis key; gauge uses allLabels (no task_id)
			for i := 0; i < result.PingPacketsSent; i++ {
				e.redisCounter.Inc(ctx, "srehub_inspect_ping_packet_send_total", pingSentLabels)
			}
			for i := 0; i < result.PingPacketsRecv; i++ {
				e.redisCounter.Inc(ctx, "srehub_inspect_ping_packet_recv_total", pingSentLabels)
			}
			sentVal := e.redisCounter.Get(ctx, "srehub_inspect_ping_packet_send_total", pingSentLabels)
			recvVal := e.redisCounter.Get(ctx, "srehub_inspect_ping_packet_recv_total", pingSentLabels)
			addGauge("srehub_inspect_ping_packet_send_total", "Ping total packets sent (cumulative)", sentVal)
			addGauge("srehub_inspect_ping_packet_recv_total", "Ping total packets received (cumulative)", recvVal)
		}
	case "tcp":
		addGauge("srehub_inspect_tcp_connect_duration_seconds", "TCP connect duration seconds", result.TCPConnectTime/1000.0)
		portReachable := 0.0
		if result.Success {
			portReachable = 1.0
		}
		addGauge("srehub_inspect_tcp_port_reachable", "TCP port reachable (1=yes 0=no)", portReachable)
		if e.redisCounter != nil {
			if result.Success {
				e.redisCounter.Inc(ctx, "srehub_inspect_tcp_connect_success_total", baseLabels)
			} else {
				e.redisCounter.Inc(ctx, "srehub_inspect_tcp_connect_fail_total", baseLabels)
			}
			sVal := e.redisCounter.Get(ctx, "srehub_inspect_tcp_connect_success_total", baseLabels)
			fVal := e.redisCounter.Get(ctx, "srehub_inspect_tcp_connect_fail_total", baseLabels)
			addGauge("srehub_inspect_tcp_connect_success_total", "TCP connect success count (cumulative)", sVal)
			addGauge("srehub_inspect_tcp_connect_fail_total", "TCP connect fail count (cumulative)", fVal)
		}
	case "udp":
		addGauge("srehub_inspect_udp_transfer_delay_seconds", "UDP transfer delay seconds", (result.UDPWriteTime+result.UDPReadTime)/1000.0)
		if e.redisCounter != nil {
			e.redisCounter.Inc(ctx, "srehub_inspect_udp_send_total", baseLabels)
			if result.Success {
				e.redisCounter.Inc(ctx, "srehub_inspect_udp_recv_total", baseLabels)
			} else {
				e.redisCounter.Inc(ctx, "srehub_inspect_udp_loss_total", baseLabels)
			}
			sVal := e.redisCounter.Get(ctx, "srehub_inspect_udp_send_total", baseLabels)
			rVal := e.redisCounter.Get(ctx, "srehub_inspect_udp_recv_total", baseLabels)
			lVal := e.redisCounter.Get(ctx, "srehub_inspect_udp_loss_total", baseLabels)
			addGauge("srehub_inspect_udp_send_total", "UDP packets sent (cumulative)", sVal)
			addGauge("srehub_inspect_udp_recv_total", "UDP packets received (cumulative)", rVal)
			addGauge("srehub_inspect_udp_loss_total", "UDP packets lost (cumulative)", lVal)
		}
	}

	// 总成功/失败可用性
	addGauge("srehub_inspect_task_availability_gauge", "Probe success (1=ok 0=fail) for this execution", successVal)

	hostname, _ := os.Hostname()
	grouping := map[string]string{
		"instance":  hostname,
		"task_id":   fmt.Sprintf("%d", task.ID),
		"config_id": fmt.Sprintf("%d", config.ID),
	}

	if err := pusher.PushSamples("srehub", grouping, samples); err != nil {
		appLogger.Error("push metrics failed",
			zap.Uint("taskID", task.ID),
			zap.Error(err),
		)
	}
}

func (e *NetworkProbeExecutor) pushAppMetrics(ctx context.Context, task *ProbeTask, config *ProbeConfig, result *probers.AppResult, retryAttempt int) {
	pgw, err := e.pgwRepo.GetByID(ctx, task.PushgatewayID)
	if err != nil {
		appLogger.Error("pushAppMetrics: get pushgateway config failed",
			zap.Uint("pgwID", task.PushgatewayID),
			zap.Error(err))
		return
	}
	if pgw.Status != 1 {
		appLogger.Warn("pushAppMetrics: pushgateway disabled",
			zap.Uint("pgwID", task.PushgatewayID),
			zap.String("pgwURL", pgw.URL))
		return
	}

	pusher := metrics.NewPusher(pgw.URL, pgw.Username, pgw.Password)

	groupName := ""
	if e.groupLookup != nil && task.GroupID > 0 {
		groupName = e.groupLookup(ctx, task.GroupID)
	}
	scheduleMode := task.TriggerType
	if scheduleMode == "" {
		scheduleMode = "scheduled"
	}

	// 类型扩展标签（HTTP/HTTPS/WebSocket）
	httpMethod := ""
	httpPath := "/"
	statusCodeStr := strconv.Itoa(result.HTTPStatusCode)
	if config.Method != "" {
		httpMethod = config.Method
	}

	baseLabels := map[string]string{
		"task_id":        fmt.Sprintf("%d", task.ID),
		"task_name":      task.Name,
		"task_type":      config.Type,
		"business_group": groupName,
		"schedule_mode":  scheduleMode,
	}

	// metric label 不含 task_id（已在 grouping 中，避免冲突）
	allLabels := prometheus.Labels{
		"task_name":      task.Name,
		"task_type":      config.Type,
		"business_group": groupName,
		"schedule_mode":  scheduleMode,
		"target":         config.Target,
		"probe_name":     config.Name,
		"http_method":    httpMethod,
		"http_path":      httpPath,
		"status_code":    statusCodeStr,
	}
	if config.Tags != "" {
		for _, tag := range strings.Split(config.Tags, ",") {
			parts := strings.SplitN(strings.TrimSpace(tag), "=", 2)
			if len(parts) == 2 {
				allLabels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	labelNames := make([]string, 0, len(allLabels))
	labelValues := make([]string, 0, len(allLabels))
	for k, v := range allLabels {
		labelNames = append(labelNames, k)
		labelValues = append(labelValues, v)
	}

	samples := make([]metrics.MetricSample, 0, 32)
	addGauge := func(name, help string, value float64) {
		samples = append(samples, metrics.MetricSample{
			Name: name, Help: help,
			LabelNames: labelNames, LabelValues: labelValues,
			Value: value,
		})
	}
	addCounter := func(name, help string) {
		if e.redisCounter == nil {
			return
		}
		val := e.redisCounter.Inc(ctx, name, baseLabels)
		samples = append(samples, metrics.MetricSample{
			Name: name, Help: help,
			LabelNames: labelNames, LabelValues: labelValues,
			Value: val,
		})
	}

	// 通用 Counter
	addCounter("srehub_inspect_task_exec_total", "Total task executions")
	if result.Success {
		addCounter("srehub_inspect_task_success_total", "Total task successes")
	} else {
		addCounter("srehub_inspect_task_fail_total", "Total task failures")
	}
	if retryAttempt > 0 {
		for i := 0; i < retryAttempt; i++ {
			if e.redisCounter != nil {
				e.redisCounter.Inc(ctx, "srehub_inspect_task_retry_total", baseLabels)
			}
		}
		if e.redisCounter != nil {
			retryVal := e.redisCounter.Get(ctx, "srehub_inspect_task_retry_total", baseLabels)
			samples = append(samples, metrics.MetricSample{
				Name: "srehub_inspect_task_retry_total", Help: "Total task retries",
				LabelNames: labelNames, LabelValues: labelValues,
				Value: retryVal,
			})
		}
	}

	// 通用 Gauge
	successVal := 0.0
	if result.Success {
		successVal = 1.0
	}
	addGauge("srehub_inspect_task_availability_gauge", "Probe success for this execution (1=ok 0=fail)", successVal)
	addGauge("srehub_inspect_task_exec_duration_seconds", "Task execution duration seconds", result.Latency/1000.0)
	if e.redisCounter != nil {
		total := e.redisCounter.Get(ctx, "srehub_inspect_task_exec_total", baseLabels)
		success := e.redisCounter.Get(ctx, "srehub_inspect_task_success_total", baseLabels)
		if total > 0 {
			addGauge("srehub_inspect_task_availability", "Task availability ratio (success/total)", success/total)
		}
	}

	// HTTP/HTTPS 专属指标
	addGauge("srehub_inspect_http_response_duration_seconds", "HTTP response duration seconds", result.HTTPResponseTime/1000.0)
	addGauge("srehub_inspect_http_dns_duration_seconds", "HTTP DNS lookup duration seconds", result.DNSLookupTime/1000.0)
	addGauge("srehub_inspect_http_tls_duration_seconds", "HTTP TLS handshake duration seconds", result.TLSHandshakeTime/1000.0)
	addGauge("srehub_inspect_http_first_byte_seconds", "HTTP TTFB seconds", result.TTFB/1000.0)
	assertVal := 0.0
	if result.AssertionSuccess {
		assertVal = 1.0
	}
	addGauge("srehub_inspect_http_assertion_result", "HTTP assertion result (1=pass 0=fail)", assertVal)

	// HTTPS 证书剩余天数
	if result.SSLCertNotAfter > 0 {
		validDays := float64(result.SSLCertNotAfter-time.Now().Unix()) / 86400.0
		if validDays < 0 {
			validDays = 0
		}
		addGauge("srehub_inspect_https_cert_valid_days", "HTTPS certificate remaining valid days", validDays)
	}

	// WebSocket 专属指标（当类型为 websocket 时附加）
	if config.Type == "websocket" {
		connEstablished := 0.0
		if result.Success {
			connEstablished = 1.0
		}
		addGauge("srehub_inspect_ws_connection_established", "WebSocket connection established (1=yes 0=no)", connEstablished)
		// 握手耗时用 HTTP response time 近似
		addGauge("srehub_inspect_ws_handshake_duration_seconds", "WebSocket handshake duration seconds", result.HTTPResponseTime/1000.0)
		if e.redisCounter != nil {
			if result.Success {
				e.redisCounter.Inc(ctx, "srehub_inspect_ws_handshake_success_total", baseLabels)
			} else {
				e.redisCounter.Inc(ctx, "srehub_inspect_ws_disconnect_total", baseLabels)
			}
			handshakeVal := e.redisCounter.Get(ctx, "srehub_inspect_ws_handshake_success_total", baseLabels)
			disconnectVal := e.redisCounter.Get(ctx, "srehub_inspect_ws_disconnect_total", baseLabels)
			addGauge("srehub_inspect_ws_handshake_success_total", "WebSocket handshake success count (cumulative)", handshakeVal)
			addGauge("srehub_inspect_ws_disconnect_total", "WebSocket disconnect count (cumulative)", disconnectVal)
		}
	}

	// HTTP Counter：状态码分布
	if e.redisCounter != nil && result.HTTPStatusCode > 0 {
		statusLabels := map[string]string{
			"task_id": baseLabels["task_id"], "task_name": baseLabels["task_name"],
			"task_type": baseLabels["task_type"], "business_group": baseLabels["business_group"],
			"schedule_mode": baseLabels["schedule_mode"], "status_code": statusCodeStr,
		}
		statusVal := e.redisCounter.Inc(ctx, "srehub_inspect_http_status_code_total", statusLabels)
		statusLabelNames := []string{"task_name", "task_type", "business_group", "schedule_mode", "status_code"}
		statusLabelValues := []string{statusLabels["task_name"], statusLabels["task_type"], statusLabels["business_group"], statusLabels["schedule_mode"], statusCodeStr}
		samples = append(samples, metrics.MetricSample{
			Name: "srehub_inspect_http_status_code_total", Help: "HTTP status code distribution (cumulative)",
			LabelNames: statusLabelNames, LabelValues: statusLabelValues,
			Value: statusVal,
		})
	}

	hostname, _ := os.Hostname()
	grouping := map[string]string{
		"instance":  hostname,
		"task_id":   fmt.Sprintf("%d", task.ID),
		"config_id": fmt.Sprintf("%d", config.ID),
	}

	if err := pusher.PushSamples("srehub", grouping, samples); err != nil {
		appLogger.Error("push app metrics failed",
			zap.Uint("taskID", task.ID),
			zap.Error(err),
		)
	}
}

func (e *NetworkProbeExecutor) pushWorkflowMetrics(ctx context.Context, task *ProbeTask, config *ProbeConfig, result *WorkflowResult) {
	pgw, err := e.pgwRepo.GetByID(ctx, task.PushgatewayID)
	if err != nil {
		appLogger.Error("pushWorkflowMetrics: get pushgateway config failed",
			zap.Uint("pgwID", task.PushgatewayID),
			zap.Error(err))
		return
	}
	if pgw.Status != 1 {
		appLogger.Warn("pushWorkflowMetrics: pushgateway disabled",
			zap.Uint("pgwID", task.PushgatewayID),
			zap.String("pgwURL", pgw.URL))
		return
	}

	pusher := metrics.NewPusher(pgw.URL, pgw.Username, pgw.Password)

	groupName := ""
	if e.groupLookup != nil && task.GroupID > 0 {
		groupName = e.groupLookup(ctx, task.GroupID)
	}
	scheduleMode := task.TriggerType
	if scheduleMode == "" {
		scheduleMode = "scheduled"
	}

	baseLabels := map[string]string{
		"task_id":        fmt.Sprintf("%d", task.ID),
		"task_name":      task.Name,
		"task_type":      "probe_flow",
		"business_group": groupName,
		"schedule_mode":  scheduleMode,
	}

	// metric label 不含 task_id（已在 grouping 中，避免冲突）
	allLabels := prometheus.Labels{
		"task_name":      task.Name,
		"task_type":      "probe_flow",
		"business_group": groupName,
		"schedule_mode":  scheduleMode,
		"probe_name":     config.Name,
		"flow_id":        fmt.Sprintf("%d", config.ID),
	}

	labelNames := make([]string, 0, len(allLabels))
	labelValues := make([]string, 0, len(allLabels))
	for k, v := range allLabels {
		labelNames = append(labelNames, k)
		labelValues = append(labelValues, v)
	}

	samples := make([]metrics.MetricSample, 0, 32)
	addGauge := func(name, help string, value float64) {
		samples = append(samples, metrics.MetricSample{
			Name: name, Help: help,
			LabelNames: labelNames, LabelValues: labelValues,
			Value: value,
		})
	}
	addCounter := func(name, help string) {
		if e.redisCounter == nil {
			return
		}
		val := e.redisCounter.Inc(ctx, name, baseLabels)
		samples = append(samples, metrics.MetricSample{
			Name: name, Help: help,
			LabelNames: labelNames, LabelValues: labelValues,
			Value: val,
		})
	}

	// 通用 Counter
	addCounter("srehub_inspect_task_exec_total", "Total task executions")
	if result.Success {
		addCounter("srehub_inspect_task_success_total", "Total task successes")
	} else {
		addCounter("srehub_inspect_task_fail_total", "Total task failures")
	}

	// 通用 Gauge
	successVal := 0.0
	if result.Success {
		successVal = 1.0
	}
	addGauge("srehub_inspect_task_availability_gauge", "Probe success for this execution (1=ok 0=fail)", successVal)
	addGauge("srehub_inspect_task_exec_duration_seconds", "Task execution duration seconds", result.TotalLatency/1000.0)
	if e.redisCounter != nil {
		total := e.redisCounter.Get(ctx, "srehub_inspect_task_exec_total", baseLabels)
		success := e.redisCounter.Get(ctx, "srehub_inspect_task_success_total", baseLabels)
		if total > 0 {
			addGauge("srehub_inspect_task_availability", "Task availability ratio (success/total)", success/total)
		}
	}

	// 业务编排步骤级指标
	for i, sr := range result.StepResults {
		stepLabels := map[string]string{
			"task_id": baseLabels["task_id"], "task_name": baseLabels["task_name"],
			"task_type": baseLabels["task_type"], "business_group": baseLabels["business_group"],
			"schedule_mode": baseLabels["schedule_mode"],
			"flow_id": fmt.Sprintf("%d", config.ID),
			"step_id": fmt.Sprintf("%d", i),
			"step_name": sr.StepName,
		}
		stepLabelNames := []string{"task_name", "task_type", "business_group", "schedule_mode", "flow_id", "step_id", "step_name"}
		stepLabelValues := []string{
			stepLabels["task_name"], stepLabels["task_type"],
			stepLabels["business_group"], stepLabels["schedule_mode"],
			stepLabels["flow_id"], stepLabels["step_id"], stepLabels["step_name"],
		}
		addStepGauge := func(name, help string, val float64) {
			samples = append(samples, metrics.MetricSample{
				Name: name, Help: help,
				LabelNames: stepLabelNames, LabelValues: stepLabelValues,
				Value: val,
			})
		}

		if e.redisCounter != nil {
			e.redisCounter.Inc(ctx, "srehub_inspect_flow_step_exec_total", stepLabels)
			if !sr.Success && !sr.Skipped {
				e.redisCounter.Inc(ctx, "srehub_inspect_flow_step_fail_total", stepLabels)
			}
			execVal := e.redisCounter.Get(ctx, "srehub_inspect_flow_step_exec_total", stepLabels)
			failVal := e.redisCounter.Get(ctx, "srehub_inspect_flow_step_fail_total", stepLabels)
			addStepGauge("srehub_inspect_flow_step_exec_total", "Flow step execution count (cumulative)", execVal)
			addStepGauge("srehub_inspect_flow_step_fail_total", "Flow step failure count (cumulative)", failVal)
		}

		stepStatus := 0.0
		if sr.Success {
			stepStatus = 1.0
		} else if sr.Skipped {
			stepStatus = 2.0
		}
		addStepGauge("srehub_inspect_flow_step_status", "Flow step status (1=success 0=fail 2=skipped)", stepStatus)
		addStepGauge("srehub_inspect_flow_step_exec_duration", "Flow step execution duration seconds", sr.Latency/1000.0)

		assertResult := -1.0
		for _, ar := range sr.AssertionResults {
			if ar.Success {
				assertResult = 1.0
			} else {
				assertResult = 0.0
				break
			}
		}
		addStepGauge("srehub_inspect_flow_step_assert_result", "Flow step assertion result (1=pass 0=fail -1=no_assertion)", assertResult)
	}

	hostname, _ := os.Hostname()
	grouping := map[string]string{
		"instance":  hostname,
		"task_id":   fmt.Sprintf("%d", task.ID),
		"config_id": fmt.Sprintf("%d", config.ID),
	}

	if err := pusher.PushSamples("srehub", grouping, samples); err != nil {
		appLogger.Error("push workflow metrics failed",
			zap.Uint("taskID", task.ID),
			zap.Error(err),
		)
	}
}

// BuildProbeRequest 构建 protobuf ProbeRequest（导出供 service 层复用）
func BuildProbeRequest(cfg *ProbeConfig, appCfg *probers.AppProbeConfig) *pb.ProbeRequest {
	req := &pb.ProbeRequest{
		RequestId:  generateRequestID(),
		ProbeType:  cfg.Type,
		Target:     cfg.Target,
		Port:       int32(cfg.Port),
		Url:        appCfg.URL,
		Method:     appCfg.Method,
		Body:       appCfg.Body,
		Timeout:    int32(appCfg.Timeout),
		SkipVerify: appCfg.SkipVerify,
		ProxyUrl:   appCfg.ProxyURL,
	}

	// Content-Type
	if appCfg.ContentType != "" {
		req.ContentType = appCfg.ContentType
	}

	// Headers
	if len(appCfg.Headers) > 0 {
		req.Headers = appCfg.Headers
	}

	// Params
	if len(appCfg.Params) > 0 {
		req.Params = appCfg.Params
	}

	// Assertions
	if len(appCfg.Assertions) > 0 {
		req.Assertions = make([]*pb.ProbeAssertion, 0, len(appCfg.Assertions))
		for _, a := range appCfg.Assertions {
			req.Assertions = append(req.Assertions, &pb.ProbeAssertion{
				Name:      a.Name,
				Source:    a.Source,
				Path:      a.Path,
				Condition: a.Condition,
				Value:     a.Value,
			})
		}
	}

	// WebSocket specific
	if cfg.Type == "websocket" {
		req.WsMessage = appCfg.WSMessage
		req.WsMessageType = int32(appCfg.WSMessageType)
		req.WsReadTimeout = int32(appCfg.WSReadTimeout)
	}

	return req
}

// ConvertProbeResultToAppResult 转换 pb.ProbeResult 到 probers.AppResult（导出供 service 层复用）
func ConvertProbeResultToAppResult(pbResult *pb.ProbeResult) *probers.AppResult {
	result := &probers.AppResult{
		Success:           pbResult.Success,
		Error:             pbResult.Error,
		Latency:           pbResult.Latency,
		HTTPStatusCode:    int(pbResult.HttpStatusCode),
		HTTPResponseTime:  pbResult.HttpResponseTime,
		HTTPContentLength: pbResult.HttpContentLength,
		ResponseBody:      string(pbResult.ResponseBody), // 转换 []byte 为 string
		ResponseHeaders:   pbResult.ResponseHeaders,

		// Performance breakdown
		DNSLookupTime:       pbResult.DnsLookupTime,
		TCPConnectTime:      pbResult.TcpConnectTimeHttp,
		TLSHandshakeTime:    pbResult.TlsHandshakeTime,
		TTFB:                pbResult.Ttfb,
		ContentTransferTime: pbResult.ContentTransferTime,

		// TLS info
		TLSVersion:      pbResult.TlsVersion,
		TLSCipherSuite:  pbResult.TlsCipherSuite,
		SSLCertNotAfter: pbResult.SslCertNotAfter,

		// HTTP details
		RedirectCount:       int(pbResult.RedirectCount),
		RedirectTime:        pbResult.RedirectTime,
		FinalURL:            pbResult.FinalUrl,
		ResponseHeaderBytes: int(pbResult.ResponseHeaderBytes),
		ResponseBodyBytes:   int(pbResult.ResponseBodyBytes),

		// Assertions
		AssertionSuccess:   pbResult.AssertionSuccess,
		AssertionPassCount: int(pbResult.AssertionPassCount),
		AssertionFailCount: int(pbResult.AssertionFailCount),
		AssertionEvalTime:  pbResult.AssertionEvalTime,
	}

	// Convert assertion results
	if len(pbResult.AssertionResults) > 0 {
		result.AssertionResults = make([]probers.AssertionResult, 0, len(pbResult.AssertionResults))
		for _, ar := range pbResult.AssertionResults {
			result.AssertionResults = append(result.AssertionResults, probers.AssertionResult{
				Name:    ar.Name,
				Success: ar.Success,
				Actual:  ar.Actual,
				Error:   ar.Error,
			})
		}
	}

	return result
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	return fmt.Sprintf("probe_%d_%d", time.Now().UnixNano(), rand.Int63())
}


// NetworkProbeV2Executor implements scheduler.TaskExecutor for probe tasks from inspection_tasks table.
type NetworkProbeV2Executor struct {
	inspectionTaskRepo InspectionTaskRepo
	configRepo         ProbeConfigRepo
	resultRepo         ProbeResultRepo
	pgwRepo            PushgatewayConfigRepo
	groupLookup        func(ctx context.Context, id uint) string
	agentFactory       AgentCommandFactory
	variableResolver   *VariableResolver
	redisCounter       *metrics.RedisCounter
	teleAIEnabled      bool
}

// InspectionTaskRepo interface for accessing inspection_tasks table
type InspectionTaskRepo interface {
	GetByID(ctx context.Context, id uint) (*InspectionTaskV2, error)
	UpdateLastRun(ctx context.Context, id uint, status string) error
}

// InspectionTaskV2 represents a task from inspection_tasks table
type InspectionTaskV2 struct {
	ID              uint
	Name            string
	TaskType        string
	CronExpr        string
	Enabled         bool
	GroupIDs        string
	ItemIDs         string
	PushgatewayID   uint
	Concurrency     int
	// 需求一新增：任务调度级别覆盖配置
	ExecutionMode   string // 执行方式覆盖（拨测：local/agent；空=不覆盖）
	AgentHostIDs    string // Agent 主机 ID 列表（JSON 数组）
	BusinessGroupID uint   // 业务分组 ID 覆盖
	CustomVariables string // 自定义变量 JSON 对象（优先级最高）
}

// NewNetworkProbeV2Executor creates a new executor for probe tasks from inspection_tasks table.
func NewNetworkProbeV2Executor(
	inspectionTaskRepo InspectionTaskRepo,
	configRepo ProbeConfigRepo,
	resultRepo ProbeResultRepo,
	pgwRepo PushgatewayConfigRepo,
	groupLookup func(ctx context.Context, id uint) string,
) *NetworkProbeV2Executor {
	return &NetworkProbeV2Executor{
		inspectionTaskRepo: inspectionTaskRepo,
		configRepo:         configRepo,
		resultRepo:         resultRepo,
		pgwRepo:            pgwRepo,
		groupLookup:        groupLookup,
	}
}

// SetAgentCommandFactory injects Agent capability.
func (e *NetworkProbeV2Executor) SetAgentCommandFactory(f AgentCommandFactory) {
	e.agentFactory = f
}

// SetVariableResolver injects variable resolver capability.
func (e *NetworkProbeV2Executor) SetVariableResolver(r *VariableResolver) {
	e.variableResolver = r
}

// SetRedisCounter injects a Redis-backed counter for persistent metric counting.
func (e *NetworkProbeV2Executor) SetRedisCounter(rc *metrics.RedisCounter) {
	e.redisCounter = rc
}

// SetTeleAIEnabled sets the global TeleAI Authorization auto-fill switch.
func (e *NetworkProbeV2Executor) SetTeleAIEnabled(enabled bool) {
	e.teleAIEnabled = enabled
}

func (e *NetworkProbeV2Executor) Type() string { return "network_probe_v2" }

// Execute runs a probe task from inspection_tasks table.
func (e *NetworkProbeV2Executor) Execute(ctx context.Context, task scheduler.Task) error {
	var payload struct {
		TaskID      uint   `json:"task_id"`
		TriggerType string `json:"trigger_type"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}
	triggerType := payload.TriggerType
	if triggerType == "" {
		triggerType = "scheduled"
	}

	inspectionTask, err := e.inspectionTaskRepo.GetByID(ctx, payload.TaskID)
	if err != nil {
		return fmt.Errorf("get inspection task: %w", err)
	}

	// Parse item_ids (probe config IDs) from JSON array
	configIDs := parseJSONArray(inspectionTask.ItemIDs)
	if len(configIDs) == 0 {
		return fmt.Errorf("no probe config IDs in task %d", payload.TaskID)
	}

	// Load probe configs
	var configs []*ProbeConfig
	for _, configID := range configIDs {
		cfg, err := e.configRepo.GetByID(ctx, configID)
		if err != nil {
			appLogger.Error("get probe config failed", zap.Uint("config_id", configID), zap.Error(err))
			continue
		}
		configs = append(configs, cfg)
	}

	if len(configs) == 0 {
		return fmt.Errorf("no valid probe configs for task %d", payload.TaskID)
	}

	concurrency := inspectionTask.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	// Create a temporary ProbeTask for compatibility with existing execution logic
	probeTask := &ProbeTask{
		ID:            payload.TaskID,
		Name:          inspectionTask.Name,
		PushgatewayID: inspectionTask.PushgatewayID,
		Concurrency:   concurrency,
		TriggerType:   triggerType,
	}

	// 需求一：解析任务级自定义变量（优先级最高）
	taskCustomVars := make(map[string]string)
	if inspectionTask.CustomVariables != "" {
		_ = json.Unmarshal([]byte(inspectionTask.CustomVariables), &taskCustomVars)
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var failCount int64

	for _, config := range configs {
		wg.Add(1)
		sem <- struct{}{}
		go func(cfg *ProbeConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			// 需求一：任务级执行方式覆盖（优先于各探测配置自身设置）
			effectiveCfg := *cfg // 浅拷贝，避免修改原配置
			if inspectionTask.ExecutionMode != "" {
				effectiveCfg.ExecMode = inspectionTask.ExecutionMode
				if inspectionTask.ExecutionMode == ExecModeAgent && inspectionTask.AgentHostIDs != "" {
					effectiveCfg.AgentHostIDs = inspectionTask.AgentHostIDs
				}
			}
			// 需求一：业务分组覆盖（影响变量作用域）
			if inspectionTask.BusinessGroupID > 0 {
				effectiveCfg.GroupID = inspectionTask.BusinessGroupID
			}
			cfgPtr := &effectiveCfg

			// 变量解析：优先级 任务变量 > 系统变量(ProbeVariable) > 探测配置内联变量
			resolvedCfg := cfgPtr
			if e.variableResolver != nil {
				if rc, err := e.variableResolver.ResolveConfigWithExtra(ctx, cfgPtr, taskCustomVars); err != nil {
					appLogger.Error("resolve variables failed", zap.String("name", cfgPtr.Name), zap.Error(err))
				} else {
					resolvedCfg = rc
				}
			}

			// Execute probe using the same logic as NetworkProbeExecutor
			executor := &NetworkProbeExecutor{
				resultRepo:       e.resultRepo,
				pgwRepo:          e.pgwRepo,
				groupLookup:      e.groupLookup,
				agentFactory:     e.agentFactory,
				variableResolver: e.variableResolver,
				redisCounter:     e.redisCounter,
				teleAIEnabled:    e.teleAIEnabled,
			}

			if resolvedCfg.Category == ProbeCategoryApplication {
				executor.executeAndSaveAppProbe(ctx, probeTask, cfgPtr, resolvedCfg, &failCount)
			} else if resolvedCfg.Category == ProbeCategoryWorkflow {
				executor.executeAndSaveWorkflowProbe(ctx, probeTask, cfgPtr, resolvedCfg, &failCount)
			} else {
				executor.executeAndSaveNetworkProbe(ctx, probeTask, cfgPtr, resolvedCfg, &failCount)
			}
		}(config)
	}
	wg.Wait()

	// Auto-cleanup: keep latest 30 * len(configs) results per task
	keepCount := 30 * len(configs)
	if err := e.resultRepo.CleanupByTaskID(ctx, probeTask.ID, keepCount); err != nil {
		appLogger.Error("cleanup probe results failed", zap.Error(err))
	}

	runStatus := "success"
	if failCount > 0 {
		runStatus = "failed"
	}
	if err := e.inspectionTaskRepo.UpdateLastRun(ctx, payload.TaskID, runStatus); err != nil {
		appLogger.Error("update last run failed", zap.Error(err))
	}

	return nil
}

// ProbeSyncResult 拨测同步执行结果（需求二）
type ProbeSyncResult struct {
	TaskID       uint              `json:"task_id"`
	TaskName     string            `json:"task_name"`
	TaskType     string            `json:"task_type"`
	Status       string            `json:"status"`
	Duration     float64           `json:"duration"`
	TotalItems   int               `json:"total_items"`
	SuccessCount int               `json:"success_count"`
	FailedCount  int               `json:"failed_count"`
	Details      []ProbeItemDetail `json:"details"`
}

// ProbeItemDetail 单个拨测项执行明细
type ProbeItemDetail struct {
	ConfigID         uint    `json:"config_id"`
	ConfigName       string  `json:"config_name"`
	ConfigType       string  `json:"config_type"`
	Target           string  `json:"target"`
	Success          bool    `json:"success"`
	Latency          float64 `json:"latency_ms"`
	ErrorMessage     string  `json:"error_message,omitempty"`
	Output           string  `json:"output,omitempty"`          // 格式化的执行输出
	AssertionSuccess bool    `json:"assertion_success"`
	AssertionDetail  string  `json:"assertion_detail,omitempty"` // JSON
	AgentHostID      uint    `json:"agent_host_id,omitempty"`
	RetryAttempt     int     `json:"retry_attempt"`
	ExecutedAt       string  `json:"executed_at"`
}

// ExecuteSync 同步执行拨测任务，阻塞直到完成，返回完整结果（需求二）
func (e *NetworkProbeV2Executor) ExecuteSync(ctx context.Context, taskID uint) (*ProbeSyncResult, error) {
	inspectionTask, err := e.inspectionTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get inspection task: %w", err)
	}

	configIDs := parseJSONArray(inspectionTask.ItemIDs)
	if len(configIDs) == 0 {
		return nil, fmt.Errorf("no probe config IDs in task %d", taskID)
	}

	var configs []*ProbeConfig
	for _, configID := range configIDs {
		cfg, err := e.configRepo.GetByID(ctx, configID)
		if err != nil {
			appLogger.Error("get probe config failed", zap.Uint("config_id", configID), zap.Error(err))
			continue
		}
		configs = append(configs, cfg)
	}
	if len(configs) == 0 {
		return nil, fmt.Errorf("no valid probe configs for task %d", taskID)
	}

	concurrency := inspectionTask.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	// 解析任务级自定义变量
	taskCustomVars := make(map[string]string)
	if inspectionTask.CustomVariables != "" {
		_ = json.Unmarshal([]byte(inspectionTask.CustomVariables), &taskCustomVars)
	}

	startTime := time.Now()
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var details []ProbeItemDetail
	var failCount int64

	for _, config := range configs {
		wg.Add(1)
		sem <- struct{}{}
		go func(cfg *ProbeConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			effectiveCfg := *cfg
			if inspectionTask.ExecutionMode != "" {
				effectiveCfg.ExecMode = inspectionTask.ExecutionMode
				if inspectionTask.ExecutionMode == ExecModeAgent && inspectionTask.AgentHostIDs != "" {
					effectiveCfg.AgentHostIDs = inspectionTask.AgentHostIDs
				}
			}
			if inspectionTask.BusinessGroupID > 0 {
				effectiveCfg.GroupID = inspectionTask.BusinessGroupID
			}
			cfgPtr := &effectiveCfg

			resolvedCfg := cfgPtr
			if e.variableResolver != nil {
				if rc, err := e.variableResolver.ResolveConfigWithExtra(ctx, cfgPtr, taskCustomVars); err == nil {
					resolvedCfg = rc
				}
			}

			execAt := time.Now().Format(time.RFC3339)
			detail := ProbeItemDetail{
				ConfigID:   cfg.ID,
				ConfigName: cfg.Name,
				ConfigType: cfg.Type,
				Target:     cfg.Target,
				ExecutedAt: execAt,
			}

			// 复用现有执行逻辑
			helperExec := &NetworkProbeExecutor{
				agentFactory:     e.agentFactory,
				variableResolver: e.variableResolver,
				teleAIEnabled:    e.teleAIEnabled,
			}

			switch resolvedCfg.Category {
			case ProbeCategoryApplication:
				appResult, agentHostID := helperExec.executeAppProbe(resolvedCfg)
				detail.Success = appResult.Success
				detail.Latency = appResult.Latency
				detail.ErrorMessage = appResult.Error
				detail.AgentHostID = agentHostID
				detail.AssertionSuccess = appResult.AssertionSuccess
				if len(appResult.AssertionResults) > 0 {
					if b, _ := json.Marshal(appResult.AssertionResults); b != nil {
						detail.AssertionDetail = string(b)
					}
				}
			case ProbeCategoryWorkflow:
				wfResult := ExecuteWorkflowProbe(ctx, resolvedCfg, e.variableResolver, e.agentFactory, e.teleAIEnabled)
				detail.Success = wfResult.Success
				detail.Latency = wfResult.TotalLatency
				detail.ErrorMessage = wfResult.Error
				if b, _ := json.Marshal(wfResult.StepResults); b != nil {
					detail.Output = string(b)
				}
			default:
				netResult, agentHostID := helperExec.executeProbe(resolvedCfg)
				detail.Success = netResult.Success
				detail.Latency = netResult.Latency
				detail.ErrorMessage = netResult.Error
				detail.AgentHostID = agentHostID
			}

			if !detail.Success {
				atomic.AddInt64(&failCount, 1)
			}
			mu.Lock()
			details = append(details, detail)
			mu.Unlock()
		}(config)
	}
	wg.Wait()

	duration := time.Since(startTime).Seconds()
	status := "success"
	if failCount > 0 && int(failCount) == len(details) {
		status = "failed"
	} else if failCount > 0 {
		status = "partial"
	}

	return &ProbeSyncResult{
		TaskID:       taskID,
		TaskName:     inspectionTask.Name,
		TaskType:     "probe",
		Status:       status,
		Duration:     duration,
		TotalItems:   len(configs),
		SuccessCount: len(details) - int(failCount),
		FailedCount:  int(failCount),
		Details:      details,
	}, nil
}

// WsAction represents a single WebSocket action (send or receive) in a multi-step WS probe.
type WsAction struct {
	Type        string `json:"type"`        // "send" or "receive"
	Message     string `json:"message"`     // send: message content
	MessageType int    `json:"msgType"`     // send: 1=text, 2=binary
	ReadTimeout int    `json:"readTimeout"` // receive: timeout in seconds
	ReceiveMode string `json:"receiveMode"` // receive: "" or "single" = one message; "stream" = collect all until timeout
}

// WsActionResult holds the result of a single WS action executed by the Agent.
type WsActionResult struct {
	Success      bool    `json:"success"`
	ResponseBody string  `json:"body"`
	Latency      float64 `json:"latency"` // ms
	Error        string  `json:"error"`
}

// parseJSONArray parses a JSON array string like "[1,2,3]" into []uint
func parseJSONArray(jsonStr string) []uint {
	var ids []uint
	if err := json.Unmarshal([]byte(jsonStr), &ids); err != nil {
		return nil
	}
	return ids
}
