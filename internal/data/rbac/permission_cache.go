package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	bizrbac "github.com/ydcloud-dy/opshub/internal/biz/rbac"
)

const (
	userPermsKeyPrefix = "user:perms:"
	userIsAdminPrefix  = "user:is_admin:"
	registeredAPIsKey  = "sys:registered_apis"
	userPermsTTL       = 5 * time.Minute
	adminTTL           = 5 * time.Minute
	registeredAPIsTTL  = 10 * time.Minute
)

// PermissionCache Redis权限缓存
type PermissionCache struct {
	client *redis.Client
}

// NewPermissionCache 创建权限缓存
func NewPermissionCache(client *redis.Client) *PermissionCache {
	return &PermissionCache{client: client}
}

// GetUserPermissions 获取用户API权限（缓存）
func (c *PermissionCache) GetUserPermissions(ctx context.Context, userID uint) ([]bizrbac.APIPermission, error) {
	key := fmt.Sprintf("%s%d", userPermsKeyPrefix, userID)
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var perms []bizrbac.APIPermission
	if err := json.Unmarshal([]byte(val), &perms); err != nil {
		return nil, err
	}
	return perms, nil
}

// SetUserPermissions 缓存用户API权限
func (c *PermissionCache) SetUserPermissions(ctx context.Context, userID uint, perms []bizrbac.APIPermission) error {
	key := fmt.Sprintf("%s%d", userPermsKeyPrefix, userID)
	data, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, string(data), userPermsTTL).Err()
}

// GetUserIsAdmin 获取用户是否admin（缓存）
func (c *PermissionCache) GetUserIsAdmin(ctx context.Context, userID uint) (bool, bool) {
	key := fmt.Sprintf("%s%d", userIsAdminPrefix, userID)
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return false, false
	}
	return val == "1", true
}

// SetUserIsAdmin 缓存用户是否admin
func (c *PermissionCache) SetUserIsAdmin(ctx context.Context, userID uint, isAdmin bool) error {
	key := fmt.Sprintf("%s%d", userIsAdminPrefix, userID)
	val := "0"
	if isAdmin {
		val = "1"
	}
	return c.client.Set(ctx, key, val, adminTTL).Err()
}

// GetRegisteredAPIs 获取已注册API列表（缓存）
func (c *PermissionCache) GetRegisteredAPIs(ctx context.Context) ([]bizrbac.APIPermission, error) {
	val, err := c.client.Get(ctx, registeredAPIsKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var perms []bizrbac.APIPermission
	if err := json.Unmarshal([]byte(val), &perms); err != nil {
		return nil, err
	}
	return perms, nil
}

// SetRegisteredAPIs 缓存已注册API列表
func (c *PermissionCache) SetRegisteredAPIs(ctx context.Context, perms []bizrbac.APIPermission) error {
	data, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, registeredAPIsKey, string(data), registeredAPIsTTL).Err()
}

// ClearUserPermissions 清除指定用户的权限缓存
func (c *PermissionCache) ClearUserPermissions(ctx context.Context, userID uint) {
	c.client.Del(ctx, fmt.Sprintf("%s%d", userPermsKeyPrefix, userID))
	c.client.Del(ctx, fmt.Sprintf("%s%d", userIsAdminPrefix, userID))
}

// ClearAllUserPermissions 清除所有用户权限缓存
func (c *PermissionCache) ClearAllUserPermissions(ctx context.Context) {
	c.clearByPattern(ctx, userPermsKeyPrefix+"*")
	c.clearByPattern(ctx, userIsAdminPrefix+"*")
}

// ClearRegisteredAPIs 清除已注册API缓存
func (c *PermissionCache) ClearRegisteredAPIs(ctx context.Context) {
	c.client.Del(ctx, registeredAPIsKey)
}

func (c *PermissionCache) clearByPattern(ctx context.Context, pattern string) {
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		c.client.Del(ctx, iter.Val())
	}
}
