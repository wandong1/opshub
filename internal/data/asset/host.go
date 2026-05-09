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

	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"gorm.io/gorm"
)

type hostRepo struct {
	db *gorm.DB
}

// NewHostRepo 创建主机仓库
func NewHostRepo(db *gorm.DB) asset.HostRepo {
	return &hostRepo{db: db}
}

// Create 创建主机
func (r *hostRepo) Create(ctx context.Context, host *asset.Host) error {
	return r.db.WithContext(ctx).Create(host).Error
}

// CreateOrUpdate 创建或恢复主机（如果IP已被软删除则恢复）
func (r *hostRepo) CreateOrUpdate(ctx context.Context, host *asset.Host) error {
	// 先查找是否存在该IP的记录（包括软删除的）
	var existing asset.Host
	err := r.db.WithContext(ctx).Unscoped().Where("ip = ?", host.IP).First(&existing).Error

	if err == nil {
		// 找到了记录
		if existing.DeletedAt.Valid {
			// 记录已被软删除，恢复它
			existing.Name = host.Name
			existing.GroupID = host.GroupID
			existing.SSHUser = host.SSHUser
			existing.IP = host.IP
			existing.Port = host.Port
			existing.CredentialID = host.CredentialID
			existing.Tags = host.Tags
			existing.Description = host.Description
			existing.Status = host.Status
			existing.DeletedAt.Time = *new(time.Time) // 清除删除时间
			existing.DeletedAt.Valid = false
			return r.db.WithContext(ctx).Unscoped().Save(&existing).Error
		}
		// 记录未被删除，返回错误
		return fmt.Errorf("IP地址 %s 已存在", host.IP)
	}

	// 没找到记录，创建新的
	return r.db.WithContext(ctx).Create(host).Error
}

// Update 更新主机
func (r *hostRepo) Update(ctx context.Context, host *asset.Host) error {
	return r.db.WithContext(ctx).Save(host).Error
}

// Delete 删除主机
func (r *hostRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&asset.Host{}, id).Error
}

// GetByID 根据ID获取主机
func (r *hostRepo) GetByID(ctx context.Context, id uint) (*asset.Host, error) {
	var host asset.Host
	err := r.db.WithContext(ctx).First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// List 列表查询
func (r *hostRepo) List(ctx context.Context, page, pageSize int, keyword string, groupIDs []uint, status *int, tags []string, accessibleHostIDs []uint) ([]*asset.Host, int64, error) {
	var hosts []*asset.Host
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.Host{})

	if keyword != "" {
		query = query.Where("name LIKE ? OR ip LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 添加分组ID筛选（支持多个分组ID）
	if len(groupIDs) > 0 {
		query = query.Where("group_id IN ?", groupIDs)
	}

	// 添加状态筛选
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 添加标签筛选（多选，AND 逻辑：主机必须包含所有选中的标签）
	if len(tags) > 0 {
		for _, tag := range tags {
			if tag != "" {
				query = query.Where("tags LIKE ?", "%"+tag+"%")
			}
		}
	}

	// 添加可访问主机ID筛选
	// 如果 accessibleHostIDs 为空切片（非nil），表示用户没有任何权限，应该返回空列表
	// 如果 accessibleHostIDs 为nil，表示不进行权限筛选（管理员或未启用权限控制）
	if accessibleHostIDs != nil {
		if len(accessibleHostIDs) == 0 {
			// 用户没有任何主机访问权限，返回空列表
			return []*asset.Host{}, 0, nil
		}
		query = query.Where("id IN ?", accessibleHostIDs)
	}

	err := query.Order("id DESC").Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&hosts).Error
	if err != nil {
		return nil, 0, err
	}

	return hosts, total, nil
}

// GetByGroupID 根据分组ID获取主机列表
func (r *hostRepo) GetByGroupID(ctx context.Context, groupID uint) ([]*asset.Host, error) {
	var hosts []*asset.Host
	err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&hosts).Error
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

// GetByIP 根据IP获取主机
func (r *hostRepo) GetByIP(ctx context.Context, ip string) (*asset.Host, error) {
	var host asset.Host
	err := r.db.WithContext(ctx).Where("ip = ?", ip).First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// GetByCloudInstanceID 根据云实例ID获取主机
func (r *hostRepo) GetByCloudInstanceID(ctx context.Context, instanceID string) (*asset.Host, error) {
	var host asset.Host
	err := r.db.WithContext(ctx).Where("cloud_instance_id = ?", instanceID).First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// CountByCredentialID 统计使用指定凭证的主机数量
func (r *hostRepo) CountByCredentialID(ctx context.Context, credentialID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&asset.Host{}).Where("credential_id = ?", credentialID).Count(&count).Error
	return count, err
}

// GetAll 获取所有主机（仅健康检查需要的字段）
func (r *hostRepo) GetAll(ctx context.Context) ([]*asset.Host, error) {
	var hosts []*asset.Host
	err := r.db.WithContext(ctx).
		Select("id, ip, port, ssh_user, credential_id, status, connection_mode, agent_id").
		Find(&hosts).Error
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

// credentialRepo 凭证仓库
type credentialRepo struct {
	db           *gorm.DB
	encryptionKey []byte
}

// NewCredentialRepo 创建凭证仓库
func NewCredentialRepo(db *gorm.DB) asset.CredentialRepo {
	// AES-256要求密钥长度必须是32字节（256位）
	encryptionKey := []byte("opshub-enc-key-32-bytes-long!!!!")
	return &credentialRepo{
		db:           db,
		encryptionKey: encryptionKey,
	}
}

// encrypt 加密
func (r *credentialRepo) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(r.encryptionKey)
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
func (r *credentialRepo) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(r.encryptionKey)
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

// Create 创建凭证
func (r *credentialRepo) Create(ctx context.Context, credential *asset.Credential) error {
	// 加密敏感信息
	if credential.Password != "" {
		encrypted, err := r.encrypt(credential.Password)
		if err != nil {
			return fmt.Errorf("加密密码失败: %w", err)
		}
		credential.Password = encrypted
	}

	if credential.PrivateKey != "" {
		encrypted, err := r.encrypt(credential.PrivateKey)
		if err != nil {
			return fmt.Errorf("加密私钥失败: %w", err)
		}
		credential.PrivateKey = encrypted
	}

	if credential.Passphrase != "" {
		encrypted, err := r.encrypt(credential.Passphrase)
		if err != nil {
			return fmt.Errorf("加密私钥密码失败: %w", err)
		}
		credential.Passphrase = encrypted
	}

	return r.db.WithContext(ctx).Create(credential).Error
}

// Update 更新凭证
func (r *credentialRepo) Update(ctx context.Context, credential *asset.Credential) error {
	// 加密敏感信息
	if credential.Password != "" {
		encrypted, err := r.encrypt(credential.Password)
		if err != nil {
			return fmt.Errorf("加密密码失败: %w", err)
		}
		credential.Password = encrypted
	}

	if credential.PrivateKey != "" {
		encrypted, err := r.encrypt(credential.PrivateKey)
		if err != nil {
			return fmt.Errorf("加密私钥失败: %w", err)
		}
		credential.PrivateKey = encrypted
	}

	if credential.Passphrase != "" {
		encrypted, err := r.encrypt(credential.Passphrase)
		if err != nil {
			return fmt.Errorf("加密私钥密码失败: %w", err)
		}
		credential.Passphrase = encrypted
	}

	return r.db.WithContext(ctx).Save(credential).Error
}

// Delete 删除凭证
func (r *credentialRepo) Delete(ctx context.Context, id uint) error {
	// 检查是否有主机使用此凭证
	var count int64
	if err := r.db.WithContext(ctx).Model(&asset.Host{}).Where("credential_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该凭证正在被 %d 个主机使用，无法删除", count)
	}

	return r.db.WithContext(ctx).Delete(&asset.Credential{}, id).Error
}

// GetByID 根据ID获取凭证
func (r *credentialRepo) GetByID(ctx context.Context, id uint) (*asset.Credential, error) {
	var credential asset.Credential
	err := r.db.WithContext(ctx).First(&credential, id).Error
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

// GetByIDDecrypted 根据ID获取凭证（解密后的）
func (r *credentialRepo) GetByIDDecrypted(ctx context.Context, id uint) (*asset.Credential, error) {
	credential, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 解密敏感信息
	if credential.Password != "" {
		decrypted, err := r.decrypt(credential.Password)
		if err != nil {
			return nil, fmt.Errorf("解密密码失败: %w", err)
		}
		credential.Password = decrypted
	}

	if credential.PrivateKey != "" {
		decrypted, err := r.decrypt(credential.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("解密私钥失败: %w", err)
		}
		credential.PrivateKey = decrypted
	}

	if credential.Passphrase != "" {
		decrypted, err := r.decrypt(credential.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("解密私钥密码失败: %w", err)
		}
		credential.Passphrase = decrypted
	}

	return credential, nil
}

// List 列表查询
func (r *credentialRepo) List(ctx context.Context, page, pageSize int, keyword string) ([]*asset.Credential, int64, error) {
	var credentials []*asset.Credential
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.Credential{})

	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	err := query.Order("id DESC").Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&credentials).Error
	if err != nil {
		return nil, 0, err
	}

	return credentials, total, nil
}

// GetAll 获取所有凭证
func (r *credentialRepo) GetAll(ctx context.Context) ([]*asset.Credential, error) {
	var credentials []*asset.Credential
	err := r.db.WithContext(ctx).Order("id DESC").Find(&credentials).Error
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

// cloudAccountRepo 云平台账号仓库
type cloudAccountRepo struct {
	db *gorm.DB
}

// NewCloudAccountRepo 创建云平台账号仓库
func NewCloudAccountRepo(db *gorm.DB) asset.CloudAccountRepo {
	return &cloudAccountRepo{db: db}
}

// Create 创建云平台账号
func (r *cloudAccountRepo) Create(ctx context.Context, account *asset.CloudAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// Update 更新云平台账号
func (r *cloudAccountRepo) Update(ctx context.Context, account *asset.CloudAccount) error {
	return r.db.WithContext(ctx).Save(account).Error
}

// Delete 删除云平台账号
func (r *cloudAccountRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&asset.CloudAccount{}, id).Error
}

// GetByID 根据ID获取云平台账号
func (r *cloudAccountRepo) GetByID(ctx context.Context, id uint) (*asset.CloudAccount, error) {
	var account asset.CloudAccount
	err := r.db.WithContext(ctx).First(&account, id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// List 列表查询
func (r *cloudAccountRepo) List(ctx context.Context, page, pageSize int) ([]*asset.CloudAccount, int64, error) {
	var accounts []*asset.CloudAccount
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.CloudAccount{})

	err := query.Order("id DESC").Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&accounts).Error
	if err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

// GetAll 获取所有启用的云平台账号
func (r *cloudAccountRepo) GetAll(ctx context.Context) ([]*asset.CloudAccount, error) {
	var accounts []*asset.CloudAccount
	err := r.db.WithContext(ctx).Order("id DESC").Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetStatistics 获取主机统计信息（支持筛选条件）
func (r *hostRepo) GetStatistics(ctx context.Context, keyword string, groupIDs []uint, status *int, tags []string) (*asset.HostStatistics, error) {
	stats := &asset.HostStatistics{
		TypeStats:       make(map[string]int64),
		ArchStats:       make(map[string]int64),
		ConnectionStats: make(map[string]int64),
	}

	// 构建基础查询函数，每次调用返回新的查询对象
	buildBaseQuery := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&asset.Host{})
		if keyword != "" {
			query = query.Where("name LIKE ? OR ip LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
		if len(groupIDs) > 0 {
			query = query.Where("group_id IN ?", groupIDs)
		}
		if status != nil {
			query = query.Where("status = ?", *status)
		}
		if len(tags) > 0 {
			for _, tag := range tags {
				if tag != "" {
					query = query.Where("tags LIKE ?", "%"+tag+"%")
				}
			}
		}
		return query
	}

	// 总数统计
	buildBaseQuery().Count(&stats.TotalCount)

	// 状态统计
	buildBaseQuery().Where("status = ?", 1).Count(&stats.OnlineCount)
	buildBaseQuery().Where("status = ?", 0).Count(&stats.OfflineCount)

	// SSH和Agent数量统计
	buildBaseQuery().Where("connection_mode = ?", "ssh").Count(&stats.SSHCount)
	buildBaseQuery().Where("connection_mode = ?", "agent").Count(&stats.AgentCount)
	buildBaseQuery().Where("connection_mode = ? AND agent_status = ?", "agent", "online").Count(&stats.AgentOnlineCount)
	buildBaseQuery().Where("connection_mode = ? AND agent_status = ?", "agent", "offline").Count(&stats.AgentOfflineCount)

	// 类型统计
	var typeResults []struct {
		Type  string
		Count int64
	}
	buildBaseQuery().Select("type, COUNT(*) as count").Group("type").Scan(&typeResults)
	for _, result := range typeResults {
		stats.TypeStats[result.Type] = result.Count
	}

	// 分组数量
	r.db.WithContext(ctx).Model(&asset.AssetGroup{}).Count(&stats.GroupCount)

	// 资源统计
	var resourceStats struct {
		TotalCPU    *int64
		TotalMemory *uint64
	}
	buildBaseQuery().Select("SUM(cpu_cores) as total_cpu, SUM(memory_total) as total_memory").
		Scan(&resourceStats)
	if resourceStats.TotalCPU != nil {
		stats.CPUTotalCores = *resourceStats.TotalCPU
	}
	if resourceStats.TotalMemory != nil {
		stats.MemoryTotal = *resourceStats.TotalMemory
	}

	// 架构统计
	var archResults []struct {
		Arch  string
		Count int64
	}
	buildBaseQuery().Where("arch != ''").
		Select("arch, COUNT(*) as count").Group("arch").Scan(&archResults)
	for _, result := range archResults {
		stats.ArchStats[result.Arch] = result.Count
	}

	// 连接方式统计
	var connResults []struct {
		Mode  string
		Count int64
	}
	buildBaseQuery().Select("connection_mode, COUNT(*) as count").Group("connection_mode").Scan(&connResults)
	for _, result := range connResults {
		stats.ConnectionStats[result.Mode] = result.Count
	}

	// Agent统计
	buildBaseQuery().Where("agent_status = ?", "online").Count(&stats.AgentStats.OnlineCount)
	buildBaseQuery().Where("agent_status = ?", "offline").Count(&stats.AgentStats.OfflineCount)

	// GPU统计
	var gpuSum struct {
		TotalGPUs   *int64
		TotalMemory *uint64
	}
	buildBaseQuery().Select("SUM(gpu_count) as total_gpus, SUM(gpu_memory_total) as total_memory").Scan(&gpuSum)
	if gpuSum.TotalGPUs != nil {
		stats.GPUStats.TotalGPUs = *gpuSum.TotalGPUs
	}
	if gpuSum.TotalMemory != nil {
		stats.GPUStats.TotalMemory = *gpuSum.TotalMemory
	}
	buildBaseQuery().Where("gpu_count > 0").Count(&stats.GPUStats.HostsWithGPU)

	// GPU型号统计
	var gpuModelResults []struct {
		Model string
		Count int64
	}
	buildBaseQuery().Where("gpu_model != ''").
		Select("gpu_model, SUM(gpu_count) as count").Group("gpu_model").Scan(&gpuModelResults)
	stats.GPUStats.ModelStats = make(map[string]int64)
	for _, result := range gpuModelResults {
		stats.GPUStats.ModelStats[result.Model] = result.Count
	}

	return stats, nil
}
