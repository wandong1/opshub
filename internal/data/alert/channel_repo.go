package alert

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type ChannelRepo struct{ db *gorm.DB }

func NewChannelRepo(db *gorm.DB) *ChannelRepo {
	return &ChannelRepo{db: db}
}

func (r *ChannelRepo) Create(ctx context.Context, ch *biz.AlertNotifyChannel) error {
	return r.db.WithContext(ctx).Create(ch).Error
}

func (r *ChannelRepo) Update(ctx context.Context, ch *biz.AlertNotifyChannel) error {
	return r.db.WithContext(ctx).Model(ch).Omit("created_at").Select("*").Updates(ch).Error
}

func (r *ChannelRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertNotifyChannel{}, id).Error
}

func (r *ChannelRepo) GetByID(ctx context.Context, id uint) (*biz.AlertNotifyChannel, error) {
	var ch biz.AlertNotifyChannel
	if err := r.db.WithContext(ctx).First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *ChannelRepo) List(ctx context.Context) ([]*biz.AlertNotifyChannel, error) {
	var list []*biz.AlertNotifyChannel
	return list, r.db.WithContext(ctx).Order("id asc").Find(&list).Error
}

func (r *ChannelRepo) ListEnabled(ctx context.Context) ([]*biz.AlertNotifyChannel, error) {
	var list []*biz.AlertNotifyChannel
	return list, r.db.WithContext(ctx).Where("enabled = true").Order("id asc").Find(&list).Error
}

func (r *ChannelRepo) ListByIDs(ctx context.Context, ids []uint) ([]*biz.AlertNotifyChannel, error) {
	var list []*biz.AlertNotifyChannel
	return list, r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
}
