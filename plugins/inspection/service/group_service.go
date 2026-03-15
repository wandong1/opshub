package service

import (
	"context"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/inspection/dto"
	"github.com/ydcloud-dy/opshub/plugins/inspection/model"
	"github.com/ydcloud-dy/opshub/plugins/inspection/repository"
)

type GroupService struct {
	groupRepo repository.GroupRepository
}

func NewGroupService(groupRepo repository.GroupRepository) *GroupService {
	return &GroupService{
		groupRepo: groupRepo,
	}
}

func (s *GroupService) Create(ctx context.Context, req *dto.GroupCreateRequest) error {
	group := &model.InspectionGroup{
		Name:               req.Name,
		Description:        req.Description,
		Status:             req.Status,
		Sort:               req.Sort,
		PrometheusURL:      req.PrometheusURL,
		PrometheusUsername: req.PrometheusUsername,
		PrometheusPassword: req.PrometheusPassword,
		ExecutionMode:      req.ExecutionMode,
		GroupIDs:           req.GroupIDs,
	}

	if group.Status == "" {
		group.Status = "enabled"
	}
	if group.ExecutionMode == "" {
		group.ExecutionMode = "auto"
	}

	return s.groupRepo.Create(ctx, group)
}

func (s *GroupService) Update(ctx context.Context, id uint, req *dto.GroupUpdateRequest) error {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Description != "" {
		group.Description = req.Description
	}
	if req.Status != "" {
		group.Status = req.Status
	}
	group.Sort = req.Sort
	if req.PrometheusURL != "" {
		group.PrometheusURL = req.PrometheusURL
	}
	group.PrometheusUsername = req.PrometheusUsername
	if req.PrometheusPassword != "" {
		group.PrometheusPassword = req.PrometheusPassword
	}
	if req.ExecutionMode != "" {
		group.ExecutionMode = req.ExecutionMode
	}
	if req.GroupIDs != "" {
		group.GroupIDs = req.GroupIDs
	}

	return s.groupRepo.Update(ctx, group)
}

func (s *GroupService) Delete(ctx context.Context, id uint) error {
	return s.groupRepo.Delete(ctx, id)
}

func (s *GroupService) GetByID(ctx context.Context, id uint) (*dto.GroupResponse, error) {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(group), nil
}

func (s *GroupService) List(ctx context.Context, req *dto.GroupListRequest) ([]*dto.GroupResponse, int64, error) {
	groups, total, err := s.groupRepo.List(ctx, req.Page, req.PageSize, req.Name, req.Status)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.GroupResponse, len(groups))
	for i, group := range groups {
		responses[i] = s.toResponse(group)
	}

	return responses, total, nil
}

func (s *GroupService) GetAll(ctx context.Context) ([]*dto.GroupResponse, error) {
	groups, err := s.groupRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.GroupResponse, len(groups))
	for i, group := range groups {
		responses[i] = s.toResponse(group)
	}

	return responses, nil
}

func (s *GroupService) toResponse(group *model.InspectionGroup) *dto.GroupResponse {
	return &dto.GroupResponse{
		ID:                 group.ID,
		Name:               group.Name,
		Description:        group.Description,
		Status:             group.Status,
		Sort:               group.Sort,
		PrometheusURL:      group.PrometheusURL,
		PrometheusUsername: group.PrometheusUsername,
		ExecutionMode:      group.ExecutionMode,
		GroupIDs:           group.GroupIDs,
		CreatedAt:          group.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          group.UpdatedAt.Format(time.RFC3339),
	}
}

// GetStats 获取统计数据
func (s *GroupService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats, err := s.groupRepo.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
