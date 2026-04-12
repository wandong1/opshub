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

package system

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// APIKeyRepoInterface API Key 仓库接口
type APIKeyRepoInterface interface {
	Create(ctx context.Context, apiKey *SysAPIKey) error
	GetByID(ctx context.Context, id uint) (*SysAPIKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*SysAPIKey, error)
	List(ctx context.Context, page, pageSize int) ([]*SysAPIKey, int64, error)
	Delete(ctx context.Context, id uint) error
	GetAll(ctx context.Context) ([]*SysAPIKey, error)
}

// APIKeyUseCase API Key 用例
type APIKeyUseCase struct {
	apiKeyRepo APIKeyRepoInterface
	rdb        *redis.Client
}

// NewAPIKeyUseCase 创建 API Key 用例
func NewAPIKeyUseCase(apiKeyRepo APIKeyRepoInterface, rdb *redis.Client) *APIKeyUseCase {
	return &APIKeyUseCase{
		apiKeyRepo: apiKeyRepo,
		rdb:        rdb,
	}
}

// GenerateAPIKey 生成随机 API Key（格式：opshub_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx，40字符）
func (uc *APIKeyUseCase) GenerateAPIKey() (string, error) {
	// 生成32字节随机数
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// 转换为十六进制字符串（64字符）
	hexString := hex.EncodeToString(randomBytes)

	// 添加前缀，截取前32位（总长度40字符：opshub_ + 32字符）
	apiKey := "opshub_" + hexString[:32]
	return apiKey, nil
}

// HashAPIKey 对 API Key 进行 SHA256 哈希
func (uc *APIKeyUseCase) HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// MaskAPIKey 脱敏 API Key（显示前缀+***+后缀）
func (uc *APIKeyUseCase) MaskAPIKey(prefix, suffix string) string {
	return prefix + "***" + suffix
}

// CreateAPIKey 创建 API Key
func (uc *APIKeyUseCase) CreateAPIKey(ctx context.Context, name, description string) (*CreateAPIKeyResponse, error) {
	// 生成随机 API Key
	apiKey, err := uc.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("生成 API Key 失败: %w", err)
	}

	// 计算哈希值
	keyHash := uc.HashAPIKey(apiKey)

	// 提取前缀和后缀（用于脱敏展示）
	keyPrefix := apiKey[:10]  // opshub_xxx
	keySuffix := apiKey[len(apiKey)-4:] // 最后4位

	// 创建数据库记录
	sysAPIKey := &SysAPIKey{
		Name:        name,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		KeySuffix:   keySuffix,
		Description: description,
	}

	if err := uc.apiKeyRepo.Create(ctx, sysAPIKey); err != nil {
		return nil, fmt.Errorf("保存 API Key 失败: %w", err)
	}

	// 返回完整明文密钥（仅此一次）
	return &CreateAPIKeyResponse{
		ID:          sysAPIKey.ID,
		Name:        sysAPIKey.Name,
		APIKey:      apiKey, // 完整明文
		Description: sysAPIKey.Description,
		CreatedAt:   sysAPIKey.CreatedAt,
	}, nil
}

// ListAPIKeys 获取 API Key 列表（带统计信息）
func (uc *APIKeyUseCase) ListAPIKeys(ctx context.Context, page, pageSize int) ([]*APIKeyVO, int64, error) {
	// 从数据库获取 API Key 列表
	apiKeys, total, err := uc.apiKeyRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 构建 VO 列表，从 Redis 读取统计信息
	vos := make([]*APIKeyVO, 0, len(apiKeys))
	for _, key := range apiKeys {
		vo := &APIKeyVO{
			ID:          key.ID,
			Name:        key.Name,
			MaskedKey:   uc.MaskAPIKey(key.KeyPrefix, key.KeySuffix),
			Description: key.Description,
			CreatedAt:   key.CreatedAt,
		}

		// 从 Redis 读取调用次数
		callCountKey := RedisKeyAPIKeyCallCount + key.KeyHash
		callCountStr, _ := uc.rdb.Get(ctx, callCountKey).Result()
		if callCountStr != "" {
			if count, err := strconv.ParseInt(callCountStr, 10, 64); err == nil {
				vo.TotalCalls = count
			}
		}

		// 从 Redis 读取最后调用时间
		lastCalledKey := RedisKeyAPIKeyLastCalled + key.KeyHash
		lastCalledStr, _ := uc.rdb.Get(ctx, lastCalledKey).Result()
		if lastCalledStr != "" {
			if timestamp, err := strconv.ParseInt(lastCalledStr, 10, 64); err == nil {
				vo.LastCalledAt = time.Unix(timestamp, 0)
			}
		}

		vos = append(vos, vo)
	}

	return vos, total, nil
}

// DeleteAPIKey 删除 API Key（同时清理 Redis 统计数据）
func (uc *APIKeyUseCase) DeleteAPIKey(ctx context.Context, id uint) error {
	// 先获取 API Key 信息（用于清理 Redis）
	apiKey, err := uc.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("API Key 不存在: %w", err)
	}

	// 删除数据库记录
	if err := uc.apiKeyRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除 API Key 失败: %w", err)
	}

	// 清理 Redis 统计数据
	callCountKey := RedisKeyAPIKeyCallCount + apiKey.KeyHash
	lastCalledKey := RedisKeyAPIKeyLastCalled + apiKey.KeyHash
	uc.rdb.Del(ctx, callCountKey, lastCalledKey)

	return nil
}

// VerifyAPIKey 验证 API Key 是否有效
func (uc *APIKeyUseCase) VerifyAPIKey(ctx context.Context, apiKey string) (*SysAPIKey, error) {
	// 计算哈希值
	keyHash := uc.HashAPIKey(apiKey)

	// 从数据库查询
	sysAPIKey, err := uc.apiKeyRepo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("API Key 无效或已删除")
	}

	return sysAPIKey, nil
}

// RecordAPIKeyUsage 记录 API Key 使用（增加调用次数，更新最后调用时间）
func (uc *APIKeyUseCase) RecordAPIKeyUsage(ctx context.Context, keyHash string) error {
	// 增加调用次数
	callCountKey := RedisKeyAPIKeyCallCount + keyHash
	uc.rdb.Incr(ctx, callCountKey)

	// 更新最后调用时间（Unix 时间戳）
	lastCalledKey := RedisKeyAPIKeyLastCalled + keyHash
	uc.rdb.Set(ctx, lastCalledKey, time.Now().Unix(), 0)

	return nil
}
