package alert

import (
	"context"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type FingerprintRepo struct {
	db *gorm.DB
}

func NewFingerprintRepo(db *gorm.DB) *FingerprintRepo {
	return &FingerprintRepo{db: db}
}

func (r *FingerprintRepo) Create(ctx context.Context, fp *biz.AlertFingerprint) error {
	return r.db.WithContext(ctx).Create(fp).Error
}

func (r *FingerprintRepo) GetByFingerprint(ctx context.Context, subscriptionID uint, fingerprint string) (*biz.AlertFingerprint, error) {
	var fp biz.AlertFingerprint
	err := r.db.WithContext(ctx).
		Where("subscription_id = ? AND fingerprint = ?", subscriptionID, fingerprint).
		First(&fp).Error
	return &fp, err
}

func (r *FingerprintRepo) UpdateOccurrence(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&biz.AlertFingerprint{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_seen_at":     time.Now(),
			"occurrence_count": gorm.Expr("occurrence_count + 1"),
		}).Error
}

func (r *FingerprintRepo) UpdateSent(ctx context.Context, id uint, sentAt time.Time) error {
	return r.db.WithContext(ctx).Model(&biz.AlertFingerprint{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_seen_at": time.Now(),
			"last_sent_at": sentAt,
			"occurrence_count": gorm.Expr("occurrence_count + 1"),
		}).Error
}

// CleanupExpired 清理过期指纹记录（保留30天）
func (r *FingerprintRepo) CleanupExpired(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("last_seen_at < ?", before).
		Delete(&biz.AlertFingerprint{}).Error
}
