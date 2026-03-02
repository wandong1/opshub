package inspection

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"github.com/ydcloud-dy/opshub/internal/biz/inspection/probers"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"gopkg.in/yaml.v3"
)

// ProbeConfigService handles HTTP requests for probe configs.
type ProbeConfigService struct {
	useCase          *biz.ProbeConfigUseCase
	agentFactory     biz.AgentCommandFactory
	variableResolver *biz.VariableResolver
}

func NewProbeConfigService(uc *biz.ProbeConfigUseCase) *ProbeConfigService {
	return &ProbeConfigService{useCase: uc}
}

// SetAgentCommandFactory injects Agent capability for RunOnce.
func (s *ProbeConfigService) SetAgentCommandFactory(f biz.AgentCommandFactory) {
	s.agentFactory = f
}

// SetVariableResolver injects variable resolver for RunOnce.
func (s *ProbeConfigService) SetVariableResolver(r *biz.VariableResolver) {
	s.variableResolver = r
}

func (s *ProbeConfigService) Create(c *gin.Context) {
	var req biz.ProbeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	model := req.ToModel()
	if err := s.useCase.Create(c.Request.Context(), model); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}
	response.Success(c, model)
}

func (s *ProbeConfigService) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req biz.ProbeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	model := req.ToModel()
	model.ID = uint(id)
	if err := s.useCase.Update(c.Request.Context(), model); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}
	response.Success(c, model)
}

func (s *ProbeConfigService) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}
	response.Success(c, nil)
}

func (s *ProbeConfigService) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	config, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "未找到: "+err.Error())
		return
	}
	response.Success(c, config)
}

func (s *ProbeConfigService) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	probeType := c.Query("type")
	category := c.Query("category")
	groupID, _ := strconv.ParseUint(c.Query("group_id"), 10, 64)
	var status *int8
	if s := c.Query("status"); s != "" {
		v, _ := strconv.ParseInt(s, 10, 8)
		sv := int8(v)
		status = &sv
	}

	configs, total, err := s.useCase.List(c.Request.Context(), page, pageSize, keyword, probeType, category, uint(groupID), status)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	response.Pagination(c, total, page, pageSize, configs)
}

func (s *ProbeConfigService) Import(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "请上传文件")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "读取文件失败")
		return
	}

	var items []biz.ProbeConfigImportExport
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &items); err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "YAML解析失败: "+err.Error())
			return
		}
	case ".json":
		if err := json.Unmarshal(data, &items); err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "JSON解析失败: "+err.Error())
			return
		}
	default:
		response.ErrorCode(c, http.StatusBadRequest, "仅支持 YAML/JSON 格式")
		return
	}

	configs := make([]*biz.ProbeConfig, 0, len(items))
	for _, item := range items {
		category := item.Category
		if category == "" {
			if cat, ok := biz.TypeToCategoryMap[item.Type]; ok {
				category = cat
			} else {
				category = biz.ProbeCategoryNetwork
			}
		}
		cfg := &biz.ProbeConfig{
			Name: item.Name, Type: item.Type, Category: category, Target: item.Target,
			Port: item.Port, Timeout: item.Timeout, Count: item.Count,
			PacketSize: item.PacketSize, Description: item.Description,
			Tags: item.Tags, GroupIDs: item.GroupIDs,
			ExecMode: item.ExecMode, AgentHostIDs: item.AgentHostIDs, RetryCount: item.RetryCount,
			SkipVerify: item.SkipVerify, WSMessage: item.WSMessage,
			WSMessageType: item.WSMessageType, WSReadTimeout: item.WSReadTimeout,
			Status: 1,
		}
		if cfg.Timeout == 0 {
			cfg.Timeout = 5
		}
		if cfg.Count == 0 {
			cfg.Count = 4
		}
		if cfg.PacketSize == 0 {
			cfg.PacketSize = 64
		}
		configs = append(configs, cfg)
	}

	if err := s.useCase.BatchCreate(c.Request.Context(), configs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "导入失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "导入成功", gin.H{"count": len(configs)})
}

func (s *ProbeConfigService) Export(c *gin.Context) {
	configs, err := s.useCase.ListAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "导出失败: "+err.Error())
		return
	}

	items := make([]biz.ProbeConfigImportExport, 0, len(configs))
	for _, cfg := range configs {
		items = append(items, biz.ProbeConfigImportExport{
			Name: cfg.Name, Type: cfg.Type, Category: cfg.Category, Target: cfg.Target,
			Port: cfg.Port, Timeout: cfg.Timeout, Count: cfg.Count,
			PacketSize: cfg.PacketSize, Description: cfg.Description,
			Tags: cfg.Tags, GroupIDs: cfg.GroupIDs,
			ExecMode: cfg.ExecMode, AgentHostIDs: cfg.AgentHostIDs, RetryCount: cfg.RetryCount,
			SkipVerify: cfg.SkipVerify, WSMessage: cfg.WSMessage,
			WSMessageType: cfg.WSMessageType, WSReadTimeout: cfg.WSReadTimeout,
		})
	}

	format := c.DefaultQuery("format", "yaml")
	switch format {
	case "json":
		c.Header("Content-Disposition", "attachment; filename=probe_configs.json")
		c.JSON(http.StatusOK, items)
	default:
		data, _ := yaml.Marshal(items)
		c.Header("Content-Disposition", "attachment; filename=probe_configs.yaml")
		c.Data(http.StatusOK, "application/x-yaml", data)
	}
}

func (s *ProbeConfigService) RunOnce(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	config, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "未找到: "+err.Error())
		return
	}

	// Variable resolution
	resolvedConfig := config
	if s.variableResolver != nil {
		if rc, err := s.variableResolver.ResolveConfig(c.Request.Context(), config); err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "变量替换失败: "+err.Error())
			return
		} else {
			resolvedConfig = rc
		}
	}

	// Application probe
	if resolvedConfig.Category == biz.ProbeCategoryApplication {
		s.runOnceApp(c, resolvedConfig, config)
		return
	}

	// Workflow probe
	if resolvedConfig.Category == biz.ProbeCategoryWorkflow {
		wfResult := biz.ExecuteWorkflowProbe(c.Request.Context(), resolvedConfig, s.variableResolver)
		response.Success(c, wfResult)
		return
	}

	// Network/Layer4 probe
	var result *probers.Result
	var agentHostID uint

	if resolvedConfig.ExecMode == biz.ExecModeAgent && s.agentFactory != nil {
		result, agentHostID = s.runOnceViaAgent(resolvedConfig)
	} else {
		prober, err := probers.GetProber(resolvedConfig.Type)
		if err != nil {
			response.ErrorCode(c, http.StatusBadRequest, err.Error())
			return
		}
		result = prober.Probe(resolvedConfig.Target, resolvedConfig.Port, resolvedConfig.Timeout, resolvedConfig.Count, resolvedConfig.PacketSize)
	}

	// Retry logic
	retryAttempt := 0
	for !result.Success && retryAttempt < config.RetryCount {
		retryAttempt++
		if resolvedConfig.ExecMode == biz.ExecModeAgent && s.agentFactory != nil {
			result, agentHostID = s.runOnceViaAgent(resolvedConfig)
		} else {
			prober, _ := probers.GetProber(resolvedConfig.Type)
			result = prober.Probe(resolvedConfig.Target, resolvedConfig.Port, resolvedConfig.Timeout, resolvedConfig.Count, resolvedConfig.PacketSize)
		}
	}

	response.Success(c, gin.H{
		"Success":         result.Success,
		"Latency":         result.Latency,
		"PacketLoss":      result.PacketLoss,
		"PingRttAvg":      result.PingRttAvg,
		"PingRttMin":      result.PingRttMin,
		"PingRttMax":      result.PingRttMax,
		"PingStddev":      result.PingStddev,
		"PingPacketsSent": result.PingPacketsSent,
		"PingPacketsRecv": result.PingPacketsRecv,
		"TCPConnectTime":  result.TCPConnectTime,
		"UDPWriteTime":    result.UDPWriteTime,
		"UDPReadTime":     result.UDPReadTime,
		"Error":           result.Error,
		"agentHostId":     agentHostID,
		"retryAttempt":    retryAttempt,
	})
}

// runOnceApp executes a single application probe and returns the result.
func (s *ProbeConfigService) runOnceApp(c *gin.Context, resolvedConfig, origConfig *biz.ProbeConfig) {
	appCfg := buildAppProbeConfig(resolvedConfig)
	prober, err := probers.GetAppProber(resolvedConfig.Type)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	appResult := prober.ProbeApp(appCfg)
	retryAttempt := 0
	for !appResult.Success && retryAttempt < origConfig.RetryCount {
		retryAttempt++
		appResult = prober.ProbeApp(appCfg)
	}

	response.Success(c, gin.H{
		"Success":           appResult.Success,
		"Latency":           appResult.Latency,
		"Error":             appResult.Error,
		"HTTPStatusCode":    appResult.HTTPStatusCode,
		"HTTPResponseTime":  appResult.HTTPResponseTime,
		"HTTPContentLength": appResult.HTTPContentLength,
		"AssertionSuccess":  appResult.AssertionSuccess,
		"AssertionResults":  appResult.AssertionResults,
		"ResponseBody":      appResult.ResponseBody,
		"ResponseHeaders":   appResult.ResponseHeaders,
		"retryAttempt":      retryAttempt,
	})
}

func buildAppProbeConfig(cfg *biz.ProbeConfig) *probers.AppProbeConfig {
	appCfg := &probers.AppProbeConfig{
		URL:           cfg.URL,
		Method:        cfg.Method,
		Body:          cfg.Body,
		ContentType:   cfg.ContentType,
		ProxyURL:      cfg.ProxyURL,
		Timeout:       cfg.Timeout,
		SkipVerify:    cfg.SkipVerify == nil || *cfg.SkipVerify,
		WSMessage:     cfg.WSMessage,
		WSMessageType: cfg.WSMessageType,
		WSReadTimeout: cfg.WSReadTimeout,
	}
	if appCfg.URL == "" {
		appCfg.URL = cfg.Target
	}
	if appCfg.Method == "" {
		appCfg.Method = "GET"
	}
	if cfg.Headers != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(cfg.Headers), &headers); err == nil {
			appCfg.Headers = headers
		}
	}
	if cfg.Params != "" {
		var params map[string]string
		if err := json.Unmarshal([]byte(cfg.Params), &params); err == nil {
			appCfg.Params = params
		}
	}
	if cfg.Assertions != "" {
		var assertions []probers.Assertion
		if err := json.Unmarshal([]byte(cfg.Assertions), &assertions); err == nil {
			appCfg.Assertions = assertions
		}
	}
	return appCfg
}

// runOnceViaAgent picks a random online agent and executes the probe.
func (s *ProbeConfigService) runOnceViaAgent(config *biz.ProbeConfig) (*probers.Result, uint) {
	hostIDs := parseHostIDs(config.AgentHostIDs)
	if len(hostIDs) == 0 {
		return &probers.Result{Error: "no agent host IDs configured"}, 0
	}

	var onlineIDs []uint
	for _, id := range hostIDs {
		if s.agentFactory.IsOnline(id) {
			onlineIDs = append(onlineIDs, id)
		}
	}
	if len(onlineIDs) == 0 {
		return &probers.Result{Error: "no online agent available"}, 0
	}

	hostID := onlineIDs[rand.Intn(len(onlineIDs))]
	executor, err := s.agentFactory.NewExecutor(hostID)
	if err != nil {
		return &probers.Result{Error: fmt.Sprintf("create agent executor: %v", err)}, hostID
	}

	prober, err := probers.GetAgentProber(config.Type, executor)
	if err != nil {
		return &probers.Result{Error: err.Error()}, hostID
	}
	return prober.Probe(config.Target, config.Port, config.Timeout, config.Count, config.PacketSize), hostID
}

// parseHostIDs parses a comma-separated string of host IDs.
func parseHostIDs(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	ids := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if id, err := strconv.ParseUint(p, 10, 64); err == nil && id > 0 {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

// TestProbe executes a probe from raw config data (without saving) for debugging.
func (s *ProbeConfigService) TestProbe(c *gin.Context) {
	var config biz.ProbeConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// Determine category from type if not set
	if config.Category == "" {
		if cat, ok := biz.TypeToCategoryMap[config.Type]; ok {
			config.Category = cat
		}
	}

	// Variable resolution
	resolvedConfig := &config
	if s.variableResolver != nil {
		if rc, err := s.variableResolver.ResolveConfig(c.Request.Context(), &config); err == nil {
			resolvedConfig = rc
		}
	}

	// Application probe
	if resolvedConfig.Category == biz.ProbeCategoryApplication {
		s.runOnceApp(c, resolvedConfig, &config)
		return
	}

	// Workflow probe
	if resolvedConfig.Category == biz.ProbeCategoryWorkflow {
		wfResult := biz.ExecuteWorkflowProbe(c.Request.Context(), resolvedConfig, s.variableResolver)
		response.Success(c, wfResult)
		return
	}

	// Network/Layer4 probe
	var result *probers.Result
	if resolvedConfig.ExecMode == biz.ExecModeAgent && s.agentFactory != nil {
		result, _ = s.runOnceViaAgent(resolvedConfig)
	} else {
		prober, err := probers.GetProber(resolvedConfig.Type)
		if err != nil {
			response.ErrorCode(c, http.StatusBadRequest, err.Error())
			return
		}
		result = prober.Probe(resolvedConfig.Target, resolvedConfig.Port, resolvedConfig.Timeout, resolvedConfig.Count, resolvedConfig.PacketSize)
	}

	response.Success(c, gin.H{
		"Success":         result.Success,
		"Latency":         result.Latency,
		"PacketLoss":      result.PacketLoss,
		"PingRttAvg":      result.PingRttAvg,
		"PingRttMin":      result.PingRttMin,
		"PingRttMax":      result.PingRttMax,
		"PingStddev":      result.PingStddev,
		"PingPacketsSent": result.PingPacketsSent,
		"PingPacketsRecv": result.PingPacketsRecv,
		"TCPConnectTime":  result.TCPConnectTime,
		"UDPWriteTime":    result.UDPWriteTime,
		"UDPReadTime":     result.UDPReadTime,
		"Error":           result.Error,
	})
}
