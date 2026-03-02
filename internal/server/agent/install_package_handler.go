package agent

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

// installPackage 临时安装包信息
type installPackage struct {
	ID        string
	AgentID   string
	FilePath  string
	CreatedAt time.Time
}

var (
	installPackages   = make(map[string]*installPackage)
	installPackagesMu sync.RWMutex
)

// GenerateInstallPackage 生成Agent安装包（用于手动部署）
func (s *HTTPServer) GenerateInstallPackage(c *gin.Context) {
	var req struct {
		ServerAddr string `json:"serverAddr"`
	}
	c.ShouldBindJSON(&req)

	if req.ServerAddr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请指定服务端地址"})
		return
	}

	// 生成AgentID
	agentID := uuid.New().String()

	// 签发Agent证书
	certPEM, keyPEM, err := s.tlsMgr.SignAgentCert(agentID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("签发证书失败: %v", err)})
		return
	}
	caCertPEM, err := s.tlsMgr.GetCACertPEM()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("获取CA证书失败: %v", err)})
		return
	}

	// 生成Agent配置
	agentYAML, err := s.renderAgentConfig(agentID, req.ServerAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("生成配置失败: %v", err)})
		return
	}

	// 查找原始安装包
	srcTarball, err := s.findAgentTarball()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("查找安装包失败: %v", err)})
		return
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "opshub-install-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建临时目录失败"})
		return
	}

	// 生成新的tar.gz（包含证书和配置）
	pkgID := uuid.New().String()[:8]
	outPath := filepath.Join(tmpDir, fmt.Sprintf("srehub-agent-%s.tar.gz", pkgID))

	if err := s.buildInstallPackage(outPath, srcTarball, caCertPEM, certPEM, keyPEM, agentYAML); err != nil {
		os.RemoveAll(tmpDir)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("打包失败: %v", err)})
		return
	}

	// 保存到临时映射
	pkg := &installPackage{
		ID:        pkgID,
		AgentID:   agentID,
		FilePath:  outPath,
		CreatedAt: time.Now(),
	}
	installPackagesMu.Lock()
	installPackages[pkgID] = pkg
	installPackagesMu.Unlock()

	// 30分钟后自动清理
	go func() {
		time.Sleep(30 * time.Minute)
		installPackagesMu.Lock()
		delete(installPackages, pkgID)
		installPackagesMu.Unlock()
		os.RemoveAll(tmpDir)
	}()

	appLogger.Info("生成Agent安装包", zap.String("agentID", agentID), zap.String("pkgID", pkgID))
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "安装包生成成功",
		"data": gin.H{
			"agentId":     agentID,
			"packageId":   pkgID,
			"downloadUrl": fmt.Sprintf("/api/v1/agents/install-package/%s", pkgID),
		},
	})
}

// DownloadInstallPackage 下载安装包（需认证）
func (s *HTTPServer) DownloadInstallPackage(c *gin.Context) {
	DownloadInstallPackagePublic(c)
}

// DownloadInstallPackagePublic 下载安装包（公开，包ID随机且有时效）
func DownloadInstallPackagePublic(c *gin.Context) {
	pkgID := c.Param("id")

	installPackagesMu.RLock()
	pkg, ok := installPackages[pkgID]
	installPackagesMu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "安装包不存在或已过期"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=srehub-agent-%s.tar.gz", pkgID))
	c.Header("Content-Type", "application/gzip")
	c.File(pkg.FilePath)
}

// buildInstallPackage 构建包含证书和配置的安装包
func (s *HTTPServer) buildInstallPackage(outPath, srcTarball string, caCert, cert, key, config []byte) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	twOut := tar.NewWriter(gzWriter)
	defer twOut.Close()

	// 先复制原始tarball内容
	srcFile, err := os.Open(srcTarball)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	twIn := tar.NewReader(gzReader)
	prefix := ""
	for {
		hdr, err := twIn.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// 检测顶层目录前缀
		if prefix == "" && strings.Contains(hdr.Name, "/") {
			parts := strings.SplitN(hdr.Name, "/", 2)
			if len(parts) == 2 {
				prefix = parts[0] + "/"
			}
		}
		if err := twOut.WriteHeader(hdr); err != nil {
			return err
		}
		if hdr.Size > 0 {
			if _, err := io.Copy(twOut, twIn); err != nil {
				return err
			}
		}
	}

	// 添加证书和配置文件
	extraFiles := map[string][]byte{
		prefix + "certs/ca.pem":   caCert,
		prefix + "certs/cert.pem": cert,
		prefix + "certs/key.pem":  key,
		prefix + "agent.yaml":     config,
	}
	for name, data := range extraFiles {
		hdr := &tar.Header{
			Name:    name,
			Size:    int64(len(data)),
			Mode:    0644,
			ModTime: time.Now(),
		}
		if strings.HasSuffix(name, "key.pem") {
			hdr.Mode = 0600
		}
		if err := twOut.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := twOut.Write(data); err != nil {
			return err
		}
	}

	return nil
}
