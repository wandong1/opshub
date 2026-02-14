package inspection

import "context"

// ProbeTaskUseCase handles probe task business logic.
type ProbeTaskUseCase struct {
	repo       ProbeTaskRepo
	configRepo ProbeConfigRepo
}

func NewProbeTaskUseCase(repo ProbeTaskRepo, configRepo ProbeConfigRepo) *ProbeTaskUseCase {
	return &ProbeTaskUseCase{repo: repo, configRepo: configRepo}
}

func (uc *ProbeTaskUseCase) Create(ctx context.Context, task *ProbeTask) error {
	return uc.repo.Create(ctx, task)
}

func (uc *ProbeTaskUseCase) Update(ctx context.Context, task *ProbeTask) error {
	return uc.repo.Update(ctx, task)
}

func (uc *ProbeTaskUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *ProbeTaskUseCase) GetByID(ctx context.Context, id uint) (*ProbeTask, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *ProbeTaskUseCase) List(ctx context.Context, page, pageSize int, keyword string, status *int8) ([]*ProbeTask, int64, error) {
	return uc.repo.List(ctx, page, pageSize, keyword, status)
}

func (uc *ProbeTaskUseCase) GetEnabled(ctx context.Context) ([]*ProbeTask, error) {
	return uc.repo.GetEnabled(ctx)
}

func (uc *ProbeTaskUseCase) UpdateLastRun(ctx context.Context, id uint, result string) error {
	return uc.repo.UpdateLastRun(ctx, id, result)
}

// CreateWithConfigs creates a task and sets its associated config IDs.
func (uc *ProbeTaskUseCase) CreateWithConfigs(ctx context.Context, task *ProbeTask, configIDs []uint) error {
	if err := uc.repo.Create(ctx, task); err != nil {
		return err
	}
	return uc.repo.SetConfigs(ctx, task.ID, configIDs)
}

// UpdateWithConfigs updates a task and replaces its associated config IDs.
func (uc *ProbeTaskUseCase) UpdateWithConfigs(ctx context.Context, task *ProbeTask, configIDs []uint) error {
	if err := uc.repo.Update(ctx, task); err != nil {
		return err
	}
	return uc.repo.SetConfigs(ctx, task.ID, configIDs)
}

func (uc *ProbeTaskUseCase) GetConfigIDs(ctx context.Context, taskID uint) ([]uint, error) {
	return uc.repo.GetConfigIDs(ctx, taskID)
}

func (uc *ProbeTaskUseCase) GetConfigsByTaskID(ctx context.Context, taskID uint) ([]*ProbeConfig, error) {
	return uc.repo.GetConfigsByTaskID(ctx, taskID)
}

func (uc *ProbeTaskUseCase) BatchGetConfigIDs(ctx context.Context, taskIDs []uint) (map[uint][]uint, error) {
	return uc.repo.BatchGetConfigIDs(ctx, taskIDs)
}
