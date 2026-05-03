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

package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"sync"
)

var (
	globalRSAManager *RSAManager
	once             sync.Once
)

// RSAManager RSA 密钥管理器
type RSAManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	mu         sync.RWMutex
}

// GetRSAManager 获取全局 RSA 管理器单例
func GetRSAManager() *RSAManager {
	once.Do(func() {
		globalRSAManager = &RSAManager{}
		// 初始化时生成密钥对
		if err := globalRSAManager.GenerateKeyPair(2048); err != nil {
			panic("Failed to generate RSA key pair: " + err.Error())
		}
	})
	return globalRSAManager
}

// GenerateKeyPair 生成 RSA 密钥对
func (m *RSAManager) GenerateKeyPair(bits int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}

	m.privateKey = privateKey
	m.publicKey = &privateKey.PublicKey
	return nil
}

// GetPublicKeyPEM 获取 PEM 格式的公钥
func (m *RSAManager) GetPublicKeyPEM() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.publicKey == nil {
		return "", errors.New("public key not initialized")
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(m.publicKey)
	if err != nil {
		return "", err
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM), nil
}

// DecryptPassword 解密密码（Base64 编码的密文）
func (m *RSAManager) DecryptPassword(encryptedPassword string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.privateKey == nil {
		return "", errors.New("private key not initialized")
	}

	// Base64 解码
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", errors.New("invalid base64 encoded password")
	}

	// RSA 解密
	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, m.privateKey, ciphertext)
	if err != nil {
		return "", errors.New("failed to decrypt password")
	}

	return string(plaintext), nil
}

// LoadPrivateKeyFromPEM 从 PEM 格式加载私钥（可选功能，用于密钥持久化）
func (m *RSAManager) LoadPrivateKeyFromPEM(pemData string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return errors.New("failed to parse PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	m.privateKey = privateKey
	m.publicKey = &privateKey.PublicKey
	return nil
}

// GetPrivateKeyPEM 获取 PEM 格式的私钥（用于密钥持久化）
func (m *RSAManager) GetPrivateKeyPEM() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.privateKey == nil {
		return "", errors.New("private key not initialized")
	}

	privKeyBytes := x509.MarshalPKCS1PrivateKey(m.privateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	return string(privKeyPEM), nil
}
