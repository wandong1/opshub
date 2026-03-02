package asset

import (
	"context"
	"fmt"
)

// ServiceLabelUseCase 服务标签业务逻辑
type ServiceLabelUseCase struct {
	repo ServiceLabelRepo
}

// NewServiceLabelUseCase 创建服务标签UseCase
func NewServiceLabelUseCase(repo ServiceLabelRepo) *ServiceLabelUseCase {
	return &ServiceLabelUseCase{repo: repo}
}

// Create 创建服务标签
func (uc *ServiceLabelUseCase) Create(ctx context.Context, req *ServiceLabelRequest) (*ServiceLabelVO, error) {
	label := req.ToModel()
	if err := uc.repo.Create(ctx, label); err != nil {
		return nil, fmt.Errorf("创建服务标签失败: %w", err)
	}
	return uc.toVO(label), nil
}

// Update 更新服务标签
func (uc *ServiceLabelUseCase) Update(ctx context.Context, id uint, req *ServiceLabelRequest) (*ServiceLabelVO, error) {
	label, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("服务标签不存在: %w", err)
	}
	label.Name = req.Name
	label.MatchProcesses = req.MatchProcesses
	label.Description = req.Description
	label.Status = req.Status
	if err := uc.repo.Update(ctx, label); err != nil {
		return nil, fmt.Errorf("更新服务标签失败: %w", err)
	}
	return uc.toVO(label), nil
}

// Delete 删除服务标签
func (uc *ServiceLabelUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

// GetByID 获取服务标签详情
func (uc *ServiceLabelUseCase) GetByID(ctx context.Context, id uint) (*ServiceLabelVO, error) {
	label, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.toVO(label), nil
}

// List 列表查询
func (uc *ServiceLabelUseCase) List(ctx context.Context, page, pageSize int, keyword string) ([]*ServiceLabelVO, int64, error) {
	labels, total, err := uc.repo.List(ctx, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}
	vos := make([]*ServiceLabelVO, 0, len(labels))
	for _, l := range labels {
		vos = append(vos, uc.toVO(l))
	}
	return vos, total, nil
}

// GetAllEnabled 获取所有启用的标签
func (uc *ServiceLabelUseCase) GetAllEnabled(ctx context.Context) ([]*ServiceLabel, error) {
	return uc.repo.GetAllEnabled(ctx)
}

func (uc *ServiceLabelUseCase) toVO(label *ServiceLabel) *ServiceLabelVO {
	return &ServiceLabelVO{
		ID:             label.ID,
		Name:           label.Name,
		MatchProcesses: label.MatchProcesses,
		Description:    label.Description,
		Status:         int(label.Status),
		CreateTime:     label.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdateTime:     label.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
