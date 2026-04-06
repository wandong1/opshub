package inspection_mgmt

import (
	"context"
	"time"

	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
)

type TaskService struct {
	taskRepo inspectionmgmtdata.TaskRepository
}

func NewTaskService(taskRepo inspectionmgmtdata.TaskRepository) *TaskService {
	return &TaskService{
		taskRepo: taskRepo,
	}
}

func (s *TaskService) Create(ctx context.Context, req *TaskCreateRequest) error {
	task := &inspectionmgmtdata.InspectionTask{
		Name:            req.Name,
		Description:     req.Description,
		TaskType:        req.TaskType,
		CronExpr:        req.CronExpr,
		Enabled:         req.Enabled,
		GroupIDs:        req.GroupIDs,
		ItemIDs:         req.ItemIDs,
		PushgatewayID:   req.PushgatewayID,
		Concurrency:     req.Concurrency,
		Owner:           req.Owner,
		ExecutionMode:   req.ExecutionMode,
		AgentHostIDs:    req.AgentHostIDs,
		BusinessGroupID: req.BusinessGroupID,
		CustomVariables: req.CustomVariables,
		Status:          "pending",
	}

	return s.taskRepo.Create(ctx, task)
}

func (s *TaskService) Update(ctx context.Context, id uint, req *TaskUpdateRequest) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if req.Name != "" {
		task.Name = req.Name
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.TaskType != "" {
		task.TaskType = req.TaskType
	}
	if req.CronExpr != "" {
		task.CronExpr = req.CronExpr
	}
	task.Enabled = req.Enabled
	if req.GroupIDs != "" {
		task.GroupIDs = req.GroupIDs
	}
	if req.ItemIDs != "" {
		task.ItemIDs = req.ItemIDs
	}
	task.PushgatewayID = req.PushgatewayID
	if req.Concurrency > 0 {
		task.Concurrency = req.Concurrency
	}
	task.Owner = req.Owner
	// 需求一：更新执行覆盖配置（允许置空，故直接赋值）
	task.ExecutionMode = req.ExecutionMode
	task.AgentHostIDs = req.AgentHostIDs
	task.BusinessGroupID = req.BusinessGroupID
	task.CustomVariables = req.CustomVariables

	return s.taskRepo.Update(ctx, task)
}

func (s *TaskService) Delete(ctx context.Context, id uint) error {
	return s.taskRepo.Delete(ctx, id)
}

func (s *TaskService) GetByID(ctx context.Context, id uint) (*TaskResponse, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(task), nil
}

func (s *TaskService) List(ctx context.Context, req *TaskListRequest) ([]*TaskResponse, int64, error) {
	tasks, total, err := s.taskRepo.List(ctx, req.Page, req.PageSize, req.Name, req.Enabled)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = s.toResponse(task)
	}

	return responses, total, nil
}

func (s *TaskService) toResponse(task *inspectionmgmtdata.InspectionTask) *TaskResponse {
	resp := &TaskResponse{
		ID:              task.ID,
		Name:            task.Name,
		Description:     task.Description,
		TaskType:        task.TaskType,
		CronExpr:        task.CronExpr,
		Status:          task.Status,
		Enabled:         task.Enabled,
		GroupIDs:        task.GroupIDs,
		ItemIDs:         task.ItemIDs,
		PushgatewayID:   task.PushgatewayID,
		Concurrency:     task.Concurrency,
		Owner:           task.Owner,
		ExecutionMode:   task.ExecutionMode,
		AgentHostIDs:    task.AgentHostIDs,
		BusinessGroupID: task.BusinessGroupID,
		CustomVariables: task.CustomVariables,
		LastRunStatus:   task.LastRunStatus,
		CreatedAt:       task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       task.UpdatedAt.Format(time.RFC3339),
	}

	if task.LastRunAt != nil {
		resp.LastRunAt = task.LastRunAt.Format(time.RFC3339)
	}
	if task.NextRunAt != nil {
		resp.NextRunAt = task.NextRunAt.Format(time.RFC3339)
	}

	return resp
}
