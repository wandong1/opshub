package inspection

import "context"

// PushgatewayUseCase handles Pushgateway config business logic.
type PushgatewayUseCase struct {
	repo PushgatewayConfigRepo
}

func NewPushgatewayUseCase(repo PushgatewayConfigRepo) *PushgatewayUseCase {
	return &PushgatewayUseCase{repo: repo}
}

func (uc *PushgatewayUseCase) Create(ctx context.Context, config *PushgatewayConfig) error {
	return uc.repo.Create(ctx, config)
}

func (uc *PushgatewayUseCase) Update(ctx context.Context, config *PushgatewayConfig) error {
	return uc.repo.Update(ctx, config)
}

func (uc *PushgatewayUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *PushgatewayUseCase) GetByID(ctx context.Context, id uint) (*PushgatewayConfig, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *PushgatewayUseCase) List(ctx context.Context) ([]*PushgatewayConfig, error) {
	return uc.repo.List(ctx)
}
