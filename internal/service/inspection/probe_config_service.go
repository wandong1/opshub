package inspection

import (
	"encoding/json"
	"io"
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
	useCase *biz.ProbeConfigUseCase
}

func NewProbeConfigService(uc *biz.ProbeConfigUseCase) *ProbeConfigService {
	return &ProbeConfigService{useCase: uc}
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
			Tags: item.Tags, Status: 1,
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
			Tags: cfg.Tags,
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

	prober, err := probers.GetProber(config.Type)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	result := prober.Probe(config.Target, config.Port, config.Timeout, config.Count, config.PacketSize)
	response.Success(c, result)
}
