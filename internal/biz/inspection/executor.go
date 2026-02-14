package inspection

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ydcloud-dy/opshub/internal/biz/inspection/probers"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/metrics"
	"github.com/ydcloud-dy/opshub/pkg/scheduler"
	"go.uber.org/zap"
)

// NetworkProbeExecutor implements scheduler.TaskExecutor for network probes.
type NetworkProbeExecutor struct {
	taskRepo    ProbeTaskRepo
	resultRepo  ProbeResultRepo
	pgwRepo     PushgatewayConfigRepo
	groupLookup func(ctx context.Context, id uint) string // returns group name
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

			prober, err := probers.GetProber(cfg.Type)
			if err != nil {
				appLogger.Error("get prober failed", zap.String("type", cfg.Type), zap.Error(err))
				atomic.AddInt64(&failCount, 1)
				return
			}
			result := prober.Probe(cfg.Target, cfg.Port, cfg.Timeout, cfg.Count, cfg.PacketSize)

			dbResult := &ProbeResult{
				ProbeTaskID:     probeTask.ID,
				ProbeConfigID:   cfg.ID,
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
			}
			if err := e.resultRepo.Create(ctx, dbResult); err != nil {
				appLogger.Error("save probe result failed", zap.Error(err))
			}

			if !result.Success {
				atomic.AddInt64(&failCount, 1)
			}

			if probeTask.PushgatewayID > 0 {
				e.pushMetrics(ctx, probeTask, cfg, result)
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
	}

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
