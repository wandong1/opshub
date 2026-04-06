package alert

import (
	"context"
	"fmt"
	"strings"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

// fillGroupNames 批量回填 AssetGroupName 和 RuleGroupName
func (r *EventRepo) fillGroupNames(ctx context.Context, list []*biz.AlertEvent) {
	if len(list) == 0 {
		return
	}
	// 收集 ID 集合
	agIDs := map[uint]bool{}
	rgIDs := map[uint]bool{}
	for _, e := range list {
		if e.AssetGroupID > 0 {
			agIDs[e.AssetGroupID] = true
		}
		if e.AlertRuleID > 0 {
			rgIDs[e.AlertRuleID] = true
		}
	}
	// 查询 asset_group 名称
	agNames := map[uint]string{}
	if len(agIDs) > 0 {
		var ids []uint
		for id := range agIDs {
			ids = append(ids, id)
		}
		var rows []struct {
			ID   uint   `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		r.db.WithContext(ctx).Table("asset_group").Select("id, name").Where("id IN ?", ids).Scan(&rows)
		for _, row := range rows {
			agNames[row.ID] = row.Name
		}
	}
	// 查询 alert_rule 对应的 rule_group_id 和名称
	type ruleInfo struct {
		ID          uint   `gorm:"column:id"`
		RuleGroupID uint   `gorm:"column:rule_group_id"`
	}
	ruleGroupMap := map[uint]uint{} // ruleID -> ruleGroupID
	if len(rgIDs) > 0 {
		var ids []uint
		for id := range rgIDs {
			ids = append(ids, id)
		}
		var rules []ruleInfo
		r.db.WithContext(ctx).Table("alert_rules").Select("id, rule_group_id").Where("id IN ?", ids).Scan(&rules)
		for _, ru := range rules {
			if ru.RuleGroupID > 0 {
				ruleGroupMap[ru.ID] = ru.RuleGroupID
			}
		}
	}
	// 查询 rule_group 名称
	rgNames := map[uint]string{}
	if len(ruleGroupMap) > 0 {
		rgIDSet := map[uint]bool{}
		for _, rgID := range ruleGroupMap {
			rgIDSet[rgID] = true
		}
		var rgIDSlice []uint
		for id := range rgIDSet {
			rgIDSlice = append(rgIDSlice, id)
		}
		var rows []struct {
			ID   uint   `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		r.db.WithContext(ctx).Table("alert_rule_groups").Select("id, name").Where("id IN ?", rgIDSlice).Scan(&rows)
		for _, row := range rows {
			rgNames[row.ID] = row.Name
		}
	}
	// 回填
	for _, e := range list {
		if name, ok := agNames[e.AssetGroupID]; ok {
			e.AssetGroupName = name
		}
		if rgID, ok := ruleGroupMap[e.AlertRuleID]; ok {
			if name, ok := rgNames[rgID]; ok {
				e.RuleGroupName = name
			}
		}
	}
}

type EventRepo struct{ db *gorm.DB }

func NewEventRepo(db *gorm.DB) *EventRepo {
	return &EventRepo{db: db}
}

// GetDB 返回数据库连接（用于复杂查询）
func (r *EventRepo) GetDB() *gorm.DB {
	return r.db
}

func (r *EventRepo) Create(ctx context.Context, e *biz.AlertEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *EventRepo) Update(ctx context.Context, e *biz.AlertEvent) error {
	return r.db.WithContext(ctx).Model(e).Omit("created_at").Select("*").Updates(e).Error
}

func (r *EventRepo) GetByID(ctx context.Context, id uint) (*biz.AlertEvent, error) {
	var e biz.AlertEvent
	if err := r.db.WithContext(ctx).First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventRepo) GetFiringByFingerprint(ctx context.Context, fingerprint string) (*biz.AlertEvent, error) {
	var e biz.AlertEvent
	err := r.db.WithContext(ctx).Where("fingerprint = ? AND status = 'firing'", fingerprint).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventRepo) ListActive(ctx context.Context, page, pageSize int, assetGroupID uint, severity, keyword string) ([]*biz.AlertEvent, int64, error) {
	var list []*biz.AlertEvent
	var total int64
	q := r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("status = 'firing'")
	if assetGroupID > 0 {
		q = q.Where("asset_group_id = ?", assetGroupID)
	}
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if keyword != "" {
		q = q.Where("rule_name LIKE ?", "%"+keyword+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := q.Order("fired_at desc").Offset(offset).Limit(pageSize).Find(&list).Error
	if err == nil {
		r.fillGroupNames(ctx, list)
	}
	return list, total, err
}

func (r *EventRepo) ListHistory(ctx context.Context, page, pageSize int, assetGroupID uint, severity, status, resolveType, keyword string, startTime, endTime *time.Time) ([]*biz.AlertEvent, int64, error) {
	var list []*biz.AlertEvent
	var total int64
	q := r.db.WithContext(ctx).Model(&biz.AlertEvent{})
	if assetGroupID > 0 {
		q = q.Where("asset_group_id = ?", assetGroupID)
	}
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if resolveType != "" {
		q = q.Where("resolve_type = ?", resolveType)
	}
	if keyword != "" {
		q = q.Where("rule_name LIKE ?", "%"+keyword+"%")
	}
	if startTime != nil {
		q = q.Where("fired_at >= ?", *startTime)
	}
	if endTime != nil {
		q = q.Where("fired_at <= ?", *endTime)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := q.Order("fired_at desc").Offset(offset).Limit(pageSize).Find(&list).Error
	if err == nil {
		r.fillGroupNames(ctx, list)
	}
	return list, total, err
}

// TrendData 趋势数据点
type TrendData struct {
	Date        string `json:"date"`
	FiringCount int64  `json:"firingCount"`
	ResolvedCount int64 `json:"resolvedCount"`
}

func (r *EventRepo) GetTrend(ctx context.Context, days int) ([]*TrendData, error) {
	var result []*TrendData
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			DATE_FORMAT(fired_at, '%Y-%m-%d') as date,
			COUNT(*) as firing_count,
			SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END) as resolved_count
		FROM alert_events
		WHERE fired_at >= DATE_SUB(NOW(), INTERVAL ? DAY)
		GROUP BY DATE_FORMAT(fired_at, '%Y-%m-%d')
		ORDER BY date ASC
	`, days).Scan(&result).Error
	return result, err
}

// StatsData 统计数据
type StatsData struct {
	CriticalCount       int64 `json:"criticalCount"`
	MajorCount          int64 `json:"majorCount"`
	MinorCount          int64 `json:"minorCount"`
	WarningCount        int64 `json:"warningCount"`
	InfoCount           int64 `json:"infoCount"`
	AutoResolvedCount   int64 `json:"autoResolvedCount"`
	ManualResolvedCount int64 `json:"manualResolvedCount"`
	FiringCount         int64 `json:"firingCount"`
}

func (r *EventRepo) GetStats(ctx context.Context) (*StatsData, error) {
	var stats StatsData
	r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("status = 'firing' AND severity = 'critical'").Count(&stats.CriticalCount)
	r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("status = 'firing' AND severity = 'major'").Count(&stats.MajorCount)
	r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("status = 'firing' AND severity = 'minor'").Count(&stats.MinorCount)
	r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("status = 'firing' AND severity = 'warning'").Count(&stats.WarningCount)
	r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("status = 'firing' AND severity = 'info'").Count(&stats.InfoCount)
	r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("status = 'resolved' AND resolve_type = 'auto'").Count(&stats.AutoResolvedCount)
	r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("status = 'resolved' AND resolve_type = 'manual'").Count(&stats.ManualResolvedCount)
	r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("status = 'firing'").Count(&stats.FiringCount)
	return &stats, nil
}

// ListSilenced 查询已屏蔽告警列表
func (r *EventRepo) ListSilenced(ctx context.Context, page, pageSize int, severity, status, keyword, labelFilter string) ([]*biz.AlertEvent, int64, error) {
	var list []*biz.AlertEvent
	var total int64
	q := r.db.WithContext(ctx).Model(&biz.AlertEvent{}).Where("silenced = ?", true)

	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if keyword != "" {
		q = q.Where("rule_name LIKE ?", "%"+keyword+"%")
	}
	if labelFilter != "" {
		q = r.applyLabelFilter(q, labelFilter)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := q.Order("fired_at desc").Offset(offset).Limit(pageSize).Find(&list).Error
	if err == nil {
		r.fillGroupNames(ctx, list)
	}
	return list, total, err
}

// applyLabelFilter 应用标签过滤（支持模糊匹配）
func (r *EventRepo) applyLabelFilter(q *gorm.DB, filter string) *gorm.DB {
	// 解析 key=value 或 key=value* 格式
	parts := strings.SplitN(filter, "=", 2)
	if len(parts) != 2 {
		return q
	}

	key := strings.TrimSpace(parts[0])
	pattern := strings.TrimSpace(parts[1])

	// 使用 JSON_EXTRACT + LIKE 实现模糊匹配
	jsonPath := fmt.Sprintf("$.%s", key)

	if strings.HasSuffix(pattern, "*") {
		// 前缀匹配：job=prome*
		prefix := strings.TrimSuffix(pattern, "*")
		q = q.Where("JSON_EXTRACT(labels, ?) LIKE ?", jsonPath, prefix+"%")
	} else if strings.HasPrefix(pattern, "*") {
		// 后缀匹配：instance=*:9090
		suffix := strings.TrimPrefix(pattern, "*")
		q = q.Where("JSON_EXTRACT(labels, ?) LIKE ?", jsonPath, "%"+suffix)
	} else if strings.Contains(pattern, "*") {
		// 包含匹配：instance=*:90*
		pattern = strings.ReplaceAll(pattern, "*", "%")
		q = q.Where("JSON_EXTRACT(labels, ?) LIKE ?", jsonPath, pattern)
	} else {
		// 精确匹配
		q = q.Where("JSON_EXTRACT(labels, ?) = ?", jsonPath, pattern)
	}

	return q
}

// ResolveActiveByRuleID 将指定规则的所有告警中的告警标记为已恢复
func (r *EventRepo) ResolveActiveByRuleID(ctx context.Context, ruleID uint) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":       "resolved",
		"resolve_type": "manual_then_auto",
		"resolved_at":  now,
	}
	return r.db.WithContext(ctx).
		Model(&biz.AlertEvent{}).
		Where("alert_rule_id = ? AND status = 'firing'", ruleID).
		Updates(updates).Error
}
