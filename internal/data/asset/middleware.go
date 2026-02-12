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

	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"gorm.io/gorm"
)

type middlewareRepo struct {
	db            *gorm.DB
	encryptionKey []byte
}

// NewMiddlewareRepo 创建中间件仓库
func NewMiddlewareRepo(db *gorm.DB) asset.MiddlewareRepo {
	encryptionKey := []byte("opshub-enc-key-32-bytes-long!!!!")
	return &middlewareRepo{db: db, encryptionKey: encryptionKey}
}

// encrypt 加密
func (r *middlewareRepo) encrypt(plaintext string) (string, error) {
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
func (r *middlewareRepo) decrypt(ciphertext string) (string, error) {
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

// Create 创建中间件
func (r *middlewareRepo) Create(ctx context.Context, mw *asset.Middleware) error {
	if mw.Password != "" {
		encrypted, err := r.encrypt(mw.Password)
		if err != nil {
			return fmt.Errorf("加密密码失败: %w", err)
		}
		mw.Password = encrypted
	}
	return r.db.WithContext(ctx).Create(mw).Error
}

// Update 更新中间件
func (r *middlewareRepo) Update(ctx context.Context, mw *asset.Middleware) error {
	if mw.Password != "" {
		// 检查密码是否已经是加密格式（base64 编码的 AES-GCM 密文）
		// 如果能成功解密，说明是已加密的旧密码，不需要再次加密
		if _, err := r.decrypt(mw.Password); err != nil {
			// 解密失败，说明是新的明文密码，需要加密
			encrypted, err := r.encrypt(mw.Password)
			if err != nil {
				return fmt.Errorf("加密密码失败: %w", err)
			}
			mw.Password = encrypted
		}
	}
	return r.db.WithContext(ctx).Save(mw).Error
}

// Delete 删除中间件
func (r *middlewareRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&asset.Middleware{}, id).Error
}

// GetByID 根据ID获取中间件
func (r *middlewareRepo) GetByID(ctx context.Context, id uint) (*asset.Middleware, error) {
	var mw asset.Middleware
	err := r.db.WithContext(ctx).First(&mw, id).Error
	if err != nil {
		return nil, err
	}
	return &mw, nil
}

// GetByIDDecrypted 根据ID获取中间件（解密密码）
func (r *middlewareRepo) GetByIDDecrypted(ctx context.Context, id uint) (*asset.Middleware, error) {
	mw, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if mw.Password != "" {
		decrypted, err := r.decrypt(mw.Password)
		if err != nil {
			return nil, fmt.Errorf("解密密码失败: %w", err)
		}
		mw.Password = decrypted
	}
	return mw, nil
}

// List 列表查询
func (r *middlewareRepo) List(ctx context.Context, page, pageSize int, keyword, mwType string, groupIDs []uint, status *int, accessibleIDs []uint) ([]*asset.Middleware, int64, error) {
	var middlewares []*asset.Middleware
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.Middleware{})

	if keyword != "" {
		query = query.Where("name LIKE ? OR host LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if mwType != "" {
		query = query.Where("type = ?", mwType)
	}
	if len(groupIDs) > 0 {
		query = query.Where("group_id IN ?", groupIDs)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if accessibleIDs != nil {
		if len(accessibleIDs) == 0 {
			return []*asset.Middleware{}, 0, nil
		}
		query = query.Where("id IN ?", accessibleIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&middlewares).Error
	if err != nil {
		return nil, 0, err
	}

	return middlewares, total, nil
}

// BatchDelete 批量删除
func (r *middlewareRepo) BatchDelete(ctx context.Context, ids []uint) error {
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&asset.Middleware{}).Error
}

// UpdateStatus 更新状态
func (r *middlewareRepo) UpdateStatus(ctx context.Context, id uint, status int, version string) error {
	updates := map[string]interface{}{
		"status":       status,
		"last_checked": r.db.NowFunc(),
	}
	if version != "" {
		updates["version"] = version
	}
	return r.db.WithContext(ctx).Model(&asset.Middleware{}).Where("id = ?", id).Updates(updates).Error
}
