package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	agentmodel "github.com/ydcloud-dy/opshub/internal/agent"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	sshclient "github.com/ydcloud-dy/opshub/pkg/ssh"
	"go.uber.org/zap"
)

// DeployAgent 部署Agent到目标主机
func (s *HTTPServer) DeployAgent(c *gin.Context) {
	hostIDStr := c.Param("hostId")
	hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的主机ID"})
		return
	}

	var req struct {
		ServerAddr string `json:"serverAddr"`
	}
	c.ShouldBindJSON(&req)

	if err := s.deployToHost(c.Request.Context(), uint(hostID), req.ServerAddr); err != nil {
		appLogger.Error("部署Agent失败", zap.Uint64("hostID", hostID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("部署失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Agent部署成功"})
}

// BatchDeployAgent 批量部署Agent
func (s *HTTPServer) BatchDeployAgent(c *gin.Context) {
	var req struct {
		HostIDs    []uint `json:"hostIds" binding:"required"`
		ServerAddr string `json:"serverAddr"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	results := make([]map[string]any, 0, len(req.HostIDs))
	for _, hostID := range req.HostIDs {
		err := s.deployToHost(c.Request.Context(), hostID, req.ServerAddr)
		result := map[string]any{"hostId": hostID, "success": err == nil}
		if err != nil {
			result["error"] = err.Error()
		}
		results = append(results, result)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "批量部署完成", "data": results})
}

// deployToHost 部署Agent到单台主机
// serverAddr 为用户手动指定的服务端地址（IP或IP:Port），为空时自动检测
func (s *HTTPServer) deployToHost(ctx context.Context, hostID uint, serverAddr string) error {
	// 获取主机信息
	hostVO, err := s.hostUseCase.GetByID(ctx, hostID)
	if err != nil {
		return fmt.Errorf("获取主机信息失败: %w", err)
	}

	// 获取解密后的凭证
	credentialRepo := s.hostUseCase.GetCredentialRepo()
	credential, err := credentialRepo.GetByIDDecrypted(ctx, hostVO.CredentialID)
	if err != nil {
		return fmt.Errorf("获取凭证失败: %w", err)
	}

	// 生成AgentID
	agentID := uuid.New().String()

	// 签发Agent证书
	certPEM, keyPEM, err := s.tlsMgr.SignAgentCert(agentID, hostVO.IP)
	if err != nil {
		return fmt.Errorf("签发证书失败: %w", err)
	}

	caCertPEM, err := s.tlsMgr.GetCACertPEM()
	if err != nil {
		return fmt.Errorf("获取CA证书失败: %w", err)
	}

	// SSH连接目标主机
	var privateKey []byte
	if credential.PrivateKey != "" {
		privateKey = []byte(credential.PrivateKey)
	}
	client, err := sshclient.NewClient(hostVO.IP, hostVO.Port, hostVO.SSHUser, credential.Password, privateKey, credential.Passphrase)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	defer client.Close()

	// 获取服务端地址：优先使用用户指定的地址，否则自动检测
	if serverAddr == "" {
		if out, err := client.Execute("echo $SSH_CLIENT | awk '{print $1}'"); err == nil {
			addr := strings.TrimSpace(out)
			if addr != "" {
				serverAddr = addr
			}
		}
		if serverAddr == "" {
			serverAddr, _ = os.Hostname()
		}
	}

	// 生成Agent配置
	agentYAML, err := s.renderAgentConfig(agentID, serverAddr)
	if err != nil {
		return fmt.Errorf("生成Agent配置失败: %w", err)
	}

	deployPath := s.grpcServer.conf.Agent.DeployPath

	// 查找本地tar.gz安装包
	tarballPath, err := s.findAgentTarball()
	if err != nil {
		return fmt.Errorf("查找Agent安装包失败: %w", err)
	}

	// 通过SSH在远程主机创建临时目录并上传安装包
	tmpDir := "/tmp/srehub-agent-install"
	if _, err := client.Execute(fmt.Sprintf("rm -rf %s && mkdir -p %s", tmpDir, tmpDir)); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 上传tar.gz安装包
	remoteTarball := tmpDir + "/srehub-agent.tar.gz"
	if err := client.UploadFile(tarballPath, remoteTarball); err != nil {
		return fmt.Errorf("上传安装包失败: %w", err)
	}

	// 解压安装包
	extractCmd := fmt.Sprintf("cd %s && tar xzf srehub-agent.tar.gz --strip-components=1", tmpDir)
	if _, err := client.Execute(extractCmd); err != nil {
		return fmt.Errorf("解压安装包失败: %w", err)
	}

	// 通过SSH写入证书和配置文件（使用 bash heredoc，避免SFTP目录问题）
	certFiles := map[string][]byte{
		tmpDir + "/certs/ca.pem":   caCertPEM,
		tmpDir + "/certs/cert.pem": certPEM,
		tmpDir + "/certs/key.pem":  keyPEM,
		tmpDir + "/agent.yaml":     agentYAML,
	}
	// 确保certs目录存在
	client.Execute(fmt.Sprintf("mkdir -p %s/certs", tmpDir))

	for filePath, data := range certFiles {
		// 使用 base64 编码通过SSH写入文件，避免特殊字符问题
		encoded := base64Encode(data)
		writeCmd := fmt.Sprintf("echo '%s' | base64 -d > %s", encoded, filePath)
		if _, err := client.Execute(writeCmd); err != nil {
			return fmt.Errorf("写入文件 %s 失败: %w", filePath, err)
		}
	}

	// 设置key.pem权限
	client.Execute(fmt.Sprintf("chmod 600 %s/certs/key.pem", tmpDir))

	// 执行安装脚本（使用sudo提权）
	installCmd := fmt.Sprintf("cd %s && sudo INSTALL_DIR=%s bash install.sh", tmpDir, deployPath)
	if out, err := client.Execute(installCmd); err != nil {
		return fmt.Errorf("执行安装脚本失败: %s, %w", out, err)
	}

	// 将证书和配置复制到安装目录（install.sh不会覆盖已有配置，但证书是新生成的）
	copyCertsCmd := fmt.Sprintf("sudo mkdir -p %s/certs && sudo cp -f %s/certs/* %s/certs/ && sudo cp -f %s/agent.yaml %s/agent.yaml && sudo chmod 600 %s/certs/key.pem",
		deployPath, tmpDir, deployPath, tmpDir, deployPath, deployPath)
	if _, err := client.Execute(copyCertsCmd); err != nil {
		return fmt.Errorf("复制证书到安装目录失败: %w", err)
	}

	// 启动服务
	client.Execute("sudo systemctl daemon-reload")
	client.Execute("sudo systemctl enable --now srehub-agent")

	// 清理临时目录
	client.Execute(fmt.Sprintf("rm -rf %s", tmpDir))

	// 更新Host记录
	s.db.Exec("UPDATE hosts SET agent_id = ?, agent_status = 'installed', connection_mode = 'agent' WHERE id = ?", agentID, hostID)

	// 创建AgentInfo记录
	agentInfo := &agentmodel.AgentInfo{
		AgentID: agentID,
		HostID:  hostID,
		Status:  "installed",
	}
	s.grpcServer.AgentRepo().Create(ctx, agentInfo)

	appLogger.Info("Agent部署完成", zap.Uint("hostID", hostID), zap.String("agentID", agentID))
	return nil
}

// findAgentTarball 查找Agent安装包tar.gz
func (s *HTTPServer) findAgentTarball() (string, error) {
	binaryDir := s.grpcServer.conf.Agent.BinaryDir
	entries, err := os.ReadDir(binaryDir)
	if err != nil {
		return "", fmt.Errorf("读取目录 %s 失败: %w", binaryDir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tar.gz") {
			return binaryDir + "/" + entry.Name(), nil
		}
	}
	// 回退：检查裸二进制是否存在
	binaryPath := binaryDir + "/srehub-agent"
	if _, err := os.Stat(binaryPath); err == nil {
		return "", fmt.Errorf("仅找到裸二进制文件，请先运行 agent/build.sh 生成安装包")
	}
	return "", fmt.Errorf("在 %s 中未找到Agent安装包(.tar.gz)", binaryDir)
}

// base64Encode 将字节数据编码为base64字符串
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// UpdateAgent 更新目标主机上的Agent
func (s *HTTPServer) UpdateAgent(c *gin.Context) {
	hostIDStr := c.Param("hostId")
	hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的主机ID"})
		return
	}

	var req struct {
		ServerAddr string `json:"serverAddr"`
	}
	c.ShouldBindJSON(&req)

	if err := s.updateAgentOnHost(c.Request.Context(), uint(hostID), req.ServerAddr); err != nil {
		appLogger.Error("更新Agent失败", zap.Uint64("hostID", hostID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("更新失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Agent更新成功"})
}

// UninstallAgent 卸载目标主机上的Agent
func (s *HTTPServer) UninstallAgent(c *gin.Context) {
	hostIDStr := c.Param("hostId")
	hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的主机ID"})
		return
	}

	if err := s.uninstallAgentOnHost(c.Request.Context(), uint(hostID)); err != nil {
		appLogger.Error("卸载Agent失败", zap.Uint64("hostID", hostID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("卸载失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Agent卸载成功"})
}

// updateAgentOnHost 更新单台主机上的Agent二进制
// serverAddr 不为空时同时更新Agent配置中的服务端地址
func (s *HTTPServer) updateAgentOnHost(ctx context.Context, hostID uint, serverAddr string) error {
	hostVO, err := s.hostUseCase.GetByID(ctx, hostID)
	if err != nil {
		return fmt.Errorf("获取主机信息失败: %w", err)
	}

	credentialRepo := s.hostUseCase.GetCredentialRepo()
	credential, err := credentialRepo.GetByIDDecrypted(ctx, hostVO.CredentialID)
	if err != nil {
		return fmt.Errorf("获取凭证失败: %w", err)
	}

	var privateKey []byte
	if credential.PrivateKey != "" {
		privateKey = []byte(credential.PrivateKey)
	}
	client, err := sshclient.NewClient(hostVO.IP, hostVO.Port, hostVO.SSHUser, credential.Password, privateKey, credential.Passphrase)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	defer client.Close()

	tarballPath, err := s.findAgentTarball()
	if err != nil {
		return fmt.Errorf("查找Agent安装包失败: %w", err)
	}

	deployPath := s.grpcServer.conf.Agent.DeployPath
	tmpDir := "/tmp/srehub-agent-install"

	// 创建临时目录
	if _, err := client.Execute(fmt.Sprintf("rm -rf %s && mkdir -p %s", tmpDir, tmpDir)); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 上传并解压安装包
	remoteTarball := tmpDir + "/srehub-agent.tar.gz"
	if err := client.UploadFile(tarballPath, remoteTarball); err != nil {
		return fmt.Errorf("上传安装包失败: %w", err)
	}
	extractCmd := fmt.Sprintf("cd %s && tar xzf srehub-agent.tar.gz --strip-components=1", tmpDir)
	if _, err := client.Execute(extractCmd); err != nil {
		return fmt.Errorf("解压安装包失败: %w", err)
	}

	// 停止服务 → 替换二进制 → 启动服务
	client.Execute("sudo systemctl stop srehub-agent")
	copyCmd := fmt.Sprintf("sudo cp -f %s/srehub-agent %s/srehub-agent", tmpDir, deployPath)
	if _, err := client.Execute(copyCmd); err != nil {
		return fmt.Errorf("替换Agent二进制失败: %w", err)
	}

	// 如果指定了新的服务端地址，更新Agent配置
	if serverAddr != "" {
		agentID := hostVO.AgentID
		if agentID == "" {
			// 从数据库查询
			agentInfo, err := s.grpcServer.AgentRepo().GetByHostID(ctx, hostID)
			if err == nil {
				agentID = agentInfo.AgentID
			}
		}
		if agentID != "" {
			agentYAML, err := s.renderAgentConfig(agentID, serverAddr)
			if err == nil {
				encoded := base64Encode(agentYAML)
				writeCmd := fmt.Sprintf("echo '%s' | base64 -d | sudo tee %s/agent.yaml > /dev/null", encoded, deployPath)
				client.Execute(writeCmd)
			}
		}
	}

	client.Execute("sudo systemctl start srehub-agent")

	// 清理临时目录
	client.Execute(fmt.Sprintf("rm -rf %s", tmpDir))

	appLogger.Info("Agent更新完成", zap.Uint("hostID", hostID))
	return nil
}

// uninstallAgentOnHost 卸载单台主机上的Agent
func (s *HTTPServer) uninstallAgentOnHost(ctx context.Context, hostID uint) error {
	hostVO, err := s.hostUseCase.GetByID(ctx, hostID)
	if err != nil {
		return fmt.Errorf("获取主机信息失败: %w", err)
	}

	credentialRepo := s.hostUseCase.GetCredentialRepo()
	credential, err := credentialRepo.GetByIDDecrypted(ctx, hostVO.CredentialID)
	if err != nil {
		return fmt.Errorf("获取凭证失败: %w", err)
	}

	var privateKey []byte
	if credential.PrivateKey != "" {
		privateKey = []byte(credential.PrivateKey)
	}
	client, err := sshclient.NewClient(hostVO.IP, hostVO.Port, hostVO.SSHUser, credential.Password, privateKey, credential.Passphrase)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	defer client.Close()

	deployPath := s.grpcServer.conf.Agent.DeployPath

	// 停止并禁用服务
	client.Execute("sudo systemctl stop srehub-agent")
	client.Execute("sudo systemctl disable srehub-agent")
	client.Execute("sudo rm -f /etc/systemd/system/srehub-agent.service")
	client.Execute("sudo systemctl daemon-reload")

	// 删除安装目录
	client.Execute(fmt.Sprintf("sudo rm -rf %s", deployPath))

	// 更新Host记录
	s.db.Exec("UPDATE hosts SET agent_id = '', agent_status = 'none', connection_mode = 'ssh' WHERE id = ?", hostID)

	// 删除AgentInfo记录
	s.grpcServer.AgentRepo().DeleteByHostID(ctx, hostID)

	appLogger.Info("Agent卸载完成", zap.Uint("hostID", hostID))
	return nil
}

// renderAgentConfig 生成Agent配置文件
func (s *HTTPServer) renderAgentConfig(agentID, serverAddr string) ([]byte, error) {
	deployPath := s.grpcServer.conf.Agent.DeployPath
	tmpl := `agent_id: "{{.AgentID}}"
server_addr: "{{.ServerAddr}}:{{.RPCPort}}"
cert_dir: "{{.DeployPath}}/certs"
log_file: "/var/log/srehub-agent.log"
`
	t, err := template.New("agent").Parse(tmpl)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]any{
		"AgentID":    agentID,
		"ServerAddr": serverAddr,
		"RPCPort":    s.grpcServer.conf.Server.RPCPort,
		"DeployPath": deployPath,
	})
	return buf.Bytes(), err
}

