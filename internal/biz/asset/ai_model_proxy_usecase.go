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
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIModelProxyUseCase struct {
	proxyRepo      AIModelProxyRepo
	assetGroupRepo AssetGroupRepo
	hostRepo       HostRepo
	encryptionKey  []byte
}

func NewAIModelProxyUseCase(proxyRepo AIModelProxyRepo, assetGroupRepo AssetGroupRepo, hostRepo HostRepo) *AIModelProxyUseCase {
	// AES-256要求密钥长度必须是32字节（256位）
	encryptionKey := []byte("opshub-enc-key-32-bytes-long!!!!")
	return &AIModelProxyUseCase{
		proxyRepo:      proxyRepo,
		assetGroupRepo: assetGroupRepo,
		hostRepo:       hostRepo,
		encryptionKey:  encryptionKey,
	}
}

// encrypt 加密
func (uc *AIModelProxyUseCase) encrypt(plaintext string) (string, error) {
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
func (uc *AIModelProxyUseCase) decrypt(ciphertext string) (string, error) {
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
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// Create 创建AI模型代理
func (uc *AIModelProxyUseCase) Create(req *AIModelProxyRequest) (*AIModelProxyVO, error) {
	ctx := context.Background()

	// 验证分组是否存在
	if _, err := uc.assetGroupRepo.GetByID(ctx, req.GroupID); err != nil {
		return nil, fmt.Errorf("资产分组不存在")
	}

	// 验证Agent主机是否存在
	for _, hostID := range req.AgentHostIDs {
		if _, err := uc.hostRepo.GetByID(ctx, hostID); err != nil {
			return nil, fmt.Errorf("主机ID %d 不存在", hostID)
		}
	}

	// 转换为模型
	proxy := req.ToModel()

	// 加密API密钥
	if proxy.APIKey != "" {
		encrypted, err := uc.encrypt(proxy.APIKey)
		if err != nil {
			return nil, fmt.Errorf("加密API密钥失败: %v", err)
		}
		proxy.APIKey = encrypted
	}

	// 生成永久Token
	proxy.ProxyToken = uuid.New().String()

	// 创建代理
	if err := uc.proxyRepo.Create(proxy, req.AgentHostIDs); err != nil {
		return nil, fmt.Errorf("创建AI模型代理失败: %v", err)
	}

	// 返回VO
	return uc.toVO(proxy, req.AgentHostIDs)
}

// Update 更新AI模型代理
func (uc *AIModelProxyUseCase) Update(req *AIModelProxyRequest) (*AIModelProxyVO, error) {
	ctx := context.Background()

	// 获取现有代理
	existingProxy, err := uc.proxyRepo.GetByID(req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("AI模型代理不存在")
		}
		return nil, err
	}

	// 验证分组是否存在
	if _, err := uc.assetGroupRepo.GetByID(ctx, req.GroupID); err != nil {
		return nil, fmt.Errorf("资产分组不存在")
	}

	// 验证Agent主机是否存在
	for _, hostID := range req.AgentHostIDs {
		if _, err := uc.hostRepo.GetByID(ctx, hostID); err != nil {
			return nil, fmt.Errorf("主机ID %d 不存在", hostID)
		}
	}

	// 更新字段
	existingProxy.Name = req.Name
	existingProxy.Description = req.Description
	existingProxy.ModelType = req.ModelType
	existingProxy.Status = req.Status
	existingProxy.TargetURL = req.TargetURL
	existingProxy.GroupID = req.GroupID

	// 更新超时时间
	if req.Timeout > 0 {
		existingProxy.Timeout = req.Timeout
	}

	// 更新API密钥（如果提供了新的）
	if req.APIKey != "" {
		encrypted, err := uc.encrypt(req.APIKey)
		if err != nil {
			return nil, fmt.Errorf("加密API密钥失败: %v", err)
		}
		existingProxy.APIKey = encrypted
	}

	// 更新代理
	if err := uc.proxyRepo.Update(existingProxy, req.AgentHostIDs); err != nil {
		return nil, fmt.Errorf("更新AI模型代理失败: %v", err)
	}

	// 返回VO
	return uc.toVO(existingProxy, req.AgentHostIDs)
}

// Delete 删除AI模型代理
func (uc *AIModelProxyUseCase) Delete(id uint) error {
	// 检查是否存在
	if _, err := uc.proxyRepo.GetByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("AI模型代理不存在")
		}
		return err
	}

	return uc.proxyRepo.Delete(id)
}

// GetByID 根据ID获取AI模型代理
func (uc *AIModelProxyUseCase) GetByID(id uint) (*AIModelProxyVO, error) {
	proxy, err := uc.proxyRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("AI模型代理不存在")
		}
		return nil, err
	}

	// 获取Agent主机ID列表
	agentHostIDs, err := uc.proxyRepo.GetAgentHostIDs(proxy.ID)
	if err != nil {
		return nil, err
	}

	return uc.toVO(proxy, agentHostIDs)
}

// GetByToken 根据Token获取AI模型代理
func (uc *AIModelProxyUseCase) GetByToken(token string) (*AIModelProxy, []uint, error) {
	proxy, err := uc.proxyRepo.GetByToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("无效的代理Token")
		}
		return nil, nil, err
	}

	// 检查状态
	if proxy.Status != 1 {
		return nil, nil, fmt.Errorf("AI模型代理已禁用")
	}

	// 获取Agent主机ID列表
	agentHostIDs, err := uc.proxyRepo.GetAgentHostIDs(proxy.ID)
	if err != nil {
		return nil, nil, err
	}

	// 解密API密钥
	if proxy.APIKey != "" {
		decrypted, err := uc.decrypt(proxy.APIKey)
		if err != nil {
			return nil, nil, fmt.Errorf("解密API密钥失败: %v", err)
		}
		proxy.APIKey = decrypted
	}

	return proxy, agentHostIDs, nil
}

// List 获取AI模型代理列表
func (uc *AIModelProxyUseCase) List(page, pageSize int, groupID uint, status *int, keyword string) ([]*AIModelProxyVO, int64, error) {
	proxies, total, err := uc.proxyRepo.List(page, pageSize, groupID, status, keyword)
	if err != nil {
		return nil, 0, err
	}

	vos := make([]*AIModelProxyVO, len(proxies))
	for i, proxy := range proxies {
		// 获取Agent主机ID列表
		agentHostIDs, err := uc.proxyRepo.GetAgentHostIDs(proxy.ID)
		if err != nil {
			return nil, 0, err
		}

		vo, err := uc.toVO(proxy, agentHostIDs)
		if err != nil {
			return nil, 0, err
		}
		vos[i] = vo
	}

	return vos, total, nil
}

// RegenerateToken 重新生成Token
func (uc *AIModelProxyUseCase) RegenerateToken(id uint) (*AIModelProxyVO, error) {
	// 检查是否存在
	proxy, err := uc.proxyRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("AI模型代理不存在")
		}
		return nil, err
	}

	// 生成新Token
	newToken := uuid.New().String()
	if err := uc.proxyRepo.RegenerateToken(id, newToken); err != nil {
		return nil, fmt.Errorf("重新生成Token失败: %v", err)
	}

	// 更新内存中的Token
	proxy.ProxyToken = newToken

	// 获取Agent主机ID列表
	agentHostIDs, err := uc.proxyRepo.GetAgentHostIDs(proxy.ID)
	if err != nil {
		return nil, err
	}

	return uc.toVO(proxy, agentHostIDs)
}

// toVO 转换为VO
func (uc *AIModelProxyUseCase) toVO(proxy *AIModelProxy, agentHostIDs []uint) (*AIModelProxyVO, error) {
	ctx := context.Background()

	vo := &AIModelProxyVO{
		ID:           proxy.ID,
		Name:         proxy.Name,
		Description:  proxy.Description,
		ModelType:    proxy.ModelType,
		Status:       proxy.Status,
		TargetURL:    proxy.TargetURL,
		Timeout:      proxy.Timeout,
		ProxyToken:   proxy.ProxyToken,
		GroupID:      proxy.GroupID,
		AgentHostIDs: agentHostIDs,
		CreateTime:   proxy.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdateTime:   proxy.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	// 模型类型文本
	switch proxy.ModelType {
	case "ollama":
		vo.ModelTypeText = "Ollama"
	case "openai":
		vo.ModelTypeText = "OpenAI"
	case "custom":
		vo.ModelTypeText = "自定义"
	default:
		vo.ModelTypeText = proxy.ModelType
	}

	// 状态文本
	if proxy.Status == 1 {
		vo.StatusText = "启用"
	} else {
		vo.StatusText = "禁用"
	}

	// 获取分组名称
	if group, err := uc.assetGroupRepo.GetByID(ctx, proxy.GroupID); err == nil {
		vo.GroupName = group.Name
	}

	// 获取Agent主机名称
	agentHostNames := make([]string, 0, len(agentHostIDs))
	agentOnline := false
	for _, hostID := range agentHostIDs {
		if host, err := uc.hostRepo.GetByID(ctx, hostID); err == nil {
			agentHostNames = append(agentHostNames, host.Name)
			// 检查是否有在线的Agent
			if host.Status == 1 {
				agentOnline = true
			}
		}
	}
	vo.AgentHostNames = agentHostNames
	vo.AgentOnline = agentOnline

	// 生成完整的代理URL（需要从配置中获取服务器地址）
	// 这里暂时使用占位符，实际应该从配置中读取
	vo.ProxyURL = fmt.Sprintf("/api/v1/ai-model-proxy/%s", proxy.ProxyToken)

	// API密钥脱敏（仅显示前4位和后4位）
	if proxy.APIKey != "" {
		decrypted, err := uc.decrypt(proxy.APIKey)
		if err == nil && len(decrypted) > 8 {
			vo.APIKey = decrypted[:4] + "****" + decrypted[len(decrypted)-4:]
		} else {
			vo.APIKey = "****"
		}
	}

	return vo, nil
}
