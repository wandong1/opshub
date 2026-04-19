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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"
)

type WebsiteUseCase struct {
	websiteRepo     WebsiteRepo
	assetGroupRepo  AssetGroupRepo
	hostRepo        HostRepo
	encryptionKey   []byte
	accessManager   *WebsiteAccessManager
}

func NewWebsiteUseCase(websiteRepo WebsiteRepo, assetGroupRepo AssetGroupRepo, hostRepo HostRepo) *WebsiteUseCase {
	// AES-256要求密钥长度必须是32字节（256位）
	encryptionKey := []byte("opshub-enc-key-32-bytes-long!!!!")
	return &WebsiteUseCase{
		websiteRepo:    websiteRepo,
		assetGroupRepo: assetGroupRepo,
		hostRepo:       hostRepo,
		encryptionKey:  encryptionKey,
		accessManager:  nil, // 延迟注入
	}
}

// SetAccessManager 设置访问管理器（依赖注入）
func (uc *WebsiteUseCase) SetAccessManager(manager *WebsiteAccessManager) {
	uc.accessManager = manager
}

// encrypt 加密
func (uc *WebsiteUseCase) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(uc.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 解密
func (uc *WebsiteUseCase) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(uc.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// Create 创建站点
func (uc *WebsiteUseCase) Create(ctx context.Context, req *WebsiteRequest) error {
	// 验证内部站点必须绑定Agent
	if req.Type == "internal" && len(req.AgentHostIDs) == 0 {
		return fmt.Errorf("内部站点必须绑定至少1台Agent主机")
	}

	website := req.ToModel()

	// 生成代理访问 Token（UUID）
	website.ProxyToken = generateProxyToken()

	// 加密敏感信息
	if website.AccessPassword != "" {
		encrypted, err := uc.encrypt(website.AccessPassword)
		if err != nil {
			return fmt.Errorf("加密访问密码失败: %w", err)
		}
		website.AccessPassword = encrypted
	}

	if website.Credential != "" {
		encrypted, err := uc.encrypt(website.Credential)
		if err != nil {
			return fmt.Errorf("加密凭据失败: %w", err)
		}
		website.Credential = encrypted
	}

	// 创建站点
	if err := uc.websiteRepo.Create(ctx, website); err != nil {
		return err
	}

	// 添加分组关联
	if len(req.GroupIDs) > 0 {
		if err := uc.websiteRepo.AddGroups(ctx, website.ID, req.GroupIDs); err != nil {
			return err
		}
	}

	// 添加Agent关联（仅内部站点）
	if req.Type == "internal" && len(req.AgentHostIDs) > 0 {
		if err := uc.websiteRepo.AddAgents(ctx, website.ID, req.AgentHostIDs); err != nil {
			return err
		}
	}

	return nil
}

// Update 更新站点
func (uc *WebsiteUseCase) Update(ctx context.Context, id uint, req *WebsiteRequest) error {
	// 验证内部站点必须绑定Agent
	if req.Type == "internal" && len(req.AgentHostIDs) == 0 {
		return fmt.Errorf("内部站点必须绑定至少1台Agent主机")
	}

	website, err := uc.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 更新基本信息
	website.Name = req.Name
	website.URL = req.URL
	website.Icon = req.Icon
	website.Type = req.Type
	website.SecureCopyURL = req.SecureCopyURL
	website.AccessUser = req.AccessUser
	website.Description = req.Description
	website.Status = req.Status

	// 加密敏感信息（只有明确提供了新密码才更新）
	if req.AccessPassword != "" {
		encrypted, err := uc.encrypt(req.AccessPassword)
		if err != nil {
			return fmt.Errorf("加密访问密码失败: %w", err)
		}
		website.AccessPassword = encrypted
	}
	// 注意：如果 req.AccessPassword 为空，保留原有密码不变

	if req.Credential != "" {
		encrypted, err := uc.encrypt(req.Credential)
		if err != nil {
			return fmt.Errorf("加密凭据失败: %w", err)
		}
		website.Credential = encrypted
	}

	// 更新站点
	if err := uc.websiteRepo.Update(ctx, website); err != nil {
		return err
	}

	// 更新分组关联
	if err := uc.websiteRepo.RemoveGroups(ctx, id); err != nil {
		return err
	}
	if len(req.GroupIDs) > 0 {
		if err := uc.websiteRepo.AddGroups(ctx, id, req.GroupIDs); err != nil {
			return err
		}
	}

	// 更新Agent关联
	if err := uc.websiteRepo.RemoveAgents(ctx, id); err != nil {
		return err
	}
	if req.Type == "internal" && len(req.AgentHostIDs) > 0 {
		if err := uc.websiteRepo.AddAgents(ctx, id, req.AgentHostIDs); err != nil {
			return err
		}
	}

	return nil
}

// Delete 删除站点
func (uc *WebsiteUseCase) Delete(ctx context.Context, id uint) error {
	return uc.websiteRepo.Delete(ctx, id)
}

// GetByID 获取站点详情
func (uc *WebsiteUseCase) GetByID(ctx context.Context, id uint) (*WebsiteVO, error) {
	website, err := uc.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 获取详情时返回完整信息（包括密码）
	return uc.toVOWithPassword(ctx, website)
}

// List 获取站点列表
func (uc *WebsiteUseCase) List(ctx context.Context, page, pageSize int, keyword string, groupIDs []uint, siteType string) ([]*WebsiteVO, int64, error) {
	websites, total, err := uc.websiteRepo.List(ctx, page, pageSize, keyword, groupIDs, siteType)
	if err != nil {
		return nil, 0, err
	}

	var vos []*WebsiteVO
	for _, website := range websites {
		vo, err := uc.toVO(ctx, website)
		if err != nil {
			return nil, 0, err
		}
		vos = append(vos, vo)
	}

	return vos, total, nil
}

// toVO 转换为VO
func (uc *WebsiteUseCase) toVO(ctx context.Context, website *Website) (*WebsiteVO, error) {
	vo := &WebsiteVO{
		ID:            website.ID,
		Name:          website.Name,
		URL:           website.URL,
		Icon:          website.Icon,
		Type:          website.Type,
		SecureCopyURL: website.SecureCopyURL,
		AccessUser:    website.AccessUser,
		Description:   website.Description,
		Status:        website.Status,
		CreateTime:    website.CreatedAt.Format(time.DateTime),
		UpdateTime:    website.UpdatedAt.Format(time.DateTime),

		// 代理配置
		ProxyStrategy:  website.ProxyStrategy,
		ProxyWhitelist: website.ProxyWhitelist,
		ProxyBlacklist: website.ProxyBlacklist,
		InjectScript:   website.InjectScript,
		RewriteHTML:    website.RewriteHTML,
		RewriteCSS:     website.RewriteCSS,
		RewriteJS:      website.RewriteJS,

		// 代理访问 Token
		ProxyToken: website.ProxyToken,
		ProxyURL:   fmt.Sprintf("/api/v1/websites/proxy/t/%s", website.ProxyToken),
	}

	// 类型文本
	if website.Type == "external" {
		vo.TypeText = "外部站点"
	} else {
		vo.TypeText = "内部站点"
	}

	// 状态文本
	if website.Status == 1 {
		vo.StatusText = "启用"
	} else {
		vo.StatusText = "禁用"
	}

	// 解密凭据（仅返回是否存在）
	if website.Credential != "" {
		vo.Credential = "******"
	}

	// 获取分组信息
	groupIDs, err := uc.websiteRepo.GetGroupIDs(ctx, website.ID)
	if err == nil && len(groupIDs) > 0 {
		vo.GroupIDs = groupIDs
		var groupNames []string
		for _, gid := range groupIDs {
			group, err := uc.assetGroupRepo.GetByID(ctx, gid)
			if err == nil {
				groupNames = append(groupNames, group.Name)
			}
		}
		vo.GroupNames = groupNames
	}

	// 获取Agent主机信息
	if website.Type == "internal" {
		hostIDs, err := uc.websiteRepo.GetAgentHostIDs(ctx, website.ID)
		if err == nil && len(hostIDs) > 0 {
			vo.AgentHostIDs = hostIDs
			var hostNames []string
			for _, hid := range hostIDs {
				host, err := uc.hostRepo.GetByID(ctx, hid)
				if err == nil {
					hostNames = append(hostNames, host.Name)
					// 检查是否有在线的Agent
					if host.AgentStatus == "online" {
						vo.AgentOnline = true
					}
				}
			}
			vo.AgentHostNames = hostNames
		}
	}

	return vo, nil
}

// toVOWithPassword 转换为VO（包含密码）
func (uc *WebsiteUseCase) toVOWithPassword(ctx context.Context, website *Website) (*WebsiteVO, error) {
	vo, err := uc.toVO(ctx, website)
	if err != nil {
		return nil, err
	}

	// 解密密码
	if website.AccessPassword != "" {
		decryptedPassword, err := uc.decrypt(website.AccessPassword)
		if err != nil {
			// 解密失败时记录错误，但不中断流程
			fmt.Printf("解密密码失败: %v, 原始密文: %s\n", err, website.AccessPassword)
		} else {
			vo.AccessPassword = decryptedPassword
		}
	}

	return vo, nil
}

// generateProxyToken 生成代理访问 Token（UUID）
func generateProxyToken() string {
	// 使用 UUID v4 生成随机 token
	// 格式：xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx（去掉连字符）
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// 如果随机数生成失败，使用时间戳作为后备
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// 设置 UUID 版本（v4）和变体
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 10

	// 转换为十六进制字符串（无连字符）
	return fmt.Sprintf("%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x",
		uuid[0], uuid[1], uuid[2], uuid[3],
		uuid[4], uuid[5], uuid[6], uuid[7],
		uuid[8], uuid[9], uuid[10], uuid[11],
		uuid[12], uuid[13], uuid[14], uuid[15])
}

// RegenerateProxyToken 重新生成代理访问 Token
func (uc *WebsiteUseCase) RegenerateProxyToken(ctx context.Context, id uint) (string, error) {
	website, err := uc.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	// 生成新的 token
	newToken := generateProxyToken()
	website.ProxyToken = newToken

	// 更新数据库
	if err := uc.websiteRepo.Update(ctx, website); err != nil {
		return "", err
	}

	return newToken, nil
}

// GetByProxyToken 通过代理 Token 获取站点
func (uc *WebsiteUseCase) GetByProxyToken(ctx context.Context, token string) (*WebsiteVO, error) {
	website, err := uc.websiteRepo.GetByProxyToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return uc.toVO(ctx, website)
}
