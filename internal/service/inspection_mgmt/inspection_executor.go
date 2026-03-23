package inspection_mgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/metrics"
	"github.com/ydcloud-dy/opshub/pkg/scheduler"
	"go.uber.org/zap"
)

// PushgatewayConfig 推送网关配置
type PushgatewayConfig struct {
	ID       uint
	Name     string
	URL      string
	Username string
	Password string
	Status   int
}

// PushgatewayRepo 推送网关仓库接口
type PushgatewayRepo interface {
	GetByID(ctx context.Context, id uint) (*PushgatewayConfig, error)
}

// InspectionExecutor 巡检任务执行器
type InspectionExecutor struct {
	taskRepo     inspectionmgmtdata.TaskRepository
	groupRepo    inspectionmgmtdata.GroupRepository
	itemRepo     inspectionmgmtdata.ItemRepository
	execRepo     inspectionmgmtdata.ExecutionRecordRepository
	pgwRepo      PushgatewayRepo
	itemSvc      *ItemService
	redisCounter *metrics.RedisCounter
}

// SetRedisCounter injects a Redis-backed counter for persistent metric counting.
func (e *InspectionExecutor) SetRedisCounter(rc *metrics.RedisCounter) {
	e.redisCounter = rc
}

// NewInspectionExecutor 创建巡检执行器
func NewInspectionExecutor(
	taskRepo inspectionmgmtdata.TaskRepository,
	groupRepo inspectionmgmtdata.GroupRepository,
	itemRepo inspectionmgmtdata.ItemRepository,
	execRepo inspectionmgmtdata.ExecutionRecordRepository,
	pgwRepo PushgatewayRepo,
	itemSvc *ItemService,
) *InspectionExecutor {
	return &InspectionExecutor{
		taskRepo:  taskRepo,
		groupRepo: groupRepo,
		itemRepo:  itemRepo,
		execRepo:  execRepo,
		pgwRepo:   pgwRepo,
		itemSvc:   itemSvc,
	}
}

func (e *InspectionExecutor) Type() string { return "inspection_task" }

// Execute 执行巡检任务
func (e *InspectionExecutor) Execute(ctx context.Context, task scheduler.Task) error {
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

	// 获取任务配置
	inspectionTask, err := e.taskRepo.GetByID(ctx, payload.TaskID)
	if err != nil {
		return fmt.Errorf("get inspection task: %w", err)
	}

	// 解析巡检组ID
	var groupIDs []uint
	if inspectionTask.GroupIDs != "" {
		if err := json.Unmarshal([]byte(inspectionTask.GroupIDs), &groupIDs); err != nil {
			return fmt.Errorf("parse group ids: %w", err)
		}
	}

	// 解析巡检项ID（可选）
	var itemIDs []uint
	if inspectionTask.ItemIDs != "" {
		if err := json.Unmarshal([]byte(inspectionTask.ItemIDs), &itemIDs); err != nil {
			return fmt.Errorf("parse item ids: %w", err)
		}
	}

	if len(groupIDs) == 0 {
		return fmt.Errorf("no inspection groups configured for task %d", payload.TaskID)
	}

	// 收集所有需要执行的巡检项
	var itemsToExecute []*inspectionmgmtdata.InspectionItem
	for _, groupID := range groupIDs {
		_, err := e.groupRepo.GetByID(ctx, groupID)
		if err != nil {
			appLogger.Error("get inspection group failed", zap.Uint("groupID", groupID), zap.Error(err))
			continue
		}

		// 获取该组的所有巡检项
		items, err := e.itemRepo.GetByGroupID(ctx, groupID)
		if err != nil {
			appLogger.Error("get inspection items failed", zap.Uint("groupID", groupID), zap.Error(err))
			continue
		}

		// 如果指定了巡检项ID，则过滤
		if len(itemIDs) > 0 {
			filtered := make([]*inspectionmgmtdata.InspectionItem, 0)
			for _, item := range items {
				for _, id := range itemIDs {
					if item.ID == id {
						filtered = append(filtered, item)
						break
					}
				}
			}
			items = filtered
		}

		// 只执行启用的巡检项
		for _, item := range items {
			if item.Status == "enabled" {
				itemsToExecute = append(itemsToExecute, item)
			}
		}
	}

	if len(itemsToExecute) == 0 {
		appLogger.Warn("no enabled inspection items to execute", zap.Uint("taskID", payload.TaskID))
		return nil
	}

	// 创建执行记录主表
	startTime := time.Now()

	// 收集巡检组名称
	groupNames := make([]string, 0, len(groupIDs))
	uniqueGroups := make(map[uint]bool)
	for _, item := range itemsToExecute {
		if !uniqueGroups[item.GroupID] {
			uniqueGroups[item.GroupID] = true
			group, err := e.groupRepo.GetByID(ctx, item.GroupID)
			if err == nil {
				groupNames = append(groupNames, group.Name)
			}
		}
	}

	groupIDsJSON, _ := json.Marshal(groupIDs)
	groupNamesJSON, _ := json.Marshal(groupNames)

	executionRecord := &inspectionmgmtdata.InspectionExecutionRecord{
		TaskID:      payload.TaskID,
		TaskName:    inspectionTask.Name,
		TotalItems:  len(itemsToExecute),
		TotalHosts:  0, // 执行后更新
		Status:      "running",
		StartedAt:   startTime,
		GroupIDs:    string(groupIDsJSON),
		GroupNames:  string(groupNamesJSON),
		TriggerType: triggerType,
	}

	if err := e.execRepo.CreateRecord(ctx, executionRecord); err != nil {
		appLogger.Error("create execution record failed", zap.Error(err))
		return fmt.Errorf("create execution record: %w", err)
	}

	// 并发执行巡检项，收集明细
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failCount int
	details := make([]*inspectionmgmtdata.InspectionExecutionDetail, 0)
	hostSet := make(map[uint]bool)

	for _, item := range itemsToExecute {
		wg.Add(1)
		go func(itm *inspectionmgmtdata.InspectionItem) {
			defer wg.Done()

			// 执行巡检项
			itemResults, err := e.itemSvc.ExecuteItemByID(ctx, itm.ID)
			if err != nil {
				appLogger.Error("execute inspection item failed",
					zap.Uint("itemID", itm.ID),
					zap.String("itemName", itm.Name),
					zap.Error(err))
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}

			// 获取巡检组信息
			group, err := e.groupRepo.GetByID(ctx, itm.GroupID)
			if err != nil {
				appLogger.Error("get inspection group failed", zap.Error(err))
				return
			}

			mu.Lock()
			// 将旧的 InspectionRecord 转换为新的 InspectionExecutionDetail
			for _, oldRecord := range itemResults {
				detail := &inspectionmgmtdata.InspectionExecutionDetail{
					ExecutionID:        executionRecord.ID,
					GroupID:            itm.GroupID,
					GroupName:          group.Name,
					ItemID:             itm.ID,
					ItemName:           itm.Name,
					HostID:             oldRecord.HostID,
					HostName:           "", // 需要从 oldRecord 获取
					HostIP:             "", // 需要从 oldRecord 获取
					Status:             oldRecord.Status,
					Output:             oldRecord.Output,
					ErrorMessage:       oldRecord.ErrorMessage,
					Duration:           oldRecord.Duration,
					AssertionResult:    oldRecord.AssertionResult,
					AssertionDetails:   oldRecord.AssertionDetails,
					ExtractedVariables: oldRecord.ExtractedVariables,
					ExecutedAt:         oldRecord.ExecutedAt,
				}
				details = append(details, detail)
				hostSet[oldRecord.HostID] = true

				if oldRecord.Status == "failed" {
					failCount++
				}
			}
			mu.Unlock()

			// 推送 Metrics（使用旧记录格式）
			if inspectionTask.PushgatewayID > 0 {
				e.pushMetrics(ctx, inspectionTask, itm, itemResults, triggerType)
			}
		}(item)
	}

	wg.Wait()

	// 批量保存明细
	if len(details) > 0 {
		if err := e.execRepo.BatchCreateDetails(ctx, details); err != nil {
			appLogger.Error("batch create details failed", zap.Error(err))
		}
	}

	// 更新执行记录统计
	completedAt := time.Now()
	executionRecord.TotalHosts = len(hostSet)
	executionRecord.TotalExecutions = len(details)
	executionRecord.SuccessCount = len(details) - failCount
	executionRecord.FailedCount = failCount
	executionRecord.Duration = completedAt.Sub(startTime).Seconds()
	executionRecord.CompletedAt = &completedAt

	// 计算断言统计
	assertionPassCount := 0
	assertionFailCount := 0
	assertionSkipCount := 0
	for _, detail := range details {
		switch detail.AssertionResult {
		case "pass":
			assertionPassCount++
		case "fail":
			assertionFailCount++
		case "skip":
			assertionSkipCount++
		}
	}
	executionRecord.AssertionPassCount = assertionPassCount
	executionRecord.AssertionFailCount = assertionFailCount
	executionRecord.AssertionSkipCount = assertionSkipCount

	// 确定最终状态
	if failCount == 0 {
		executionRecord.Status = "success"
	} else if failCount == len(details) {
		executionRecord.Status = "failed"
	} else {
		executionRecord.Status = "partial"
	}

	if err := e.execRepo.UpdateRecord(ctx, executionRecord); err != nil {
		appLogger.Error("update execution record failed", zap.Error(err))
	}

	// 更新任务执行状态
	runResult := executionRecord.Status
	now := time.Now()
	inspectionTask.LastRunAt = &now
	inspectionTask.LastRunStatus = runResult

	// 计算下次执行时间
	if inspectionTask.CronExpr != "" {
		nextRunAt := e.calculateNextRunTime(inspectionTask.CronExpr)
		inspectionTask.NextRunAt = nextRunAt
	}

	if err := e.taskRepo.Update(ctx, inspectionTask); err != nil {
		appLogger.Error("update task last run failed", zap.Error(err))
	}

	appLogger.Info("inspection task executed",
		zap.Uint("taskID", payload.TaskID),
		zap.String("taskName", inspectionTask.Name),
		zap.Uint("executionRecordID", executionRecord.ID),
		zap.Int("totalItems", len(itemsToExecute)),
		zap.Int("totalHosts", executionRecord.TotalHosts),
		zap.Int("totalExecutions", len(details)),
		zap.Int("successCount", executionRecord.SuccessCount),
		zap.Int("failCount", failCount),
		zap.String("result", runResult))

	return nil
}

// pushMetrics 推送巡检指标到 Pushgateway
func (e *InspectionExecutor) pushMetrics(
	ctx context.Context,
	task *inspectionmgmtdata.InspectionTask,
	item *inspectionmgmtdata.InspectionItem,
	records []*inspectionmgmtdata.InspectionRecord,
	triggerType string,
) {
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

	// 获取巡检组信息
	group, err := e.groupRepo.GetByID(ctx, item.GroupID)
	if err != nil {
		appLogger.Error("get inspection group failed", zap.Error(err))
		return
	}

	scheduleMode := triggerType
	if scheduleMode == "" {
		scheduleMode = "scheduled"
	}

	// 为每个执行记录推送指标
	for _, record := range records {
		baseLabels := map[string]string{
			"task_id":        fmt.Sprintf("%d", task.ID),
			"task_name":      task.Name,
			"task_type":      "inspect",
			"business_group": group.Name,
			"owner":          task.Owner,
			"schedule_mode":  scheduleMode,
		}

		// metric label 不含 task_id/host_id（已在 grouping 中，避免冲突）
		allLabels := prometheus.Labels{
			"task_name":      task.Name,
			"task_type":      "inspect",
			"business_group": group.Name,
			"owner":          task.Owner,
			"schedule_mode":  scheduleMode,
			"check_group":    group.Name,
			"check_item":     item.Name,
			"check_level":    "medium",
		}

		labelNames := make([]string, 0, len(allLabels))
		labelValues := make([]string, 0, len(allLabels))
		for k, v := range allLabels {
			labelNames = append(labelNames, k)
			labelValues = append(labelValues, v)
		}

		samples := make([]metrics.MetricSample, 0, 16)
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
		if record.Status == "success" {
			addCounter("srehub_inspect_task_success_total", "Total task successes")
			addCounter("srehub_inspect_check_pass_total", "Inspection check pass count")
		} else {
			addCounter("srehub_inspect_task_fail_total", "Total task failures")
			addCounter("srehub_inspect_check_fail_total", "Inspection check fail count")
			addCounter("srehub_inspect_check_abnormal_total", "Inspection abnormal count")
		}

		// 通用 Gauge
		successVal := 0.0
		if record.Status == "success" {
			successVal = 1.0
		}
		addGauge("srehub_inspect_task_availability_gauge", "Inspection result for this execution (1=pass 0=fail)", successVal)
		addGauge("srehub_inspect_task_exec_duration_seconds", "Task execution duration seconds", float64(record.Duration)/1000.0)
		if e.redisCounter != nil {
			total := e.redisCounter.Get(ctx, "srehub_inspect_task_exec_total", baseLabels)
			success := e.redisCounter.Get(ctx, "srehub_inspect_task_success_total", baseLabels)
			if total > 0 {
				addGauge("srehub_inspect_task_availability", "Task availability ratio (success/total)", success/total)
			}
		}

		// 智能巡检专属 Gauge
		addGauge("srehub_inspect_check_status", "Inspection check status (1=pass 0=fail)", successVal)
		addGauge("srehub_inspect_check_duration_seconds", "Inspection check execution duration seconds", float64(record.Duration)/1000.0)

		// 断言结果
		if record.AssertionResult != "" {
			assertionVal := 0.0
			if record.AssertionResult == "pass" {
				assertionVal = 1.0
			}
			addGauge("srehub_inspect_check_assertion_result", "Inspection check assertion result (1=pass 0=fail)", assertionVal)
		}

		hostname, _ := os.Hostname()
		grouping := map[string]string{
			"instance": hostname,
			"task_id":  fmt.Sprintf("%d", task.ID),
			"group_id": fmt.Sprintf("%d", group.ID),
			"item_id":  fmt.Sprintf("%d", item.ID),
			"host_id":  fmt.Sprintf("%d", record.HostID),
		}

		if err := pusher.PushSamples("srehub", grouping, samples); err != nil {
			appLogger.Error("push inspection metrics failed",
				zap.Uint("taskID", task.ID),
				zap.Uint("itemID", item.ID),
				zap.Uint("recordID", record.ID),
				zap.Error(err),
			)
		}
	}

	appLogger.Info("inspection metrics pushed",
		zap.Uint("taskID", task.ID),
		zap.String("taskName", task.Name),
		zap.String("itemName", item.Name),
		zap.Int("recordCount", len(records)))
}

// calculateNextRunTime 计算下次执行时间
func (e *InspectionExecutor) calculateNextRunTime(cronExpr string) *time.Time {
	// 使用秒级 cron 解析器
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		appLogger.Error("parse cron expression failed", zap.String("cronExpr", cronExpr), zap.Error(err))
		return nil
	}

	nextTime := schedule.Next(time.Now())
	return &nextTime
}
