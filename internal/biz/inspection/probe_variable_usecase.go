package inspection

import "context"

// ProbeVariableUseCase handles probe variable business logic.
type ProbeVariableUseCase struct {
	repo ProbeVariableRepo
}

func NewProbeVariableUseCase(repo ProbeVariableRepo) *ProbeVariableUseCase {
	return &ProbeVariableUseCase{repo: repo}
}

func (uc *ProbeVariableUseCase) Create(ctx context.Context, v *ProbeVariable) error {
	return uc.repo.Create(ctx, v)
}

func (uc *ProbeVariableUseCase) Update(ctx context.Context, v *ProbeVariable) error {
	return uc.repo.Update(ctx, v)
}

func (uc *ProbeVariableUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *ProbeVariableUseCase) GetByID(ctx context.Context, id uint) (*ProbeVariable, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *ProbeVariableUseCase) List(ctx context.Context, page, pageSize int, keyword, varType, groupIDs string) ([]*ProbeVariable, int64, error) {
	return uc.repo.List(ctx, page, pageSize, keyword, varType, groupIDs)
}

func (uc *ProbeVariableUseCase) GetByNames(ctx context.Context, names []string, allowedGroupIDs []uint) ([]*ProbeVariable, error) {
	return uc.repo.GetByNames(ctx, names, allowedGroupIDs)
}
