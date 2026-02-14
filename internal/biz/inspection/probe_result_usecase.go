package inspection

import "context"

// ProbeResultUseCase handles probe result business logic.
type ProbeResultUseCase struct {
	repo ProbeResultRepo
}

func NewProbeResultUseCase(repo ProbeResultRepo) *ProbeResultUseCase {
	return &ProbeResultUseCase{repo: repo}
}

func (uc *ProbeResultUseCase) Create(ctx context.Context, result *ProbeResult) error {
	return uc.repo.Create(ctx, result)
}

func (uc *ProbeResultUseCase) ListByTaskID(ctx context.Context, taskID uint, page, pageSize int) ([]*ProbeResult, int64, error) {
	return uc.repo.ListByTaskID(ctx, taskID, page, pageSize)
}
