package inspection

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"github.com/ydcloud-dy/opshub/pkg/scheduler"
)

// ProbeTaskService handles HTTP requests for probe tasks.
type ProbeTaskService struct {
	useCase       *biz.ProbeTaskUseCase
	resultUseCase *biz.ProbeResultUseCase
	scheduler     *scheduler.Scheduler
}

func NewProbeTaskService(uc *biz.ProbeTaskUseCase, ruc *biz.ProbeResultUseCase, sched *scheduler.Scheduler) *ProbeTaskService {
	return &ProbeTaskService{useCase: uc, resultUseCase: ruc, scheduler: sched}
}

func (s *ProbeTaskService) Create(c *gin.Context) {
	var req biz.ProbeTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	model := req.ToModel()
	if err := s.useCase.CreateWithConfigs(c.Request.Context(), model, req.ProbeConfigIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}
	if s.scheduler != nil {
		s.scheduler.Reload(c.Request.Context())
	}
	response.Success(c, gin.H{
		"id": model.ID, "name": model.Name, "probeConfigIds": req.ProbeConfigIDs,
		"cronExpr": model.CronExpr, "concurrency": model.Concurrency, "status": model.Status,
	})
}

func (s *ProbeTaskService) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req biz.ProbeTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	model := req.ToModel()
	model.ID = uint(id)
	if err := s.useCase.UpdateWithConfigs(c.Request.Context(), model, req.ProbeConfigIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}
	if s.scheduler != nil {
		s.scheduler.Reload(c.Request.Context())
	}
	response.Success(c, gin.H{
		"id": model.ID, "name": model.Name, "probeConfigIds": req.ProbeConfigIDs,
		"cronExpr": model.CronExpr, "concurrency": model.Concurrency, "status": model.Status,
	})
}

func (s *ProbeTaskService) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}
	if s.scheduler != nil {
		s.scheduler.Reload(c.Request.Context())
	}
	response.Success(c, nil)
}

func (s *ProbeTaskService) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	task, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "未找到: "+err.Error())
		return
	}
	configIDs, _ := s.useCase.GetConfigIDs(c.Request.Context(), task.ID)
	response.Success(c, gin.H{
		"id": task.ID, "name": task.Name, "groupId": task.GroupID,
		"cronExpr": task.CronExpr, "pushgatewayId": task.PushgatewayID,
		"concurrency": task.Concurrency, "status": task.Status,
		"lastRunAt": task.LastRunAt, "lastResult": task.LastResult,
		"description": task.Description, "createdAt": task.CreatedAt,
		"updatedAt": task.UpdatedAt, "probeConfigIds": configIDs,
	})
}

func (s *ProbeTaskService) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	var status *int8
	if sv := c.Query("status"); sv != "" {
		v, _ := strconv.ParseInt(sv, 10, 8)
		s8 := int8(v)
		status = &s8
	}

	tasks, total, err := s.useCase.List(c.Request.Context(), page, pageSize, keyword, status)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	// Batch load configIDs to avoid N+1
	taskIDs := make([]uint, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}
	configIDsMap, _ := s.useCase.BatchGetConfigIDs(c.Request.Context(), taskIDs)

	items := make([]gin.H, len(tasks))
	for i, t := range tasks {
		ids := configIDsMap[t.ID]
		if ids == nil {
			ids = []uint{}
		}
		items[i] = gin.H{
			"id": t.ID, "name": t.Name, "groupId": t.GroupID,
			"cronExpr": t.CronExpr, "pushgatewayId": t.PushgatewayID,
			"concurrency": t.Concurrency, "status": t.Status,
			"lastRunAt": t.LastRunAt, "lastResult": t.LastResult,
			"description": t.Description, "createdAt": t.CreatedAt,
			"updatedAt": t.UpdatedAt, "probeConfigIds": ids,
		}
	}
	response.Pagination(c, total, page, pageSize, items)
}

func (s *ProbeTaskService) Toggle(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	task, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "未找到: "+err.Error())
		return
	}
	if task.Status == 1 {
		task.Status = 0
	} else {
		task.Status = 1
	}
	if err := s.useCase.Update(c.Request.Context(), task); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}
	if s.scheduler != nil {
		s.scheduler.Reload(c.Request.Context())
	}
	response.Success(c, task)
}

func (s *ProbeTaskService) Results(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	results, total, err := s.resultUseCase.ListByTaskID(c.Request.Context(), uint(id), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	response.Pagination(c, total, page, pageSize, results)
}
