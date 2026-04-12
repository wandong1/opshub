package alert

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"github.com/ydcloud-dy/opshub/internal/cache"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
)

const (
	redisAlertStatePrefix = "alert:state:"
	redisAlertStateTTL    = 24 * time.Hour
	workerPoolSize        = 50
)

// alertState Redis 中存储的告警状态
type alertState struct {
	FiredAt     time.Time `json:"firedAt"`
	LastEvalAt  time.Time `json:"lastEvalAt"`
	Value       float64   `json:"value"`
	PendingSince time.Time `json:"pendingSince"` // 首次命中时间（用于 duration 判断）
}

// EvalEngine 告警评估引擎
type EvalEngine struct {
	db              *gorm.DB
	rdb             *redis.Client
	ruleRepo        *alertdata.RuleRepo
	dsRepo          *alertdata.DataSourceRepo
	eventRepo       *alertdata.EventRepo
	subRepo         *alertdata.SubscriptionRepo
	subRuleRepo     *alertdata.SubscriptionRuleRepo
	subChannelRepo  *alertdata.SubscriptionChannelRepo
	subUserRepo     *alertdata.SubscriptionUserRepo
	channelRepo     *alertdata.ChannelRepo
	silenceRuleRepo *alertdata.SilenceRuleRepo
	notifySvc       *NotifyService
	evalCache       *EvalCache       // 评估时间缓存
	ruleCache       *RuleCache       // 规则列表缓存
	silenceCache    *SilenceRuleCache // 屏蔽规则缓存
}
// NewEvalEngine 创建评估引擎
func NewEvalEngine(db *gorm.DB, rdb *redis.Client) *EvalEngine {
	dsRepo := alertdata.NewDataSourceRepo(db)
	ruleRepo := alertdata.NewRuleRepo(db)
	eventRepo := alertdata.NewEventRepo(db)
	subRepo := alertdata.NewSubscriptionRepo(db)
	subRuleRepo := alertdata.NewSubscriptionRuleRepo(db)
	subChannelRepo := alertdata.NewSubscriptionChannelRepo(db)
	subUserRepo := alertdata.NewSubscriptionUserRepo(db)
	channelRepo := alertdata.NewChannelRepo(db)
	silenceRuleRepo := alertdata.NewSilenceRuleRepo(db)
	notifySvc := NewNotifyService(channelRepo)

	// 创建评估时间缓存管理器
	evalCache := NewEvalCache(rdb, db, cache.NewLuaScripts())

	// 创建规则列表缓存管理器
	ruleCache := NewRuleCache(rdb, ruleRepo)

	// 创建屏蔽规则缓存管理器
	silenceCache := NewSilenceRuleCache(rdb, silenceRuleRepo)

	return &EvalEngine{
		db:              db,
		rdb:             rdb,
		ruleRepo:        ruleRepo,
		dsRepo:          dsRepo,
		eventRepo:       eventRepo,
		subRepo:         subRepo,
		subRuleRepo:     subRuleRepo,
		subChannelRepo:  subChannelRepo,
		subUserRepo:     subUserRepo,
		channelRepo:     channelRepo,
		silenceRuleRepo: silenceRuleRepo,
		notifySvc:       notifySvc,
		evalCache:       evalCache,
		ruleCache:       ruleCache,
		silenceCache:    silenceCache,
	}
}

// Start 启动告警评估引擎（阻塞，在 goroutine 中调用）
func (e *EvalEngine) Start(ctx context.Context) {
	appLogger.Info("告警评估引擎启动")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// worker channel
	workCh := make(chan *biz.AlertRule, 200)
	for i := 0; i < workerPoolSize; i++ {
		go e.worker(ctx, workCh)
	}

	// 启动批量同步 Worker
	go e.startSyncWorker(ctx)

	// 启动规则缓存管理器（订阅 Pub/Sub）
	go e.ruleCache.Start(ctx)

	// 启动屏蔽规则缓存管理器（订阅 Pub/Sub）
	go e.silenceCache.Start(ctx)

	// 记录每条规则上次入队时间
	lastQueued := make(map[uint]time.Time)

	for {
		select {
		case <-ctx.Done():
			e.ruleCache.Stop()
			e.silenceCache.Stop()
			appLogger.Info("告警评估引擎停止")
			return
		case <-ticker.C:
			// 从缓存读取规则列表（不再查询 MySQL）
			rules, err := e.ruleCache.GetEnabledRules(ctx)
			if err != nil {
				appLogger.Error("加载告警规则失败", zap.Error(err))
				continue
			}
			now := time.Now()
			for _, rule := range rules {
				interval := time.Duration(rule.EvalInterval) * time.Second
				if interval <= 0 {
					interval = 15 * time.Second
				}
				last, ok := lastQueued[rule.ID]
				if !ok || now.Sub(last) >= interval {
					lastQueued[rule.ID] = now
					select {
					case workCh <- rule:
					default: // channel 满时丢弃，避免堆积
					}
				}
			}
		}
	}
}

func (e *EvalEngine) worker(ctx context.Context, ch <-chan *biz.AlertRule) {
	for {
		select {
		case <-ctx.Done():
			return
		case rule := <-ch:
			e.evalRule(ctx, rule)
		}
	}
}

// startSyncWorker 启动批量同步 Worker
func (e *EvalEngine) startSyncWorker(ctx context.Context) {
	// 从配置读取同步间隔，默认 30 秒
	syncInterval := 30 * time.Second

	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	appLogger.Info("规则评估时间批量同步 Worker 已启动", zap.Duration("interval", syncInterval))

	for {
		select {
		case <-ctx.Done():
			appLogger.Info("规则评估时间批量同步 Worker 停止")
			return
		case <-ticker.C:
			if err := e.evalCache.SyncToMySQL(ctx); err != nil {
				appLogger.Error("批量同步评估时间失败", zap.Error(err))
			}
		}
	}
}

func (e *EvalEngine) evalRule(ctx context.Context, rule *biz.AlertRule) {
	defer func() {
		if r := recover(); r != nil {
			appLogger.Error("告警规则评估 panic", zap.Any("rule", rule.ID), zap.Any("recover", r))
		}
	}()

	// 异步更新评估时间到 Redis（不阻塞评估）
	_ = e.evalCache.UpdateEvalTime(ctx, rule.ID, time.Now())

	// 支持多数据源：优先使用 DataSourceIDs JSON 数组，降级到单个 DataSourceID
	var dsIDs []uint
	if rule.DataSourceIDs != "" {
		var ids []uint
		if err := json.Unmarshal([]byte(rule.DataSourceIDs), &ids); err == nil {
			for _, id := range ids {
				if id > 0 {
					dsIDs = append(dsIDs, id)
				}
			}
		}
	}
	if len(dsIDs) == 0 && rule.DataSourceID > 0 {
		dsIDs = []uint{rule.DataSourceID}
	}
	if len(dsIDs) == 0 {
		appLogger.Warn("规则无有效数据源，跳过评估", zap.Uint("ruleID", rule.ID))
		return
	}

	var queryErr bool
	var results []QueryResult
	for _, dsID := range dsIDs {
		ds, err := e.dsRepo.GetByID(ctx, dsID)
		if err != nil {
			appLogger.Warn("数据源不存在", zap.Uint("ruleID", rule.ID), zap.Uint("dsID", dsID), zap.Error(err))
			queryErr = true
			continue
		}
		r, err := QueryDataSource(ds, rule.Expr)
		if err != nil {
			appLogger.Warn("查询数据源失败", zap.Uint("ruleID", rule.ID), zap.Error(err))
			queryErr = true
			continue
		}
		results = append(results, r...)
	}
	// 如果所有数据源都查询失败，跳过本次评估（避免误报恢复）
	if queryErr && len(results) == 0 {
		return
	}

	// 收集本次命中的所有 fingerprints（同时保留 metric 标签和值）
	type hitInfo struct {
		Value  float64
		Labels map[string]string
	}
	hitFingerprints := make(map[string]hitInfo)
	for _, res := range results {
		fp := calcFingerprint(rule.ID, res.Labels)
		hitFingerprints[fp] = hitInfo{Value: res.Value, Labels: res.Labels}
	}

	// 处理命中的
	for fp, info := range hitFingerprints {
		e.handleFiring(ctx, rule, fp, info.Value, info.Labels)
	}

	// 处理恢复：从 Redis 中找到该规则下所有 firing 但本次未命中的
	// Bug 7 修复：使用 SCAN 代替 KEYS，避免阻塞 Redis
	pattern := fmt.Sprintf("%s%d:*", redisAlertStatePrefix, rule.ID)
	var cursor uint64
	var allKeys []string
	for {
		var keys []string
		var err error
		keys, cursor, err = e.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			appLogger.Warn("扫描 Redis key 失败", zap.Error(err))
			break
		}
		allKeys = append(allKeys, keys...)
		if cursor == 0 {
			break
		}
	}

	for _, key := range allKeys {
		fp := strings.TrimPrefix(key, redisAlertStatePrefix+fmt.Sprintf("%d:", rule.ID))
		if _, hit := hitFingerprints[fp]; !hit {
			e.handleResolved(ctx, rule, fp)
		}
	}
}

func (e *EvalEngine) handleFiring(ctx context.Context, rule *biz.AlertRule, fingerprint string, value float64, metricLabels map[string]string) {
	redisKey := fmt.Sprintf("%s%d:%s", redisAlertStatePrefix, rule.ID, fingerprint)
	now := time.Now()

	var state alertState
	val, err := e.rdb.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		// 首次命中
		state = alertState{PendingSince: now, LastEvalAt: now, Value: value}
		data, _ := json.Marshal(state)
		e.rdb.Set(ctx, redisKey, data, redisAlertStateTTL)
		// duration=0s 则立即触发，否则等待
		if rule.Duration == "" || rule.Duration == "0s" || rule.Duration == "0" {
			e.createFiringEvent(ctx, rule, fingerprint, value, now, metricLabels)
		}
		return
	} else if err != nil {
		return
	}

	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return
	}
	state.LastEvalAt = now
	state.Value = value

	// Bug 5 修复：检查 duration，解析失败时记录错误并跳过评估
	if rule.Duration != "" && rule.Duration != "0s" && rule.Duration != "0" {
		dur, err := parseDuration(rule.Duration)
		if err != nil {
			appLogger.Error("解析 duration 失败，跳过本次评估", zap.Uint("ruleID", rule.ID), zap.String("duration", rule.Duration), zap.Error(err))
			return
		}
		if now.Sub(state.PendingSince) < dur {
			// 未满持续时长，更新状态
			data, _ := json.Marshal(state)
			e.rdb.Set(ctx, redisKey, data, redisAlertStateTTL)
			return
		}
	}

	// Bug 3 修复：使用分布式锁避免重复创建事件
	lockKey := fmt.Sprintf("alert:lock:%s", fingerprint)
	acquired, err := e.rdb.SetNX(ctx, lockKey, "1", 5*time.Second).Result()
	if !acquired || err != nil {
		// 其他 worker 正在处理，跳过
		data, _ := json.Marshal(state)
		e.rdb.Set(ctx, redisKey, data, redisAlertStateTTL)
		return
	}
	defer e.rdb.Del(ctx, lockKey)

	// 已满足触发条件，检查 DB 中是否已有 firing 事件
	existing, err := e.eventRepo.GetFiringByFingerprint(ctx, fingerprint)
	if err != nil || existing == nil {
		// 未触发过，创建事件
		if state.FiredAt.IsZero() {
			state.FiredAt = now
		}
		e.createFiringEvent(ctx, rule, fingerprint, value, state.FiredAt, metricLabels)
	}

	data, _ := json.Marshal(state)
	e.rdb.Set(ctx, redisKey, data, redisAlertStateTTL)
}

// mergeLabels 合并规则自定义标签和 Prometheus metric 标签（metric 标签优先）
func mergeLabels(ruleLabelsJSON string, metricLabels map[string]string) string {
	merged := make(map[string]string)
	// 先放规则自定义标签
	if ruleLabelsJSON != "" {
		_ = json.Unmarshal([]byte(ruleLabelsJSON), &merged)
	}
	// metric 标签覆盖（优先级更高，包含 instance/job 等关键字段）
	for k, v := range metricLabels {
		merged[k] = v
	}
	if len(merged) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(merged)
	return string(b)
}

func (e *EvalEngine) createFiringEvent(ctx context.Context, rule *biz.AlertRule, fingerprint string, value float64, firedAt time.Time, metricLabels map[string]string) {
	event := &biz.AlertEvent{
		AlertRuleID:  rule.ID,
		RuleName:     rule.Name,
		AssetGroupID: rule.AssetGroupID,
		Fingerprint:  fingerprint,
		Severity:     rule.Severity,
		Status:       "firing",
		Labels:       mergeLabels(rule.Labels, metricLabels), // metric 标签 + 自定义标签合并
		Annotations:  rule.Annotations,
		Value:        value,
		FiredAt:      firedAt,
	}

	// 检查是否匹配屏蔽规则（修复：告警恢复后再次触发时应继承屏蔽状态）
	if matchedRule := e.findMatchingSilenceRule(ctx, event); matchedRule != nil {
		event.Silenced = true
		event.SilenceType = matchedRule.Type
		event.SilenceReason = matchedRule.Reason
		now := time.Now()
		event.SilencedAt = &now

		if matchedRule.Type == "fixed" {
			event.SilenceUntil = matchedRule.SilenceUntil
		} else if matchedRule.Type == "periodic" {
			event.SilenceTimeRanges = matchedRule.TimeRanges
		}

		appLogger.Info("新告警匹配到屏蔽规则，自动设置为屏蔽状态",
			zap.Uint("ruleID", rule.ID),
			zap.String("fingerprint", fingerprint),
			zap.Uint("silenceRuleID", matchedRule.ID),
			zap.String("silenceType", matchedRule.Type))
	}

	if err := e.eventRepo.Create(ctx, event); err != nil {
		appLogger.Error("创建告警事件失败", zap.Error(err))
		return
	}

	// 只有未屏蔽的告警才发送通知
	if !event.Silenced {
		e.sendNotifications(ctx, rule, event, false)
	}
}

func (e *EvalEngine) handleResolved(ctx context.Context, rule *biz.AlertRule, fingerprint string) {
	redisKey := fmt.Sprintf("%s%d:%s", redisAlertStatePrefix, rule.ID, fingerprint)

	var state alertState
	val, err := e.rdb.Get(ctx, redisKey).Result()
	if err != nil {
		return
	}
	_ = json.Unmarshal([]byte(val), &state)

	// 先查询活跃事件（Bug 4 修复：先查询数据库，成功后再删除 Redis）
	event, err := e.eventRepo.GetFiringByFingerprint(ctx, fingerprint)
	if err != nil || event == nil {
		return
	}

	now := time.Now()
	resolveVal := state.Value
	event.Status = "resolved"
	// 若人工介入过则标记为 manual_then_auto，否则 auto
	if event.ManualHandled {
		event.ResolveType = "manual_then_auto"
	} else {
		event.ResolveType = "auto"
	}
	event.ResolvedAt = &now
	event.ResolveValue = &resolveVal
	if err := e.eventRepo.Update(ctx, event); err != nil {
		appLogger.Error("更新告警恢复状态失败", zap.Error(err))
		return // 更新失败，保留 Redis 状态
	}

	// 更新成功后才删除 Redis key
	e.rdb.Del(ctx, redisKey)

	// Bug 8 修复：异步发送通知，避免阻塞
	if rule.NotifyOnResolve {
		go e.sendNotifications(ctx, rule, event, true)
	}
}

// parseSeverityList 解析 JSON 级别数组，空或解析失败返回 nil（表示全部级别）
func parseSeverityList(s string) []string {
	if s == "" || s == "[]" || s == "null" {
		return nil
	}
	var list []string
	if json.Unmarshal([]byte(s), &list) != nil {
		return nil
	}
	return list
}

// parseUintList 解析 JSON uint 数组
func parseUintList(s string) []uint {
	if s == "" || s == "[]" || s == "null" {
		return nil
	}
	var list []uint
	if json.Unmarshal([]byte(s), &list) != nil {
		return nil
	}
	return list
}

func (e *EvalEngine) sendNotifications(ctx context.Context, rule *biz.AlertRule, event *biz.AlertEvent, isResolve bool) {
	// 查找关联该规则的所有启用订阅（包括 rule_id=0 的全部规则订阅）
	subRules, err := e.subRuleRepo.ListByRuleID(ctx, rule.ID)
	if err != nil || len(subRules) == 0 {
		return
	}

	now := time.Now()

	// Bug 1 修复：在循环外部检查一次屏蔽状态，避免重复查询
	isSilenced := e.shouldSilence(ctx, event)

	for _, sr := range subRules {
		// 检查生效时间
		if !isInTimeRanges(sr.TimeRanges, now) {
			continue
		}
		// 检查订阅是否启用
		sub, err := e.subRepo.GetByID(ctx, sr.SubscriptionID)
		if err != nil || !sub.Enabled {
			continue
		}
		// 检查屏蔽状态（已在循环外部检查）
		if isSilenced {
			continue
		}
		// 检查告警级别过滤（空=全部级别）
		if sevList := parseSeverityList(sr.Severities); len(sevList) > 0 {
			matched := false
			for _, s := range sevList {
				if s == event.Severity {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		// 确定通道：优先使用规则行配置的通道，否则回退到订阅全局通道
		var channelIDs []uint
		if perRuleCh := parseUintList(sr.ChannelIDs); len(perRuleCh) > 0 {
			channelIDs = perRuleCh
		} else {
			subChannels, _ := e.subChannelRepo.ListBySubscription(ctx, sr.SubscriptionID)
			for _, sc := range subChannels {
				channelIDs = append(channelIDs, sc.ChannelID)
			}
		}
		// 确定接收用户手机号：优先使用规则行配置的用户，否则回退到订阅全局用户
		var phones []string
		if perRuleUsers := parseUintList(sr.UserIDs); len(perRuleUsers) > 0 {
			phones = e.getUserPhonesByIDs(ctx, perRuleUsers)
		} else {
			phones = e.getSubUserPhones(ctx, sr.SubscriptionID)
		}
		channels, _ := e.channelRepo.ListByIDs(ctx, channelIDs)
		for _, ch := range channels {
			if !ch.Enabled {
				continue
			}
			go e.notifySvc.Send(ctx, ch, event, isResolve, phones)
		}
	}
}

// getUserPhonesByIDs 通过用户 ID 列表直接查手机号（支持 ID=0 表示 @all）
func (e *EvalEngine) getUserPhonesByIDs(ctx context.Context, userIDs []uint) []string {
	for _, uid := range userIDs {
		if uid == 0 {
			return []string{} // 空=@all
		}
	}
	type phoneRow struct{ Phone string }
	var rows []phoneRow
	e.db.WithContext(ctx).Table("sys_users").Select("phone").Where("id IN ? AND phone != ''", userIDs).Scan(&rows)
	var phones []string
	for _, r := range rows {
		if r.Phone != "" {
			phones = append(phones, r.Phone)
		}
	}
	return phones
}

// getSubUserPhones 获取订阅的接收用户手机号列表；userID=0 表示所有人，返回空切片（调用方处理为@all）
func (e *EvalEngine) getSubUserPhones(ctx context.Context, subscriptionID uint) []string {
	subUsers, err := e.subUserRepo.ListBySubscription(ctx, subscriptionID)
	if err != nil || len(subUsers) == 0 {
		return []string{}
	}
	// 检查是否有 userID=0（所有人）
	for _, su := range subUsers {
		if su.UserID == 0 {
			return []string{} // 空=@all
		}
	}
	// 批量查手机号
	var userIDs []uint
	for _, su := range subUsers {
		userIDs = append(userIDs, su.UserID)
	}
	type phoneRow struct {
		Phone string
	}
	var rows []phoneRow
	e.db.WithContext(ctx).Table("sys_users").Select("phone").Where("id IN ? AND phone != ''", userIDs).Scan(&rows)
	var phones []string
	for _, r := range rows {
		if r.Phone != "" {
			phones = append(phones, r.Phone)
		}
	}
	return phones
}

// findMatchingSilenceRule 查找匹配的屏蔽规则（返回第一个匹配的规则）
func (e *EvalEngine) findMatchingSilenceRule(ctx context.Context, event *biz.AlertEvent) *biz.AlertSilenceRule {
	rules, err := e.silenceCache.GetActiveRules(ctx)
	if err != nil || len(rules) == 0 {
		return nil
	}

	now := time.Now()
	for _, rule := range rules {
		if e.matchSilenceRule(rule, event, now) {
			return rule
		}
	}
	return nil
}

// shouldSilence 检查告警是否应该被屏蔽
func (e *EvalEngine) shouldSilence(ctx context.Context, event *biz.AlertEvent) bool {
	now := time.Now()

	// 1. 检查单条手动屏蔽（现有逻辑，保持向后兼容）
	if event.Silenced && event.SilenceUntil != nil && now.Before(*event.SilenceUntil) {
		return true
	}

	// 2. 检查屏蔽规则（三元组匹配）
	// Bug 2 修复：从缓存读取屏蔽规则，避免每次查询数据库
	rules, err := e.silenceCache.GetActiveRules(ctx)
	if err != nil || len(rules) == 0 {
		return false
	}

	for _, rule := range rules {
		if e.matchSilenceRule(rule, event, now) {
			return true
		}
	}

	return false
}

// matchSilenceRule 检查告警是否匹配屏蔽规则
func (e *EvalEngine) matchSilenceRule(rule *biz.AlertSilenceRule, event *biz.AlertEvent, now time.Time) bool {
	// 1. 检查告警等级
	if rule.Severity != event.Severity {
		return false
	}

	// 2. 检查规则名称
	if rule.RuleName != event.RuleName {
		return false
	}

	// 3. 检查标签（子集匹配）
	if !MatchLabels(event.Labels, rule.Labels) {
		return false
	}

	// 4. 检查时效
	if rule.Type == "fixed" {
		// 固定时长：检查是否在屏蔽期内
		if rule.SilenceUntil != nil && now.Before(*rule.SilenceUntil) {
			return true
		}
	} else if rule.Type == "periodic" {
		// 周期性：检查当前时间是否在时间窗口内
		if isInTimeRanges(rule.TimeRanges, now) {
			return true
		}
	}

	return false
}

// calcFingerprint 计算告警指纹
func calcFingerprint(ruleID uint, labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteString(strconv.FormatUint(uint64(ruleID), 10))
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(labels[k])
	}
	h := sha256.Sum256([]byte(sb.String()))
	return fmt.Sprintf("%x", h[:8])
}

// parseDuration 解析 "5m", "1h", "30s" 等格式
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "m") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		return time.Duration(n) * time.Minute, err
	}
	if strings.HasSuffix(s, "h") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "h"))
		return time.Duration(n) * time.Hour, err
	}
	if strings.HasSuffix(s, "s") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "s"))
		return time.Duration(n) * time.Second, err
	}
	return time.ParseDuration(s)
}

// TimeRange 生效时间段
type TimeRange struct {
	Weekdays []int  `json:"weekdays"` // 1=周一 ... 7=周日
	Start    string `json:"start"`    // "08:00"
	End      string `json:"end"`      // "18:00"
}

// isInTimeRanges 检查当前时间是否在任一生效时间段内
func isInTimeRanges(timeRangesJSON string, now time.Time) bool {
	if timeRangesJSON == "" || timeRangesJSON == "[]" || timeRangesJSON == "null" {
		return true // 空=全天生效
	}
	var ranges []TimeRange
	if err := json.Unmarshal([]byte(timeRangesJSON), &ranges); err != nil || len(ranges) == 0 {
		return true
	}
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日=7
	}
	currentTime := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
	for _, r := range ranges {
		for _, wd := range r.Weekdays {
			if wd == weekday {
				if currentTime >= r.Start && currentTime <= r.End {
					return true
				}
			}
		}
	}
	return false
}

// EvalRuleOnce 临时测试：立即执行一次查询（供 HTTP handler 调用）
func (e *EvalEngine) EvalRuleOnce(ctx context.Context, rule *biz.AlertRule) ([]QueryResult, error) {
	dsIDs := []uint{rule.DataSourceID}
	if rule.DataSourceIDs != "" {
		var ids []uint
		if err := json.Unmarshal([]byte(rule.DataSourceIDs), &ids); err == nil && len(ids) > 0 {
			dsIDs = ids
		}
	}
	var allResults []QueryResult
	for _, dsID := range dsIDs {
		if dsID == 0 {
			continue
		}
		ds, err := e.dsRepo.GetByID(ctx, dsID)
		if err != nil {
			continue
		}
		r, err := QueryDataSource(ds, rule.Expr)
		if err != nil {
			continue
		}
		allResults = append(allResults, r...)
	}
	return allResults, nil
}

// EvalExprOnDatasources 对指定数据源列表执行临时 PromQL 查询（Ad-hoc 测试）
func (e *EvalEngine) EvalExprOnDatasources(ctx context.Context, dsIDs []uint, expr string) ([]QueryResult, error) {
	var allResults []QueryResult
	for _, dsID := range dsIDs {
		if dsID == 0 {
			continue
		}
		ds, err := e.dsRepo.GetByID(ctx, dsID)
		if err != nil {
			continue
		}
		r, err := QueryDataSource(ds, expr)
		if err != nil {
			return nil, fmt.Errorf("数据源 %s 查询失败: %w", ds.Name, err)
		}
		allResults = append(allResults, r...)
	}
	return allResults, nil
}

// GetEvalCache 获取评估缓存（供外部调用）
func (e *EvalEngine) GetEvalCache() *EvalCache {
	return e.evalCache
}

// GetRuleCache 获取规则缓存（供外部调用）
func (e *EvalEngine) GetRuleCache() *RuleCache {
	return e.ruleCache
}

// GetSilenceCache 获取屏蔽规则缓存（供外部调用）
func (e *EvalEngine) GetSilenceCache() *SilenceRuleCache {
	return e.silenceCache
}


