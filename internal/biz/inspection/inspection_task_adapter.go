package inspection

import (
	"context"
	"time"

	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
)

// inspectionTaskRepoAdapter adapts inspection_mgmt TaskRepository to InspectionTaskRepo interface
type inspectionTaskRepoAdapter struct {
	repo inspectionmgmtdata.TaskRepository
}

// NewInspectionTaskRepoAdapter creates an adapter
func NewInspectionTaskRepoAdapter(repo inspectionmgmtdata.TaskRepository) InspectionTaskRepo {
	return &inspectionTaskRepoAdapter{repo: repo}
}

func (a *inspectionTaskRepoAdapter) GetByID(ctx context.Context, id uint) (*InspectionTaskV2, error) {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &InspectionTaskV2{
		ID:              task.ID,
		Name:            task.Name,
		TaskType:        task.TaskType,
		CronExpr:        task.CronExpr,
		Enabled:         task.Enabled,
		GroupIDs:        task.GroupIDs,
		ItemIDs:         task.ItemIDs,
		PushgatewayID:   task.PushgatewayID,
		Concurrency:     task.Concurrency,
		ExecutionMode:   task.ExecutionMode,
		AgentHostIDs:    task.AgentHostIDs,
		BusinessGroupID: task.BusinessGroupID,
		CustomVariables: task.CustomVariables,
	}, nil
}

func (a *inspectionTaskRepoAdapter) UpdateLastRun(ctx context.Context, id uint, status string) error {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	task.LastRunAt = &now
	task.LastRunStatus = status
	return a.repo.Update(ctx, task)
}
