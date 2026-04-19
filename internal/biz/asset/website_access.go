// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package asset

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// WebsiteAccessSession 网站访问会话（短期凭证）
type WebsiteAccessSession struct {
	AccessKey      string    `json:"accessKey"`
	WebsiteID      uint      `json:"websiteId"`
	WebsiteName    string    `json:"websiteName"`
	UserID         uint      `json:"userId"`
	Username       string    `json:"username"`
	IssuedAt       time.Time `json:"issuedAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	SourceIP       string    `json:"sourceIp"`
	UserAgent      string    `json:"userAgent"`
}

// WebsiteAccessManager 网站访问管理器
type WebsiteAccessManager struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewWebsiteAccessManager 创建网站访问管理器
func NewWebsiteAccessManager(rdb *redis.Client) *WebsiteAccessManager {
	return &WebsiteAccessManager{
		rdb: rdb,
		ttl: 15 * time.Minute, // 默认 15 分钟
	}
}

// IssueAccessKey 签发访问凭证
func (m *WebsiteAccessManager) IssueAccessKey(ctx context.Context, websiteID uint, websiteName string, userID uint, username string, sourceIP string, userAgent string) (*WebsiteAccessSession, error) {
	accessKey := uuid.New().String()
	now := time.Now()

	session := &WebsiteAccessSession{
		AccessKey:   accessKey,
		WebsiteID:   websiteID,
		WebsiteName: websiteName,
		UserID:      userID,
		Username:    username,
		IssuedAt:    now,
		ExpiresAt:   now.Add(m.ttl),
		SourceIP:    sourceIP,
		UserAgent:   userAgent,
	}

	key := fmt.Sprintf("website:access:%s", accessKey)

	// 存储到 Redis
	err := m.rdb.HSet(ctx, key,
		"websiteId", websiteID,
		"websiteName", websiteName,
		"userId", userID,
		"username", username,
		"issuedAt", now.Unix(),
		"expiresAt", session.ExpiresAt.Unix(),
		"sourceIp", sourceIP,
		"userAgent", userAgent,
	).Err()

	if err != nil {
		return nil, fmt.Errorf("存储访问会话失败: %w", err)
	}

	// 设置过期时间
	if err := m.rdb.Expire(ctx, key, m.ttl).Err(); err != nil {
		return nil, fmt.Errorf("设置过期时间失败: %w", err)
	}

	return session, nil
}

// ValidateAccessKey 验证访问凭证
func (m *WebsiteAccessManager) ValidateAccessKey(ctx context.Context, accessKey string) (*WebsiteAccessSession, error) {
	key := fmt.Sprintf("website:access:%s", accessKey)

	result, err := m.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("查询访问会话失败: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("访问凭证无效或已过期")
	}

	// 解析会话数据
	session := &WebsiteAccessSession{
		AccessKey:   accessKey,
		WebsiteName: result["websiteName"],
		Username:    result["username"],
		SourceIP:    result["sourceIp"],
		UserAgent:   result["userAgent"],
	}

	// 解析数值字段
	if websiteID, ok := result["websiteId"]; ok {
		fmt.Sscanf(websiteID, "%d", &session.WebsiteID)
	}
	if userID, ok := result["userId"]; ok {
		fmt.Sscanf(userID, "%d", &session.UserID)
	}
	if issuedAt, ok := result["issuedAt"]; ok {
		var ts int64
		fmt.Sscanf(issuedAt, "%d", &ts)
		session.IssuedAt = time.Unix(ts, 0)
	}
	if expiresAt, ok := result["expiresAt"]; ok {
		var ts int64
		fmt.Sscanf(expiresAt, "%d", &ts)
		session.ExpiresAt = time.Unix(ts, 0)
	}

	return session, nil
}

// RevokeAccessKey 撤销访问凭证
func (m *WebsiteAccessManager) RevokeAccessKey(ctx context.Context, accessKey string) error {
	key := fmt.Sprintf("website:access:%s", accessKey)
	return m.rdb.Del(ctx, key).Err()
}
