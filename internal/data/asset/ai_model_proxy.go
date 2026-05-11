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
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"gorm.io/gorm"
)

type aiModelProxyRepo struct {
	db *gorm.DB
}

func NewAIModelProxyRepo(db *gorm.DB) asset.AIModelProxyRepo {
	return &aiModelProxyRepo{db: db}
}

func (r *aiModelProxyRepo) Create(proxy *asset.AIModelProxy, agentHostIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 创建代理
		if err := tx.Create(proxy).Error; err != nil {
			return err
		}

		// 创建Agent关联
		if len(agentHostIDs) > 0 {
			agents := make([]*asset.AIModelProxyAgent, len(agentHostIDs))
			for i, hostID := range agentHostIDs {
				agents[i] = &asset.AIModelProxyAgent{
					ProxyID: proxy.ID,
					HostID:  hostID,
				}
			}
			if err := tx.Create(&agents).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *aiModelProxyRepo) Update(proxy *asset.AIModelProxy, agentHostIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 更新代理
		if err := tx.Save(proxy).Error; err != nil {
			return err
		}

		// 删除旧的Agent关联
		if err := tx.Where("proxy_id = ?", proxy.ID).Delete(&asset.AIModelProxyAgent{}).Error; err != nil {
			return err
		}

		// 创建新的Agent关联
		if len(agentHostIDs) > 0 {
			agents := make([]*asset.AIModelProxyAgent, len(agentHostIDs))
			for i, hostID := range agentHostIDs {
				agents[i] = &asset.AIModelProxyAgent{
					ProxyID: proxy.ID,
					HostID:  hostID,
				}
			}
			if err := tx.Create(&agents).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *aiModelProxyRepo) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 删除代理
		if err := tx.Delete(&asset.AIModelProxy{}, id).Error; err != nil {
			return err
		}
		// 删除Agent关联（级联删除）
		if err := tx.Where("proxy_id = ?", id).Delete(&asset.AIModelProxyAgent{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *aiModelProxyRepo) GetByID(id uint) (*asset.AIModelProxy, error) {
	var proxy asset.AIModelProxy
	err := r.db.First(&proxy, id).Error
	if err != nil {
		return nil, err
	}
	return &proxy, nil
}

func (r *aiModelProxyRepo) GetByToken(token string) (*asset.AIModelProxy, error) {
	var proxy asset.AIModelProxy
	err := r.db.Where("proxy_token = ?", token).First(&proxy).Error
	if err != nil {
		return nil, err
	}
	return &proxy, nil
}

func (r *aiModelProxyRepo) List(page, pageSize int, groupID uint, status *int, keyword string) ([]*asset.AIModelProxy, int64, error) {
	var proxies []*asset.AIModelProxy
	var total int64

	query := r.db.Model(&asset.AIModelProxy{})

	// 关键词搜索
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ? OR target_url LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 分组过滤
	if groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	}

	// 状态过滤
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&proxies).Error; err != nil {
		return nil, 0, err
	}

	return proxies, total, nil
}

func (r *aiModelProxyRepo) GetAgentHostIDs(proxyID uint) ([]uint, error) {
	var agents []*asset.AIModelProxyAgent
	if err := r.db.Where("proxy_id = ?", proxyID).Find(&agents).Error; err != nil {
		return nil, err
	}

	hostIDs := make([]uint, len(agents))
	for i, agent := range agents {
		hostIDs[i] = agent.HostID
	}

	return hostIDs, nil
}

func (r *aiModelProxyRepo) RegenerateToken(id uint, newToken string) error {
	return r.db.Model(&asset.AIModelProxy{}).Where("id = ?", id).Update("proxy_token", newToken).Error
}
