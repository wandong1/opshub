package inspection

import "context"

// ProbeConfigRepo defines the data access interface for ProbeConfig.
type ProbeConfigRepo interface {
	Create(ctx context.Context, config *ProbeConfig) error
	Update(ctx context.Context, config *ProbeConfig) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*ProbeConfig, error)
	List(ctx context.Context, page, pageSize int, keyword, probeType, category string, groupID uint, status *int8) ([]*ProbeConfig, int64, error)
	ListAll(ctx context.Context) ([]*ProbeConfig, error)
	BatchCreate(ctx context.Context, configs []*ProbeConfig) error
}

// ProbeTaskRepo defines the data access interface for ProbeTask.
type ProbeTaskRepo interface {
	Create(ctx context.Context, task *ProbeTask) error
	Update(ctx context.Context, task *ProbeTask) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*ProbeTask, error)
	List(ctx context.Context, page, pageSize int, keyword string, status *int8) ([]*ProbeTask, int64, error)
	GetEnabled(ctx context.Context) ([]*ProbeTask, error)
	UpdateLastRun(ctx context.Context, id uint, result string) error
	SetConfigs(ctx context.Context, taskID uint, configIDs []uint) error
	GetConfigIDs(ctx context.Context, taskID uint) ([]uint, error)
	GetConfigsByTaskID(ctx context.Context, taskID uint) ([]*ProbeConfig, error)
	BatchGetConfigIDs(ctx context.Context, taskIDs []uint) (map[uint][]uint, error)
}

// ProbeResultRepo defines the data access interface for ProbeResult.
type ProbeResultRepo interface {
	Create(ctx context.Context, result *ProbeResult) error
	ListByTaskID(ctx context.Context, taskID uint, page, pageSize int) ([]*ProbeResult, int64, error)
	CleanupByTaskID(ctx context.Context, taskID uint, keepCount int) error
}

// PushgatewayConfigRepo defines the data access interface for PushgatewayConfig.
type PushgatewayConfigRepo interface {
	Create(ctx context.Context, config *PushgatewayConfig) error
	Update(ctx context.Context, config *PushgatewayConfig) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*PushgatewayConfig, error)
	List(ctx context.Context) ([]*PushgatewayConfig, error)
}

// ProbeVariableRepo defines the data access interface for ProbeVariable.
type ProbeVariableRepo interface {
	Create(ctx context.Context, v *ProbeVariable) error
	Update(ctx context.Context, v *ProbeVariable) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*ProbeVariable, error)
	List(ctx context.Context, page, pageSize int, keyword, varType, groupIDs string) ([]*ProbeVariable, int64, error)
	GetByNames(ctx context.Context, names []string, allowedGroupIDs []uint) ([]*ProbeVariable, error)
}
