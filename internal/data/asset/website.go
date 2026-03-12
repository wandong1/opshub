// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package asset

import (
	"context"

	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"gorm.io/gorm"
)

type websiteRepo struct {
	db *gorm.DB
}

func NewWebsiteRepo(db *gorm.DB) asset.WebsiteRepo {
	return &websiteRepo{db: db}
}

func (r *websiteRepo) Create(ctx context.Context, website *asset.Website) error {
	return r.db.WithContext(ctx).Create(website).Error
}

func (r *websiteRepo) Update(ctx context.Context, website *asset.Website) error {
	return r.db.WithContext(ctx).Save(website).Error
}

func (r *websiteRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除站点
		if err := tx.Delete(&asset.Website{}, id).Error; err != nil {
			return err
		}
		// 删除分组关联
		if err := tx.Where("website_id = ?", id).Delete(&asset.WebsiteGroup{}).Error; err != nil {
			return err
		}
		// 删除Agent关联
		if err := tx.Where("website_id = ?", id).Delete(&asset.WebsiteAgent{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *websiteRepo) GetByID(ctx context.Context, id uint) (*asset.Website, error) {
	var website asset.Website
	err := r.db.WithContext(ctx).First(&website, id).Error
	if err != nil {
		return nil, err
	}
	return &website, nil
}

func (r *websiteRepo) List(ctx context.Context, page, pageSize int, keyword string, groupIDs []uint, siteType string) ([]*asset.Website, int64, error) {
	var websites []*asset.Website
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.Website{})

	// 关键词搜索
	if keyword != "" {
		query = query.Where("name LIKE ? OR url LIKE ? OR description LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 站点类型过滤
	if siteType != "" {
		query = query.Where("type = ?", siteType)
	}

	// 分组过滤
	if len(groupIDs) > 0 {
		query = query.Joins("JOIN website_groups ON websites.id = website_groups.website_id").
			Where("website_groups.group_id IN ?", groupIDs).
			Distinct()
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&websites).Error; err != nil {
		return nil, 0, err
	}

	return websites, total, nil
}

func (r *websiteRepo) GetAll(ctx context.Context) ([]*asset.Website, error) {
	var websites []*asset.Website
	err := r.db.WithContext(ctx).Find(&websites).Error
	return websites, err
}

// AddGroups 添加分组关联
func (r *websiteRepo) AddGroups(ctx context.Context, websiteID uint, groupIDs []uint) error {
	if len(groupIDs) == 0 {
		return nil
	}

	var groups []asset.WebsiteGroup
	for _, groupID := range groupIDs {
		groups = append(groups, asset.WebsiteGroup{
			WebsiteID: websiteID,
			GroupID:   groupID,
		})
	}

	return r.db.WithContext(ctx).Create(&groups).Error
}

// RemoveGroups 删除分组关联
func (r *websiteRepo) RemoveGroups(ctx context.Context, websiteID uint) error {
	return r.db.WithContext(ctx).Where("website_id = ?", websiteID).Delete(&asset.WebsiteGroup{}).Error
}

// GetGroupIDs 获取站点关联的分组ID列表
func (r *websiteRepo) GetGroupIDs(ctx context.Context, websiteID uint) ([]uint, error) {
	var groups []asset.WebsiteGroup
	err := r.db.WithContext(ctx).Where("website_id = ?", websiteID).Find(&groups).Error
	if err != nil {
		return nil, err
	}

	var groupIDs []uint
	for _, g := range groups {
		groupIDs = append(groupIDs, g.GroupID)
	}
	return groupIDs, nil
}

// AddAgents 添加Agent关联
func (r *websiteRepo) AddAgents(ctx context.Context, websiteID uint, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	var agents []asset.WebsiteAgent
	for _, hostID := range hostIDs {
		agents = append(agents, asset.WebsiteAgent{
			WebsiteID: websiteID,
			HostID:    hostID,
		})
	}

	return r.db.WithContext(ctx).Create(&agents).Error
}

// RemoveAgents 删除Agent关联
func (r *websiteRepo) RemoveAgents(ctx context.Context, websiteID uint) error {
	return r.db.WithContext(ctx).Where("website_id = ?", websiteID).Delete(&asset.WebsiteAgent{}).Error
}

// GetAgentHostIDs 获取站点关联的Agent主机ID列表
func (r *websiteRepo) GetAgentHostIDs(ctx context.Context, websiteID uint) ([]uint, error) {
	var agents []asset.WebsiteAgent
	err := r.db.WithContext(ctx).Where("website_id = ?", websiteID).Find(&agents).Error
	if err != nil {
		return nil, err
	}

	var hostIDs []uint
	for _, a := range agents {
		hostIDs = append(hostIDs, a.HostID)
	}
	return hostIDs, nil
}
