package inspection

import "context"

// ProbeConfigUseCase handles probe config business logic.
type ProbeConfigUseCase struct {
	repo ProbeConfigRepo
}

func NewProbeConfigUseCase(repo ProbeConfigRepo) *ProbeConfigUseCase {
	return &ProbeConfigUseCase{repo: repo}
}

func (uc *ProbeConfigUseCase) Create(ctx context.Context, config *ProbeConfig) error {
	return uc.repo.Create(ctx, config)
}

func (uc *ProbeConfigUseCase) Update(ctx context.Context, config *ProbeConfig) error {
	return uc.repo.Update(ctx, config)
}

func (uc *ProbeConfigUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *ProbeConfigUseCase) GetByID(ctx context.Context, id uint) (*ProbeConfig, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *ProbeConfigUseCase) List(ctx context.Context, page, pageSize int, keyword, probeType, category string, groupID uint, status *int8) ([]*ProbeConfig, int64, error) {
	return uc.repo.List(ctx, page, pageSize, keyword, probeType, category, groupID, status)
}

func (uc *ProbeConfigUseCase) ListAll(ctx context.Context) ([]*ProbeConfig, error) {
	return uc.repo.ListAll(ctx)
}

func (uc *ProbeConfigUseCase) BatchCreate(ctx context.Context, configs []*ProbeConfig) error {
	return uc.repo.BatchCreate(ctx, configs)
}
