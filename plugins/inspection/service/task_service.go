package service

import (
	"context"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/inspection/dto"
	"github.com/ydcloud-dy/opshub/plugins/inspection/model"
	"github.com/ydcloud-dy/opshub/plugins/inspection/repository"
)

type TaskService struct {
	taskRepo repository.TaskRepository
}

func NewTaskService(taskRepo repository.TaskRepository) *TaskService {
	return &TaskService{
		taskRepo: taskRepo,
	}
}

func (s *TaskService) Create(ctx context.Context, req *dto.TaskCreateRequest) error {
	task := &model.InspectionTask{
		Name:          req.Name,
		Description:   req.Description,
		CronExpr:      req.CronExpr,
		Enabled:       req.Enabled,
		GroupIDs:      req.GroupIDs,
		ItemIDs:       req.ItemIDs,
		PushgatewayID: req.PushgatewayID,
		Status:        "pending",
	}

	return s.taskRepo.Create(ctx, task)
}

func (s *TaskService) Update(ctx context.Context, id uint, req *dto.TaskUpdateRequest) error {
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

	return s.taskRepo.Update(ctx, task)
}

func (s *TaskService) Delete(ctx context.Context, id uint) error {
	return s.taskRepo.Delete(ctx, id)
}

func (s *TaskService) GetByID(ctx context.Context, id uint) (*dto.TaskResponse, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(task), nil
}

func (s *TaskService) List(ctx context.Context, req *dto.TaskListRequest) ([]*dto.TaskResponse, int64, error) {
	tasks, total, err := s.taskRepo.List(ctx, req.Page, req.PageSize, req.Name, req.Enabled)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = s.toResponse(task)
	}

	return responses, total, nil
}

func (s *TaskService) toResponse(task *model.InspectionTask) *dto.TaskResponse {
	resp := &dto.TaskResponse{
		ID:            task.ID,
		Name:          task.Name,
		Description:   task.Description,
		CronExpr:      task.CronExpr,
		Status:        task.Status,
		Enabled:       task.Enabled,
		GroupIDs:      task.GroupIDs,
		ItemIDs:       task.ItemIDs,
		PushgatewayID: task.PushgatewayID,
		LastRunStatus: task.LastRunStatus,
		CreatedAt:     task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     task.UpdatedAt.Format(time.RFC3339),
	}

	if task.LastRunAt != nil {
		resp.LastRunAt = task.LastRunAt.Format(time.RFC3339)
	}
	if task.NextRunAt != nil {
		resp.NextRunAt = task.NextRunAt.Format(time.RFC3339)
	}

	return resp
}
