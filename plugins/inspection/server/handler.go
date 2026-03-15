package server

import (
	"net/http"
	"strconv"

	"github.com/ydcloud-dy/opshub/pkg/response"
	"github.com/ydcloud-dy/opshub/plugins/inspection/dto"
	"github.com/ydcloud-dy/opshub/plugins/inspection/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	groupService  *service.GroupService
	itemService   *service.ItemService
	taskService   *service.TaskService
	recordService *service.RecordService
}

func NewHandler(
	groupService *service.GroupService,
	itemService *service.ItemService,
	taskService *service.TaskService,
	recordService *service.RecordService,
) *Handler {
	return &Handler{
		groupService:  groupService,
		itemService:   itemService,
		taskService:   taskService,
		recordService: recordService,
	}
}

// Group handlers

func (h *Handler) CreateGroup(c *gin.Context) {
	var req dto.GroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.groupService.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *Handler) UpdateGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req dto.GroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.groupService.Update(c.Request.Context(), uint(id), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *Handler) DeleteGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.groupService.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *Handler) GetGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	group, err := h.groupService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, group)
}

func (h *Handler) ListGroups(c *gin.Context) {
	var req dto.GroupListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	groups, total, err := h.groupService.List(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  groups,
		"total": total,
	})
}

func (h *Handler) GetAllGroups(c *gin.Context) {
	groups, err := h.groupService.GetAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, groups)
}

// Item handlers

func (h *Handler) CreateItem(c *gin.Context) {
	var req dto.ItemCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.itemService.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *Handler) UpdateItem(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req dto.ItemUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.itemService.Update(c.Request.Context(), uint(id), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *Handler) DeleteItem(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.itemService.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *Handler) GetItem(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	item, err := h.itemService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, item)
}

func (h *Handler) ListItems(c *gin.Context) {
	var req dto.ItemListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	items, total, err := h.itemService.List(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  items,
		"total": total,
	})
}

func (h *Handler) TestRunItems(c *gin.Context) {
	var req dto.TestRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	results, err := h.itemService.TestRun(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, results)
}

// Task handlers

func (h *Handler) CreateTask(c *gin.Context) {
	var req dto.TaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.taskService.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *Handler) UpdateTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req dto.TaskUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.taskService.Update(c.Request.Context(), uint(id), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *Handler) DeleteTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.taskService.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *Handler) GetTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	task, err := h.taskService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, task)
}

func (h *Handler) ListTasks(c *gin.Context) {
	var req dto.TaskListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	tasks, total, err := h.taskService.List(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  tasks,
		"total": total,
	})
}

// Record handlers

func (h *Handler) GetRecord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	record, err := h.recordService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, record)
}

func (h *Handler) ListRecords(c *gin.Context) {
	var req dto.RecordListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	records, total, err := h.recordService.List(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  records,
		"total": total,
	})
}

// BatchSaveItems 批量保存巡检项
func (h *Handler) BatchSaveItems(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Items []dto.ItemCreateRequest `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.itemService.BatchSave(c.Request.Context(), uint(groupID), req.Items); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetStats 获取统计数据
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.groupService.GetStats(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, stats)
}
