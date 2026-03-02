package agent

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

// TLSManager 管理CA和Agent证书
type TLSManager struct {
	certDir string
	caCert  *x509.Certificate
	caKey   *ecdsa.PrivateKey
}

// NewTLSManager 创建TLS管理器
func NewTLSManager(certDir string) *TLSManager {
	return &TLSManager{certDir: certDir}
}

// InitCA 初始化CA，如果不存在则生成
func (m *TLSManager) InitCA() error {
	if err := os.MkdirAll(m.certDir, 0700); err != nil {
		return fmt.Errorf("创建证书目录失败: %w", err)
	}
	caCertPath := filepath.Join(m.certDir, "ca.pem")
	caKeyPath := filepath.Join(m.certDir, "ca-key.pem")

	// 检查CA是否已存在
	if _, err := os.Stat(caCertPath); err == nil {
		return m.loadCA(caCertPath, caKeyPath)
	}

	appLogger.Info("生成新的CA证书...")

	// 生成CA私钥
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("生成CA私钥失败: %w", err)
	}

	// CA证书模板
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"OpsHub"},
			CommonName:   "OpsHub Agent CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("创建CA证书失败: %w", err)
	}

	// 保存CA证书
	if err := savePEM(caCertPath, "CERTIFICATE", caCertDER); err != nil {
		return err
	}

	// 保存CA私钥
	caKeyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		return fmt.Errorf("序列化CA私钥失败: %w", err)
	}
	if err := savePEM(caKeyPath, "EC PRIVATE KEY", caKeyDER); err != nil {
		return err
	}
	os.Chmod(caKeyPath, 0600)

	m.caCert = caTemplate
	m.caKey = caKey
	appLogger.Info("CA证书生成完成", zap.String("path", caCertPath))
	return nil
}

// loadCA 加载已有CA
func (m *TLSManager) loadCA(certPath, keyPath string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("读取CA证书失败: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("解析CA证书PEM失败")
	}
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("解析CA证书失败: %w", err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("读取CA私钥失败: %w", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("解析CA私钥PEM失败")
	}
	caKey, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("解析CA私钥失败: %w", err)
	}

	m.caCert = caCert
	m.caKey = caKey
	appLogger.Info("CA证书加载完成")
	return nil
}

// SignAgentCert 为Agent签发客户端证书
func (m *TLSManager) SignAgentCert(agentID string, hostIP string) (certPEM, keyPEM []byte, err error) {
	if m.caCert == nil || m.caKey == nil {
		return nil, nil, fmt.Errorf("CA未初始化")
	}

	// 生成Agent私钥
	agentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("生成Agent私钥失败: %w", err)
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"OpsHub Agent"},
			CommonName:   agentID,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	if ip := net.ParseIP(hostIP); ip != nil {
		template.IPAddresses = []net.IP{ip}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, m.caCert, &agentKey.PublicKey, m.caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("签发Agent证书失败: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	agentKeyDER, err := x509.MarshalECPrivateKey(agentKey)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化Agent私钥失败: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: agentKeyDER})

	// 同时保存到文件
	agentCertDir := filepath.Join(m.certDir, agentID)
	os.MkdirAll(agentCertDir, 0700)
	os.WriteFile(filepath.Join(agentCertDir, "cert.pem"), certPEM, 0644)
	os.WriteFile(filepath.Join(agentCertDir, "key.pem"), keyPEM, 0600)

	appLogger.Info("Agent证书签发完成", zap.String("agentID", agentID), zap.String("hostIP", hostIP))
	return certPEM, keyPEM, nil
}

// LoadServerTLSConfig 加载gRPC server的mTLS配置
func (m *TLSManager) LoadServerTLSConfig() (*tls.Config, error) {
	if m.caCert == nil || m.caKey == nil {
		return nil, fmt.Errorf("CA未初始化")
	}

	// 生成server证书
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成server私钥失败: %w", err)
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	serverTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"OpsHub"},
			CommonName:   "OpsHub gRPC Server",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		IPAddresses: []net.IP{net.ParseIP("0.0.0.0"), net.ParseIP("127.0.0.1")},
		DNSNames:    []string{"localhost"},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, m.caCert, &serverKey.PublicKey, m.caKey)
	if err != nil {
		return nil, fmt.Errorf("创建server证书失败: %w", err)
	}

	serverCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})
	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		return nil, fmt.Errorf("序列化server私钥失败: %w", err)
	}
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyDER})

	cert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("加载server证书对失败: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(m.caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
	}, nil
}

// GetCACertPEM 获取CA证书PEM
func (m *TLSManager) GetCACertPEM() ([]byte, error) {
	caCertPath := filepath.Join(m.certDir, "ca.pem")
	return os.ReadFile(caCertPath)
}

// savePEM 保存PEM文件
func savePEM(path string, pemType string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建文件 %s 失败: %w", path, err)
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: pemType, Bytes: data})
}
