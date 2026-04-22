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

// GroupBusinessGroupOverride 巡检组业务分组覆盖结构（支持多选）
type GroupBusinessGroupOverride struct {
	GroupID          uint   `json:"group_id"`
	BusinessGroupIDs []uint `json:"business_group_ids"` // 支持多个业务分组
}

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

	// 需求一：解析任务级执行方式覆盖配置
	taskExecutionMode := inspectionTask.ExecutionMode // 覆盖巡检组执行方式（空=不覆盖）
	// 解析任务级全局业务分组覆盖（支持多选）
	var taskBusinessGroupIDs []uint
	if inspectionTask.BusinessGroupIDs != "" {
		_ = json.Unmarshal([]byte(inspectionTask.BusinessGroupIDs), &taskBusinessGroupIDs)
	}

	// 需求三：解析任务级自定义变量（巡检任务，优先级最高）
	taskExtraVars := make(map[string]string)
	if inspectionTask.CustomVariables != "" {
		_ = json.Unmarshal([]byte(inspectionTask.CustomVariables), &taskExtraVars)
	}

	// 解析断言覆盖配置
	var assertionOverrides []ItemAssertionOverride
	if inspectionTask.ItemAssertionOverrides != "" {
		if err := json.Unmarshal([]byte(inspectionTask.ItemAssertionOverrides), &assertionOverrides); err != nil {
			appLogger.Warn("解析断言覆盖失败", zap.Error(err))
		}
	}

	// 构建断言覆盖映射表，便于快速查找
	assertionOverrideMap := make(map[uint]*ItemAssertionOverride)
	for i := range assertionOverrides {
		assertionOverrideMap[assertionOverrides[i].ItemID] = &assertionOverrides[i]
	}

	// 解析业务分组覆盖配置
	var groupBusinessGroupOverrides []GroupBusinessGroupOverride
	if inspectionTask.GroupBusinessGroupOverrides != "" {
		if err := json.Unmarshal([]byte(inspectionTask.GroupBusinessGroupOverrides), &groupBusinessGroupOverrides); err != nil {
			appLogger.Warn("解析业务分组覆盖失败", zap.Error(err))
		}
	}

	// 构建业务分组覆盖映射表
	groupBusinessGroupOverrideMap := make(map[uint]*GroupBusinessGroupOverride)
	for i := range groupBusinessGroupOverrides {
		groupBusinessGroupOverrideMap[groupBusinessGroupOverrides[i].GroupID] = &groupBusinessGroupOverrides[i]
	}

	// 收集所有需要执行的巡检项（按巡检组分组，以便应用不同的业务分组覆盖）
	type GroupItems struct {
		GroupID                   uint
		Items                     []*inspectionmgmtdata.InspectionItem
		EffectiveBusinessGroupIDs []uint // 应用优先级后的业务分组ID列表（支持多选）
	}
	groupItemsMap := make(map[uint]*GroupItems)

	for _, groupID := range groupIDs {
		_, err := e.groupRepo.GetByID(ctx, groupID)
		if err != nil {
			appLogger.Error("get inspection group failed", zap.Uint("groupID", groupID), zap.Error(err))
			continue
		}

		// 应用业务分组覆盖优先级（支持多选）
		var effectiveBusinessGroupIDs []uint

		// 优先级 1：巡检组级业务分组覆盖（最高优先级）
		if groupOverride, exists := groupBusinessGroupOverrideMap[groupID]; exists && len(groupOverride.BusinessGroupIDs) > 0 {
			effectiveBusinessGroupIDs = groupOverride.BusinessGroupIDs
			appLogger.Info("应用巡检组级业务分组覆盖",
				zap.Uint("group_id", groupID),
				zap.Any("business_group_ids", effectiveBusinessGroupIDs))
		} else if len(taskBusinessGroupIDs) > 0 {
			// 优先级 2：任务级全局业务分组覆盖
			effectiveBusinessGroupIDs = taskBusinessGroupIDs
			appLogger.Info("应用任务级全局业务分组覆盖",
				zap.Uint("group_id", groupID),
				zap.Any("business_group_ids", effectiveBusinessGroupIDs))
		}
		// 优先级 3：如果 effectiveBusinessGroupIDs 为空，则使用巡检组原始配置（在 ExecuteItemByIDWithOverride 中处理）

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

		// 只收集启用的巡检项
		enabledItems := make([]*inspectionmgmtdata.InspectionItem, 0)
		for _, item := range items {
			if item.Status == "enabled" {
				enabledItems = append(enabledItems, item)
			}
		}

		if len(enabledItems) > 0 {
			groupItemsMap[groupID] = &GroupItems{
				GroupID:                   groupID,
				Items:                     enabledItems,
				EffectiveBusinessGroupIDs: effectiveBusinessGroupIDs,
			}
		}
	}

	// 统计总巡检项数
	var itemsToExecute []*inspectionmgmtdata.InspectionItem
	for _, gi := range groupItemsMap {
		itemsToExecute = append(itemsToExecute, gi.Items...)
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

			// 查找该巡检项所属巡检组的有效业务分组ID列表（应用优先级后）
			var effectiveBusinessGroupIDs []uint
			if gi, exists := groupItemsMap[itm.GroupID]; exists {
				effectiveBusinessGroupIDs = gi.EffectiveBusinessGroupIDs
			}

			// 查找该巡检项的断言覆盖
			var itemOverride *ItemAssertionOverride
			if override, exists := assertionOverrideMap[itm.ID]; exists {
				itemOverride = override
			}

			// 需求一：使用覆盖参数执行巡检项（业务分组覆盖 + 断言覆盖）
			itemResults, err := e.itemSvc.ExecuteItemByIDWithOverride(ctx, itm.ID, taskExecutionMode, effectiveBusinessGroupIDs, taskExtraVars, itemOverride)
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

			// 获取该巡检项的有效断言配置（覆盖后）
			effectiveAssertionType := itm.AssertionType
			effectiveAssertionValue := itm.AssertionValue
			if itemOverride != nil {
				effectiveAssertionType = itemOverride.AssertionType
				effectiveAssertionValue = itemOverride.AssertionValue
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
					HostName:           oldRecord.HostName,
					HostIP:             oldRecord.HostIP,
					// 填充执行配置信息（使用覆盖后的值）
					BusinessGroup:      oldRecord.BusinessGroup,
					ExecutionType:      itm.ExecutionType,
					ExecutionMode:      group.ExecutionMode,
					Command:            itm.Command,
					ScriptType:         itm.ScriptType,
					ScriptContent:      itm.ScriptContent,
					AssertionType:      effectiveAssertionType,
					AssertionValue:     effectiveAssertionValue,
					// 巡检级别和风险等级（快照）
					InspectionLevel:    itm.InspectionLevel,
					RiskLevel:          itm.RiskLevel,
					// 执行结果
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

// RunSyncResult 同步执行结果（需求二）
type RunSyncResult struct {
	TaskID      uint                    `json:"task_id"`
	TaskName    string                  `json:"task_name"`
	TaskType    string                  `json:"task_type"`
	Status      string                  `json:"status"`      // success/failed/partial
	Duration    float64                 `json:"duration"`    // 秒
	TotalItems  int                     `json:"total_items"`
	SuccessCount int                    `json:"success_count"`
	FailedCount int                     `json:"failed_count"`
	Details     []RunSyncItemDetail     `json:"details"`
}

// RunSyncItemDetail 每个巡检项/拨测项的执行明细
type RunSyncItemDetail struct {
	GroupID          uint     `json:"group_id"`
	GroupName        string   `json:"group_name"`
	ItemID           uint     `json:"item_id"`
	ItemName         string   `json:"item_name"`
	HostID           uint     `json:"host_id,omitempty"`
	HostName         string   `json:"host_name,omitempty"`
	HostIP           string   `json:"host_ip,omitempty"`
	// 执行配置信息
	BusinessGroup    string   `json:"business_group,omitempty"`
	ExecutionType    string   `json:"execution_type,omitempty"`
	ExecutionMode    string   `json:"execution_mode,omitempty"`
	Command          string   `json:"command,omitempty"`
	ScriptType       string   `json:"script_type,omitempty"`
	ScriptContent    string   `json:"script_content,omitempty"`
	AssertionType    string   `json:"assertion_type,omitempty"`
	AssertionValue   string   `json:"assertion_value,omitempty"`
	InspectionLevel  string   `json:"inspectionLevel,omitempty"`
	RiskLevel        string   `json:"riskLevel,omitempty"`
	// 执行结果
	Status           string   `json:"status"`           // success/failed
	Output           string   `json:"output"`
	ErrorMessage     string   `json:"error_message,omitempty"`
	Duration         float64  `json:"duration"`
	AssertionResult  string   `json:"assertion_result,omitempty"`  // pass/fail/skip
	AssertionDetails string   `json:"assertion_details,omitempty"` // JSON
	ExtractedVars    string   `json:"extracted_vars,omitempty"`    // JSON
	ExecutedAt       string   `json:"executed_at"`
}

// ExecuteSync 同步执行巡检任务，阻塞直到完成，返回完整结果（需求二）
func (e *InspectionExecutor) ExecuteSync(ctx context.Context, taskID uint) (*RunSyncResult, error) {
	inspectionTask, err := e.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get inspection task: %w", err)
	}

	// 解析巡检组 ID
	var groupIDs []uint
	if inspectionTask.GroupIDs != "" {
		if err := json.Unmarshal([]byte(inspectionTask.GroupIDs), &groupIDs); err != nil {
			return nil, fmt.Errorf("parse group ids: %w", err)
		}
	}
	var itemIDs []uint
	if inspectionTask.ItemIDs != "" {
		_ = json.Unmarshal([]byte(inspectionTask.ItemIDs), &itemIDs)
	}

	if len(groupIDs) == 0 {
		return nil, fmt.Errorf("no inspection groups configured for task %d", taskID)
	}

	taskExecutionMode := inspectionTask.ExecutionMode
	// 解析任务级全局业务分组覆盖（支持多选）
	var taskBusinessGroupIDs []uint
	if inspectionTask.BusinessGroupIDs != "" {
		_ = json.Unmarshal([]byte(inspectionTask.BusinessGroupIDs), &taskBusinessGroupIDs)
	}
	// 需求三：解析任务级自定义变量（最高优先级）
	taskExtraVars := make(map[string]string)
	if inspectionTask.CustomVariables != "" {
		_ = json.Unmarshal([]byte(inspectionTask.CustomVariables), &taskExtraVars)
	}

	// 解析断言覆盖配置
	var assertionOverrides []ItemAssertionOverride
	if inspectionTask.ItemAssertionOverrides != "" {
		if err := json.Unmarshal([]byte(inspectionTask.ItemAssertionOverrides), &assertionOverrides); err != nil {
			appLogger.Warn("解析断言覆盖失败", zap.Error(err))
		}
	}
	assertionOverrideMap := make(map[uint]*ItemAssertionOverride)
	for i := range assertionOverrides {
		assertionOverrideMap[assertionOverrides[i].ItemID] = &assertionOverrides[i]
	}

	// 解析业务分组覆盖配置
	var groupBusinessGroupOverrides []GroupBusinessGroupOverride
	if inspectionTask.GroupBusinessGroupOverrides != "" {
		if err := json.Unmarshal([]byte(inspectionTask.GroupBusinessGroupOverrides), &groupBusinessGroupOverrides); err != nil {
			appLogger.Warn("解析业务分组覆盖失败", zap.Error(err))
		}
	}
	groupBusinessGroupOverrideMap := make(map[uint]*GroupBusinessGroupOverride)
	for i := range groupBusinessGroupOverrides {
		groupBusinessGroupOverrideMap[groupBusinessGroupOverrides[i].GroupID] = &groupBusinessGroupOverrides[i]
	}

	// 收集要执行的巡检项，按巡检组分组
	type GroupItems struct {
		GroupID                   uint
		Items                     []*inspectionmgmtdata.InspectionItem
		EffectiveBusinessGroupIDs []uint // 应用优先级后的业务分组ID列表（支持多选）
	}
	groupItemsMap := make(map[uint]*GroupItems)
	groupNameMap := make(map[uint]string)
	for _, groupID := range groupIDs {
		group, err := e.groupRepo.GetByID(ctx, groupID)
		if err != nil {
			continue
		}
		groupNameMap[groupID] = group.Name

		// 应用业务分组覆盖优先级（支持多选）
		var effectiveBusinessGroupIDs []uint
		// 优先级 1：巡检组级业务分组覆盖（最高优先级）
		if groupOverride, exists := groupBusinessGroupOverrideMap[groupID]; exists && len(groupOverride.BusinessGroupIDs) > 0 {
			effectiveBusinessGroupIDs = groupOverride.BusinessGroupIDs
			appLogger.Info("应用巡检组级业务分组覆盖",
				zap.Uint("group_id", groupID),
				zap.Any("business_group_ids", effectiveBusinessGroupIDs))
		} else if len(taskBusinessGroupIDs) > 0 {
			// 优先级 2：任务级全局业务分组覆盖
			effectiveBusinessGroupIDs = taskBusinessGroupIDs
			appLogger.Info("应用任务级全局业务分组覆盖",
				zap.Uint("group_id", groupID),
				zap.Any("business_group_ids", effectiveBusinessGroupIDs))
		}
		// 优先级 3：如果 effectiveBusinessGroupIDs 为空，则使用巡检组原始配置（在 ExecuteItemByIDWithOverride 中处理）

		items, err := e.itemRepo.GetByGroupID(ctx, groupID)
		if err != nil {
			continue
		}

		groupItems := &GroupItems{
			GroupID:                   groupID,
			Items:                     make([]*inspectionmgmtdata.InspectionItem, 0),
			EffectiveBusinessGroupIDs: effectiveBusinessGroupIDs,
		}

		if len(itemIDs) > 0 {
			for _, item := range items {
				for _, id := range itemIDs {
					if item.ID == id && item.Status == "enabled" {
						groupItems.Items = append(groupItems.Items, item)
						break
					}
				}
			}
		} else {
			for _, item := range items {
				if item.Status == "enabled" {
					groupItems.Items = append(groupItems.Items, item)
				}
			}
		}

		groupItemsMap[groupID] = groupItems
	}

	// 展平为执行列表
	var itemsToExecute []*inspectionmgmtdata.InspectionItem
	for _, gi := range groupItemsMap {
		itemsToExecute = append(itemsToExecute, gi.Items...)
	}

	startTime := time.Now()
	var mu sync.Mutex
	var wg sync.WaitGroup
	var details []RunSyncItemDetail
	failCount := 0

	for _, item := range itemsToExecute {
		wg.Add(1)
		go func(itm *inspectionmgmtdata.InspectionItem) {
			defer wg.Done()

			// 查找该巡检项所属巡检组的有效业务分组ID列表（应用优先级后）
			var effectiveBusinessGroupIDs []uint
			if gi, exists := groupItemsMap[itm.GroupID]; exists {
				effectiveBusinessGroupIDs = gi.EffectiveBusinessGroupIDs
			}

			// 查找该巡检项的断言覆盖
			var itemOverride *ItemAssertionOverride
			if override, exists := assertionOverrideMap[itm.ID]; exists {
				itemOverride = override
			}

			records, err := e.itemSvc.ExecuteItemByIDWithOverride(ctx, itm.ID, taskExecutionMode, effectiveBusinessGroupIDs, taskExtraVars, itemOverride)
			groupName := groupNameMap[itm.GroupID]
			if err != nil {
				mu.Lock()
				details = append(details, RunSyncItemDetail{
					GroupID:      itm.GroupID,
					GroupName:    groupName,
					ItemID:       itm.ID,
					ItemName:     itm.Name,
					Status:       "failed",
					ErrorMessage: err.Error(),
					ExecutedAt:   time.Now().Format(time.RFC3339),
				})
				failCount++
				mu.Unlock()
				return
			}
			mu.Lock()
			for _, r := range records {
				// 获取巡检组信息
				group, _ := e.groupRepo.GetByID(ctx, itm.GroupID)

				// 计算实际使用的断言配置（应用覆盖）
				effectiveAssertionType := itm.AssertionType
				effectiveAssertionValue := itm.AssertionValue
				if itemOverride != nil {
					effectiveAssertionType = itemOverride.AssertionType
					effectiveAssertionValue = itemOverride.AssertionValue
				}

				d := RunSyncItemDetail{
					GroupID:          itm.GroupID,
					GroupName:        groupName,
					ItemID:           itm.ID,
					ItemName:         itm.Name,
					HostID:           r.HostID,
					HostName:         r.HostName,
					HostIP:           r.HostIP,
					// 执行配置信息（使用覆盖后的值）
					BusinessGroup:    r.BusinessGroup,
					ExecutionType:    itm.ExecutionType,
					ExecutionMode:    group.ExecutionMode,
					Command:          itm.Command,
					ScriptType:       itm.ScriptType,
					ScriptContent:    itm.ScriptContent,
					AssertionType:    effectiveAssertionType,
					AssertionValue:   effectiveAssertionValue,
					InspectionLevel:  itm.InspectionLevel,
					RiskLevel:        itm.RiskLevel,
					// 执行结果
					Status:           r.Status,
					Output:           r.Output,
					ErrorMessage:     r.ErrorMessage,
					Duration:         r.Duration,
					AssertionResult:  r.AssertionResult,
					AssertionDetails: r.AssertionDetails,
					ExtractedVars:    r.ExtractedVariables,
					ExecutedAt:       r.ExecutedAt.Format(time.RFC3339),
				}
				details = append(details, d)
				if r.Status == "failed" {
					failCount++
				}
			}
			mu.Unlock()
		}(item)
	}
	wg.Wait()

	duration := time.Since(startTime).Seconds()
	status := "success"
	if failCount == len(details) && len(details) > 0 {
		status = "failed"
	} else if failCount > 0 {
		status = "partial"
	}

	return &RunSyncResult{
		TaskID:       taskID,
		TaskName:     inspectionTask.Name,
		TaskType:     "inspection",
		Status:       status,
		Duration:     duration,
		TotalItems:   len(itemsToExecute),
		SuccessCount: len(details) - failCount,
		FailedCount:  failCount,
		Details:      details,
	}, nil
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
