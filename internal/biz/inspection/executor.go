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
		probeReq := buildProbeRequest(cfg, appCfg)

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
		appResult := convertProbeResultToAppResult(pbResult)

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

// parseHostIDs parses a comma-separated string of host IDs into a uint slice.
func parseHostIDs(s string) []uint {
	if s == "" {
		return nil
	}
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
	wfResult := ExecuteWorkflowProbe(ctx, resolvedCfg, e.variableResolver, e.agentFactory)
	detail := ""
	if b, err := json.Marshal(wfResult); err == nil {
		detail = string(b)
	}
	dbResult := &ProbeResult{
		ProbeTaskID:   probeTask.ID,
		ProbeConfigID: origCfg.ID,
		Success:       wfResult.Success,
		Latency:       wfResult.TotalLatency,
		ErrorMessage:  wfResult.Error,
		Detail:        detail,
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
func ExecuteWorkflowProbe(ctx context.Context, cfg *ProbeConfig, resolver *VariableResolver, agentFactory AgentCommandFactory) *WorkflowResult {
	var def WorkflowDefinition
	if err := json.Unmarshal([]byte(cfg.Body), &def); err != nil {
		return &WorkflowResult{Success: false, Error: fmt.Sprintf("parse workflow definition: %v", err)}
	}
	if len(def.Steps) == 0 {
		return &WorkflowResult{Success: false, Error: "workflow has no steps"}
	}

	allowedGroupIDs := parseGroupIDs(cfg.GroupIDs)
	// Initialize variable context with workflow-level variables
	varCtx := make(map[string]string)
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

	for i, step := range def.Steps {
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
				for _, ar := range assertionResults {
					stepResult.AssertionResults = append(stepResult.AssertionResults, WorkflowAssertionResult{
						Name: ar.Name, Success: ar.Success, Actual: ar.Actual, Error: ar.Error,
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
				for _, ar := range assertionResults {
					stepResult.AssertionResults = append(stepResult.AssertionResults, WorkflowAssertionResult{
						Name: ar.Name, Success: ar.Success, Actual: ar.Actual, Error: ar.Error,
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
					stepResult.ExtractedVars = extracted
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

			// Check step.ExecMode for agent execution
			var appResult *probers.AppResult
			if step.ExecMode == ExecModeAgent && agentFactory != nil {
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

				// Agent mode not yet implemented for application probing
				appLogger.Warn("Agent mode for workflow step not yet implemented",
					zap.String("workflow_name", cfg.Name),
					zap.String("step_name", step.Name),
					zap.Uints("host_ids", hostIDs),
				)
				stepResult.Error = fmt.Sprintf("agent mode for application probing not yet implemented (step: %s)", step.Name)
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

			for _, ar := range appResult.AssertionResults {
				stepResult.AssertionResults = append(stepResult.AssertionResults, WorkflowAssertionResult{
					Name: ar.Name, Success: ar.Success, Actual: ar.Actual, Error: ar.Error,
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
	if err != nil || pgw.Status != 1 {
		return
	}

	pusher := metrics.NewPusher(pgw.URL, pgw.Username, pgw.Password)

	// Build labels
	groupName := ""
	if e.groupLookup != nil && task.GroupID > 0 {
		groupName = e.groupLookup(ctx, task.GroupID)
	}

	labels := prometheus.Labels{
		"probe_name": config.Name,
		"probe_type": config.Type,
		"target":     config.Target,
		"port":       fmt.Sprintf("%d", config.Port),
		"group_name": groupName,
		"task_name":  task.Name,
		"exec_mode":  config.ExecMode,
	}

	// Parse custom tags (key=value,key=value)
	if config.Tags != "" {
		for _, tag := range strings.Split(config.Tags, ",") {
			parts := strings.SplitN(strings.TrimSpace(tag), "=", 2)
			if len(parts) == 2 {
				labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))
	for k, v := range labels {
		labelNames = append(labelNames, k)
		labelValues = append(labelValues, v)
	}

	collectors := make([]prometheus.Collector, 0)
	addGauge := func(name, help string, value float64) {
		g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, labelNames)
		g.WithLabelValues(labelValues...).Set(value)
		collectors = append(collectors, g)
	}

	// Common metrics
	successVal := 0.0
	if result.Success {
		successVal = 1.0
	}
	addGauge("opshub_probe_success", "Probe success (1=ok 0=fail)", successVal)
	addGauge("opshub_probe_duration_seconds", "Probe total duration in seconds", result.Latency/1000.0)

	// Type-specific metrics
	switch config.Type {
	case "ping":
		addGauge("opshub_probe_ping_rtt_seconds", "Ping avg RTT seconds", result.PingRttAvg/1000.0)
		addGauge("opshub_probe_ping_rtt_min_seconds", "Ping min RTT seconds", result.PingRttMin/1000.0)
		addGauge("opshub_probe_ping_rtt_max_seconds", "Ping max RTT seconds", result.PingRttMax/1000.0)
		addGauge("opshub_probe_ping_packet_loss_ratio", "Ping packet loss ratio", result.PacketLoss)
		addGauge("opshub_probe_ping_packets_sent", "Ping packets sent", float64(result.PingPacketsSent))
		addGauge("opshub_probe_ping_packets_received", "Ping packets received", float64(result.PingPacketsRecv))
		addGauge("opshub_probe_ping_stddev_seconds", "Ping RTT stddev seconds", result.PingStddev/1000.0)
	case "tcp":
		addGauge("opshub_probe_tcp_connect_seconds", "TCP connect time seconds", result.TCPConnectTime/1000.0)
	case "udp":
		addGauge("opshub_probe_udp_write_seconds", "UDP write time seconds", result.UDPWriteTime/1000.0)
		addGauge("opshub_probe_udp_read_seconds", "UDP read time seconds", result.UDPReadTime/1000.0)
	case "http", "https", "websocket":
		// These are handled by pushAppMetrics; should not reach here
	}

	// Retry attempt metric (for network probes)
	addGauge("opshub_probe_retry_attempt", "Retry attempts", float64(result.RetryAttempt))

	hostname, _ := os.Hostname()
	grouping := map[string]string{
		"instance":  hostname,
		"task_id":   fmt.Sprintf("%d", task.ID),
		"config_id": fmt.Sprintf("%d", config.ID),
	}

	if err := pusher.Push("opshub_probe", grouping, collectors...); err != nil {
		appLogger.Error("push metrics failed",
			zap.Uint("taskID", task.ID),
			zap.Error(err),
		)
	}
}

func (e *NetworkProbeExecutor) pushAppMetrics(ctx context.Context, task *ProbeTask, config *ProbeConfig, result *probers.AppResult, retryAttempt int) {
	pgw, err := e.pgwRepo.GetByID(ctx, task.PushgatewayID)
	if err != nil || pgw.Status != 1 {
		return
	}

	pusher := metrics.NewPusher(pgw.URL, pgw.Username, pgw.Password)

	groupName := ""
	if e.groupLookup != nil && task.GroupID > 0 {
		groupName = e.groupLookup(ctx, task.GroupID)
	}

	labels := prometheus.Labels{
		"probe_name": config.Name,
		"probe_type": config.Type,
		"target":     config.Target,
		"group_name": groupName,
		"task_name":  task.Name,
		"exec_mode":  config.ExecMode,
	}
	if config.Tags != "" {
		for _, tag := range strings.Split(config.Tags, ",") {
			parts := strings.SplitN(strings.TrimSpace(tag), "=", 2)
			if len(parts) == 2 {
				labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))
	for k, v := range labels {
		labelNames = append(labelNames, k)
		labelValues = append(labelValues, v)
	}

	collectors := make([]prometheus.Collector, 0)
	addGauge := func(name, help string, value float64) {
		g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, labelNames)
		g.WithLabelValues(labelValues...).Set(value)
		collectors = append(collectors, g)
	}

	successVal := 0.0
	if result.Success {
		successVal = 1.0
	}
	addGauge("opshub_probe_success", "Probe success (1=ok 0=fail)", successVal)
	addGauge("opshub_probe_duration_seconds", "Probe total duration in seconds", result.Latency/1000.0)
	addGauge("opshub_probe_http_response_seconds", "HTTP real response time", result.HTTPResponseTime/1000.0)
	addGauge("opshub_probe_http_status_code", "HTTP status code", float64(result.HTTPStatusCode))
	addGauge("opshub_probe_http_content_length", "HTTP content length", float64(result.HTTPContentLength))

	assertVal := 0.0
	if result.AssertionSuccess {
		assertVal = 1.0
	}
	addGauge("opshub_probe_assertion_success", "Assertion success", assertVal)

	// Performance breakdown metrics
	if result.DNSLookupTime > 0 {
		addGauge("opshub_probe_dns_lookup_seconds", "DNS lookup time", result.DNSLookupTime/1000.0)
	}
	if result.TCPConnectTime > 0 {
		addGauge("opshub_probe_tcp_connect_seconds", "TCP connect time", result.TCPConnectTime/1000.0)
	}
	if result.TLSHandshakeTime > 0 {
		addGauge("opshub_probe_tls_handshake_seconds", "TLS handshake time", result.TLSHandshakeTime/1000.0)
	}
	if result.TTFB > 0 {
		addGauge("opshub_probe_ttfb_seconds", "Time to first byte", result.TTFB/1000.0)
	}
	if result.ContentTransferTime > 0 {
		addGauge("opshub_probe_content_transfer_seconds", "Content transfer time", result.ContentTransferTime/1000.0)
	}

	// Certificate monitoring
	if result.SSLCertNotAfter > 0 {
		addGauge("opshub_probe_ssl_cert_not_after_seconds", "SSL cert expiry timestamp", float64(result.SSLCertNotAfter))
	}

	// HTTP details
	if result.RedirectCount > 0 {
		addGauge("opshub_probe_http_redirect_count", "HTTP redirect count", float64(result.RedirectCount))
		addGauge("opshub_probe_http_redirect_time_seconds", "HTTP redirect time", result.RedirectTime/1000.0)
	}
	if result.ResponseHeaderBytes > 0 {
		addGauge("opshub_probe_http_response_header_bytes", "Response header size", float64(result.ResponseHeaderBytes))
	}
	if result.ResponseBodyBytes > 0 {
		addGauge("opshub_probe_http_response_body_bytes", "Response body size", float64(result.ResponseBodyBytes))
	}

	// Assertion statistics
	addGauge("opshub_probe_assertion_pass_count", "Passed assertions", float64(result.AssertionPassCount))
	addGauge("opshub_probe_assertion_fail_count", "Failed assertions", float64(result.AssertionFailCount))
	if result.AssertionEvalTime > 0 {
		addGauge("opshub_probe_assertion_eval_seconds", "Assertion evaluation time", result.AssertionEvalTime/1000.0)
	}

	// Retry attempt metric
	addGauge("opshub_probe_retry_attempt", "Retry attempts", float64(retryAttempt))

	hostname, _ := os.Hostname()
	grouping := map[string]string{
		"instance":  hostname,
		"task_id":   fmt.Sprintf("%d", task.ID),
		"config_id": fmt.Sprintf("%d", config.ID),
	}

	if err := pusher.Push("opshub_probe", grouping, collectors...); err != nil {
		appLogger.Error("push app metrics failed",
			zap.Uint("taskID", task.ID),
			zap.Error(err),
		)
	}
}

func (e *NetworkProbeExecutor) pushWorkflowMetrics(ctx context.Context, task *ProbeTask, config *ProbeConfig, result *WorkflowResult) {
	pgw, err := e.pgwRepo.GetByID(ctx, task.PushgatewayID)
	if err != nil || pgw.Status != 1 {
		return
	}

	pusher := metrics.NewPusher(pgw.URL, pgw.Username, pgw.Password)

	groupName := ""
	if e.groupLookup != nil && task.GroupID > 0 {
		groupName = e.groupLookup(ctx, task.GroupID)
	}

	labels := prometheus.Labels{
		"probe_name": config.Name,
		"probe_type": config.Type,
		"target":     config.Target,
		"group_name": groupName,
		"task_name":  task.Name,
		"exec_mode":  config.ExecMode,
	}
	if config.Tags != "" {
		for _, tag := range strings.Split(config.Tags, ",") {
			parts := strings.SplitN(strings.TrimSpace(tag), "=", 2)
			if len(parts) == 2 {
				labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))
	for k, v := range labels {
		labelNames = append(labelNames, k)
		labelValues = append(labelValues, v)
	}

	collectors := make([]prometheus.Collector, 0)
	addGauge := func(name, help string, value float64) {
		g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, labelNames)
		g.WithLabelValues(labelValues...).Set(value)
		collectors = append(collectors, g)
	}

	successVal := 0.0
	if result.Success {
		successVal = 1.0
	}
	addGauge("opshub_probe_success", "Probe success (1=ok 0=fail)", successVal)
	addGauge("opshub_probe_duration_seconds", "Probe total duration in seconds", result.TotalLatency/1000.0)

	totalSteps := len(result.StepResults)
	failedSteps := 0
	for _, sr := range result.StepResults {
		if !sr.Success && !sr.Skipped {
			failedSteps++
		}
	}
	addGauge("opshub_probe_workflow_step_count", "Workflow total step count", float64(totalSteps))
	addGauge("opshub_probe_workflow_failed_step_count", "Workflow failed step count", float64(failedSteps))

	hostname, _ := os.Hostname()
	grouping := map[string]string{
		"instance":  hostname,
		"task_id":   fmt.Sprintf("%d", task.ID),
		"config_id": fmt.Sprintf("%d", config.ID),
	}

	if err := pusher.Push("opshub_probe", grouping, collectors...); err != nil {
		appLogger.Error("push workflow metrics failed",
			zap.Uint("taskID", task.ID),
			zap.Error(err),
		)
	}
}

// buildProbeRequest 构建 protobuf ProbeRequest
func buildProbeRequest(cfg *ProbeConfig, appCfg *probers.AppProbeConfig) *pb.ProbeRequest {
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

// convertProbeResultToAppResult 转换 pb.ProbeResult 到 probers.AppResult
func convertProbeResultToAppResult(pbResult *pb.ProbeResult) *probers.AppResult {
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

