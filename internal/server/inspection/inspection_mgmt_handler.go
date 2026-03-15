package inspection

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/service/inspection_mgmt"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"go.uber.org/zap"
)

// RegisterInspectionMgmtRoutes 注册巡检管理路由
func (s *HTTPServer) RegisterInspectionMgmtRoutes(r *gin.RouterGroup) {
	inspection := r.Group("/inspection")
	{
		// 统计数据
		inspection.GET("/stats", s.getInspectionStats)

		// 巡检组
		groups := inspection.Group("/groups")
		{
			groups.POST("", s.createInspectionGroup)
			groups.PUT("/:id", s.updateInspectionGroup)
			groups.DELETE("/:id", s.deleteInspectionGroup)
			groups.GET("/:id", s.getInspectionGroup)
			groups.GET("", s.listInspectionGroups)
			groups.GET("/all", s.getAllInspectionGroups)
			groups.POST("/:id/items", s.batchSaveInspectionItems)
			groups.GET("/:id/export", s.exportInspectionGroup)
			groups.GET("/export-all", s.exportAllInspectionGroups)
			groups.POST("/import", s.importInspectionGroup)
			groups.POST("/import-file", s.importInspectionGroupFile)
		}

		// 巡检项
		items := inspection.Group("/items")
		{
			items.POST("", s.createInspectionItem)
			items.PUT("/:id", s.updateInspectionItem)
			items.DELETE("/:id", s.deleteInspectionItem)
			items.GET("/:id", s.getInspectionItem)
			items.GET("", s.listInspectionItems)
			items.POST("/test-run", s.testRunInspectionItems)
		}

		// 执行记录
		records := inspection.Group("/records")
		{
			records.GET("/:id", s.getInspectionRecord)
			records.GET("", s.listInspectionRecords)
			records.GET("/:id/export", s.exportInspectionRecord)
		}

		// 定时任务
		tasks := inspection.Group("/mgmt-tasks")
		{
			tasks.POST("", s.createInspectionTask)
			tasks.PUT("/:id", s.updateInspectionTask)
			tasks.DELETE("/:id", s.deleteInspectionTask)
			tasks.GET("/:id", s.getInspectionTask)
			tasks.GET("", s.listInspectionTasks)
			tasks.PUT("/:id/toggle", s.toggleInspectionTask)
			tasks.GET("/:id/export", s.exportInspectionTask)
		}

		// 调度器管理
		scheduler := inspection.Group("/scheduler")
		{
			scheduler.GET("/stats", s.getSchedulerStats)
			scheduler.POST("/reload", s.reloadScheduler)
		}
	}
}

// ==================== 调度器管理 Handlers ====================

// getSchedulerStats 获取调度器统计信息
func (s *HTTPServer) getSchedulerStats(c *gin.Context) {
	if s.scheduler == nil {
		response.ErrorCode(c, http.StatusServiceUnavailable, "调度器未初始化")
		return
	}

	stats := s.scheduler.GetStats()
	response.Success(c, gin.H{
		"tasks_total":       stats.TasksTotal,
		"tasks_enabled":     stats.TasksEnabled,
		"exec_success":      stats.ExecSuccess,
		"exec_fail":         stats.ExecFail,
		"exec_skipped":      stats.ExecSkipped,
		"lock_acquired":     stats.LockAcquired,
		"lock_skipped":      stats.LockSkipped,
		"last_reload_epoch": stats.LastReloadEpoch,
	})
}

// reloadScheduler 手动重载调度器
func (s *HTTPServer) reloadScheduler(c *gin.Context) {
	if s.scheduler == nil {
		response.ErrorCode(c, http.StatusServiceUnavailable, "调度器未初始化")
		return
	}

	if err := s.scheduler.Reload(c.Request.Context()); err != nil {
		appLogger.Error("手动重载调度器失败", zap.Error(err))
		response.ErrorCode(c, http.StatusInternalServerError, "重载失败: "+err.Error())
		return
	}

	appLogger.Info("调度器已手动重载")
	response.Success(c, gin.H{"message": "调度器重载成功"})
}

// ==================== 巡检组 Handlers ====================

func (s *HTTPServer) createInspectionGroup(c *gin.Context) {
	var req inspection_mgmt.GroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	id, err := s.inspectionGroupService.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

func (s *HTTPServer) updateInspectionGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req inspection_mgmt.GroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.inspectionGroupService.Update(c.Request.Context(), uint(id), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (s *HTTPServer) deleteInspectionGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := s.inspectionGroupService.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (s *HTTPServer) getInspectionGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	group, err := s.inspectionGroupService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, group)
}

func (s *HTTPServer) listInspectionGroups(c *gin.Context) {
	var req inspection_mgmt.GroupListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	groups, total, err := s.inspectionGroupService.List(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     groups,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
	})
}

func (s *HTTPServer) getAllInspectionGroups(c *gin.Context) {
	groups, err := s.inspectionGroupService.GetAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, groups)
}

func (s *HTTPServer) getInspectionStats(c *gin.Context) {
	stats, err := s.inspectionGroupService.GetStats(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, stats)
}

func (s *HTTPServer) exportInspectionGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	format := c.DefaultQuery("format", "json") // 默认 json 格式

	if format != "json" && format != "yaml" {
		response.ErrorCode(c, http.StatusBadRequest, "format 参数必须是 json 或 yaml")
		return
	}

	data, err := s.inspectionGroupService.Export(c.Request.Context(), uint(id), format)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("inspection_group_%d.%s", id, format)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	if format == "yaml" {
		c.Header("Content-Type", "application/x-yaml")
	} else {
		c.Header("Content-Type", "application/json")
	}

	c.String(http.StatusOK, data)
}

func (s *HTTPServer) importInspectionGroup(c *gin.Context) {
	var req inspection_mgmt.GroupImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	ids, err := s.inspectionGroupService.Import(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"ids": ids, "count": len(ids)})
}

func (s *HTTPServer) exportAllInspectionGroups(c *gin.Context) {
	format := c.DefaultQuery("format", "json") // 默认 json 格式

	if format != "json" && format != "yaml" {
		response.ErrorCode(c, http.StatusBadRequest, "format 参数必须是 json 或 yaml")
		return
	}

	data, err := s.inspectionGroupService.ExportAll(c.Request.Context(), format)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("inspection_groups_all.%s", format)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	if format == "yaml" {
		c.Header("Content-Type", "application/x-yaml")
	} else {
		c.Header("Content-Type", "application/json")
	}

	c.String(http.StatusOK, data)
}

func (s *HTTPServer) importInspectionGroupFile(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "请上传文件")
		return
	}

	// 检查文件扩展名
	format := "json"
	if len(file.Filename) > 5 && file.Filename[len(file.Filename)-5:] == ".yaml" {
		format = "yaml"
	} else if len(file.Filename) > 4 && file.Filename[len(file.Filename)-4:] == ".yml" {
		format = "yaml"
	}

	// 读取文件内容
	fileContent, err := file.Open()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取文件失败")
		return
	}
	defer fileContent.Close()

	// 读取文件内容到字符串
	buf := make([]byte, file.Size)
	if _, err := fileContent.Read(buf); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取文件内容失败")
		return
	}

	// 导入
	req := &inspection_mgmt.GroupImportRequest{
		Format: format,
		Data:   string(buf),
	}

	ids, err := s.inspectionGroupService.Import(c.Request.Context(), req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"ids": ids, "count": len(ids)})
}

// ==================== 巡检项 Handlers ====================

func (s *HTTPServer) createInspectionItem(c *gin.Context) {
	var req inspection_mgmt.ItemCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	id, err := s.inspectionItemService.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

func (s *HTTPServer) updateInspectionItem(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req inspection_mgmt.ItemUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.inspectionItemService.Update(c.Request.Context(), uint(id), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (s *HTTPServer) deleteInspectionItem(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := s.inspectionItemService.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (s *HTTPServer) getInspectionItem(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	item, err := s.inspectionItemService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, item)
}

func (s *HTTPServer) listInspectionItems(c *gin.Context) {
	var req inspection_mgmt.ItemListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	items, total, err := s.inspectionItemService.List(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     items,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
	})
}

func (s *HTTPServer) batchSaveInspectionItems(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Items []inspection_mgmt.ItemCreateRequest `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 添加日志
	fmt.Printf("[DEBUG] batchSaveInspectionItems - groupID: %d\n", groupID)
	fmt.Printf("[DEBUG] batchSaveInspectionItems - items count: %d\n", len(req.Items))
	for i, item := range req.Items {
		fmt.Printf("[DEBUG] Item %d:\n", i+1)
		fmt.Printf("  Name: %s\n", item.Name)
		fmt.Printf("  HostMatchType: %s\n", item.HostMatchType)
		fmt.Printf("  HostTags: %s\n", item.HostTags)
		fmt.Printf("  HostIDs: %s\n", item.HostIDs)
	}

	if err := s.inspectionItemService.BatchSave(c.Request.Context(), uint(groupID), req.Items); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (s *HTTPServer) testRunInspectionItems(c *gin.Context) {
	var req inspection_mgmt.TestRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	results, err := s.inspectionItemService.TestRun(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"success": true,
		"results": results,
	})
}

// ==================== 执行记录 Handlers ====================

func (s *HTTPServer) getInspectionRecord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	record, err := s.inspectionRecordService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, record)
}

func (s *HTTPServer) listInspectionRecords(c *gin.Context) {
	var req inspection_mgmt.RecordListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	records, total, err := s.inspectionRecordService.List(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     records,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
	})
}

func (s *HTTPServer) exportInspectionRecord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 生成 Excel 文件
	f, err := s.inspectionRecordService.ExportRecordToExcel(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("inspection_record_%d.xlsx", id)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Transfer-Encoding", "binary")

	// 写入响应
	if err := f.Write(c.Writer); err != nil {
		appLogger.Error("write excel failed", zap.Error(err))
	}
}

// ==================== 定时任务 Handlers ====================

func (s *HTTPServer) createInspectionTask(c *gin.Context) {
	var req inspection_mgmt.TaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	err := s.inspectionTaskService.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 重载调度器以使新任务生效
	if s.scheduler != nil {
		if err := s.scheduler.Reload(c.Request.Context()); err != nil {
			appLogger.Error("重载调度器失败", zap.Error(err))
		} else {
			appLogger.Info("调度器已重载（创建任务）")
		}
	}

	response.Success(c, gin.H{"message": "创建成功"})
}

func (s *HTTPServer) updateInspectionTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req inspection_mgmt.TaskUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	err := s.inspectionTaskService.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 重载调度器以使任务更新生效
	if s.scheduler != nil {
		if err := s.scheduler.Reload(c.Request.Context()); err != nil {
			appLogger.Error("重载调度器失败", zap.Error(err))
		} else {
			appLogger.Info("调度器已重载（更新任务）")
		}
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

func (s *HTTPServer) deleteInspectionTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	err := s.inspectionTaskService.Delete(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 重载调度器以移除已删除的任务
	if s.scheduler != nil {
		if err := s.scheduler.Reload(c.Request.Context()); err != nil {
			appLogger.Error("重载调度器失败", zap.Error(err))
		} else {
			appLogger.Info("调度器已重载（删除任务）")
		}
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

func (s *HTTPServer) getInspectionTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	task, err := s.inspectionTaskService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, task)
}

func (s *HTTPServer) listInspectionTasks(c *gin.Context) {
	var req inspection_mgmt.TaskListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	tasks, total, err := s.inspectionTaskService.List(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     tasks,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
	})
}

func (s *HTTPServer) toggleInspectionTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	task, err := s.inspectionTaskService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, err.Error())
		return
	}

	// 切换状态 - 保留所有其他字段
	updateReq := &inspection_mgmt.TaskUpdateRequest{
		Name:          task.Name,
		Description:   task.Description,
		TaskType:      task.TaskType,
		CronExpr:      task.CronExpr,
		Enabled:       !task.Enabled, // 只切换 Enabled 字段
		GroupIDs:      task.GroupIDs,
		ItemIDs:       task.ItemIDs,
		PushgatewayID: task.PushgatewayID,
		Concurrency:   task.Concurrency,
	}

	err = s.inspectionTaskService.Update(c.Request.Context(), uint(id), updateReq)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 重载调度器以使状态切换生效
	if s.scheduler != nil {
		if err := s.scheduler.Reload(c.Request.Context()); err != nil {
			appLogger.Error("重载调度器失败", zap.Error(err))
		} else {
			appLogger.Info("调度器已重载（切换任务状态）", zap.Uint("taskID", uint(id)), zap.Bool("enabled", !task.Enabled))
		}
	}

	response.Success(c, gin.H{"message": "操作成功"})
}

func (s *HTTPServer) exportInspectionTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 生成 Excel 文件
	f, err := s.inspectionTaskService.ExportTaskToExcel(
		c.Request.Context(),
		uint(id),
		s.hostRepo,
		s.recordRepo,
		s.itemRepo,
		s.groupRepo,
	)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("inspection_task_%d.xlsx", id)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Transfer-Encoding", "binary")

	// 写入响应
	if err := f.Write(c.Writer); err != nil {
		appLogger.Error("write excel failed", zap.Error(err))
	}
}
