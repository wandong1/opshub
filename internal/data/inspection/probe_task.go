package inspection

import (
	"context"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"gorm.io/gorm"
)

type probeTaskRepo struct{ db *gorm.DB }

func NewProbeTaskRepo(db *gorm.DB) biz.ProbeTaskRepo {
	return &probeTaskRepo{db: db}
}

func (r *probeTaskRepo) Create(ctx context.Context, task *biz.ProbeTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *probeTaskRepo) Update(ctx context.Context, task *biz.ProbeTask) error {
	return r.db.WithContext(ctx).Model(task).Omit("created_at").Select("*").Updates(task).Error
}

func (r *probeTaskRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("probe_task_id = ?", id).Delete(&biz.ProbeTaskConfig{}).Error; err != nil {
			return err
		}
		return tx.Delete(&biz.ProbeTask{}, id).Error
	})
}

func (r *probeTaskRepo) GetByID(ctx context.Context, id uint) (*biz.ProbeTask, error) {
	var task biz.ProbeTask
	if err := r.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *probeTaskRepo) List(ctx context.Context, page, pageSize int, keyword string, status *int8) ([]*biz.ProbeTask, int64, error) {
	var tasks []*biz.ProbeTask
	var total int64

	query := r.db.WithContext(ctx).Model(&biz.ProbeTask{})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (r *probeTaskRepo) GetEnabled(ctx context.Context) ([]*biz.ProbeTask, error) {
	var tasks []*biz.ProbeTask
	if err := r.db.WithContext(ctx).Where("status = ?", 1).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *probeTaskRepo) UpdateLastRun(ctx context.Context, id uint, result string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&biz.ProbeTask{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_run_at": &now,
			"last_result": result,
		}).Error
}

func (r *probeTaskRepo) SetConfigs(ctx context.Context, taskID uint, configIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("probe_task_id = ?", taskID).Delete(&biz.ProbeTaskConfig{}).Error; err != nil {
			return err
		}
		if len(configIDs) == 0 {
			return nil
		}
		records := make([]biz.ProbeTaskConfig, len(configIDs))
		for i, cid := range configIDs {
			records[i] = biz.ProbeTaskConfig{ProbeTaskID: taskID, ProbeConfigID: cid}
		}
		return tx.Create(&records).Error
	})
}

func (r *probeTaskRepo) GetConfigIDs(ctx context.Context, taskID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.WithContext(ctx).Model(&biz.ProbeTaskConfig{}).
		Where("probe_task_id = ?", taskID).Pluck("probe_config_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *probeTaskRepo) GetConfigsByTaskID(ctx context.Context, taskID uint) ([]*biz.ProbeConfig, error) {
	var configs []*biz.ProbeConfig
	if err := r.db.WithContext(ctx).
		Joins("JOIN probe_task_configs ON probe_task_configs.probe_config_id = probe_configs.id").
		Where("probe_task_configs.probe_task_id = ?", taskID).
		Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *probeTaskRepo) BatchGetConfigIDs(ctx context.Context, taskIDs []uint) (map[uint][]uint, error) {
	if len(taskIDs) == 0 {
		return make(map[uint][]uint), nil
	}
	var records []biz.ProbeTaskConfig
	if err := r.db.WithContext(ctx).Where("probe_task_id IN ?", taskIDs).Find(&records).Error; err != nil {
		return nil, err
	}
	result := make(map[uint][]uint)
	for _, r := range records {
		result[r.ProbeTaskID] = append(result[r.ProbeTaskID], r.ProbeConfigID)
	}
	return result, nil
}
