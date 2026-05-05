package alert

import (
	"context"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type GroupCacheRepo struct {
	db *gorm.DB
}

func NewGroupCacheRepo(db *gorm.DB) *GroupCacheRepo {
	return &GroupCacheRepo{db: db}
}

func (r *GroupCacheRepo) Create(ctx context.Context, cache *biz.AlertGroupCache) error {
	return r.db.WithContext(ctx).Create(cache).Error
}

func (r *GroupCacheRepo) GetByID(ctx context.Context, id uint) (*biz.AlertGroupCache, error) {
	var cache biz.AlertGroupCache
	err := r.db.WithContext(ctx).First(&cache, id).Error
	return &cache, err
}

func (r *GroupCacheRepo) GetActiveCache(ctx context.Context, groupRuleID uint, groupKey string) (*biz.AlertGroupCache, error) {
	var cache biz.AlertGroupCache
	err := r.db.WithContext(ctx).
		Where("group_rule_id = ? AND group_key = ? AND sent = ?", groupRuleID, groupKey, false).
		First(&cache).Error
	return &cache, err
}

func (r *GroupCacheRepo) UpdateAlerts(ctx context.Context, id uint, alerts string, lastAlertAt time.Time, count int) error {
	return r.db.WithContext(ctx).Model(&biz.AlertGroupCache{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"alerts":        alerts,
			"last_alert_at": lastAlertAt,
			"alert_count":   count,
		}).Error
}

func (r *GroupCacheRepo) MarkSent(ctx context.Context, id uint, sentAt time.Time) error {
	return r.db.WithContext(ctx).Model(&biz.AlertGroupCache{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"sent":    true,
			"sent_at": sentAt,
		}).Error
}

// ListPendingCaches 查询所有未发送且超过 group_interval 的分组
func (r *GroupCacheRepo) ListPendingCaches(ctx context.Context, before time.Time) ([]*biz.AlertGroupCache, error) {
	var caches []*biz.AlertGroupCache
	err := r.db.WithContext(ctx).
		Where("sent = ? AND first_alert_at < ?", false, before).
		Find(&caches).Error
	return caches, err
}

// CleanupSent 清理已发送的分组缓存（保留7天）
func (r *GroupCacheRepo) CleanupSent(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("sent = ? AND sent_at < ?", true, before).
		Delete(&biz.AlertGroupCache{}).Error
}
