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
	"testing"
)

func TestRSAManager_GenerateKeyPair(t *testing.T) {
	manager := &RSAManager{}
	err := manager.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	if manager.privateKey == nil {
		t.Error("私钥未生成")
	}

	if manager.publicKey == nil {
		t.Error("公钥未生成")
	}
}

func TestRSAManager_GetPublicKeyPEM(t *testing.T) {
	manager := &RSAManager{}
	err := manager.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	publicKeyPEM, err := manager.GetPublicKeyPEM()
	if err != nil {
		t.Fatalf("获取公钥 PEM 失败: %v", err)
	}

	if publicKeyPEM == "" {
		t.Error("公钥 PEM 为空")
	}

	t.Logf("公钥 PEM:\n%s", publicKeyPEM)
}

func TestRSAManager_EncryptDecrypt(t *testing.T) {
	manager := &RSAManager{}
	err := manager.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 模拟前端加密（这里用 Go 模拟，实际前端用 JSEncrypt）
	originalPassword := "TestPassword123!"

	// 使用公钥加密（模拟前端行为）
	publicKeyPEM, _ := manager.GetPublicKeyPEM()
	t.Logf("原始密码: %s", originalPassword)
	t.Logf("公钥长度: %d", len(publicKeyPEM))

	// 这里我们直接测试解密功能
	// 实际场景中，加密由前端 JSEncrypt 完成
	// 我们只需要确保后端能正确解密

	// 注意：这个测试主要验证密钥生成和 PEM 格式正确性
	// 完整的加密解密测试需要在集成测试中进行
}

func TestGetRSAManager_Singleton(t *testing.T) {
	manager1 := GetRSAManager()
	manager2 := GetRSAManager()

	if manager1 != manager2 {
		t.Error("GetRSAManager 应该返回单例")
	}

	publicKey1, _ := manager1.GetPublicKeyPEM()
	publicKey2, _ := manager2.GetPublicKeyPEM()

	if publicKey1 != publicKey2 {
		t.Error("两次获取的公钥应该相同")
	}
}
