package alert

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// --- Rule Groups ---

func (s *HTTPServer) listRuleGroups(c *gin.Context) {
	assetGroupID, _ := strconv.ParseUint(c.Query("assetGroupId"), 10, 64)
	list, err := s.ruleGroupRepo.ListByAssetGroup(c.Request.Context(), uint(assetGroupID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, list)
}

func (s *HTTPServer) createRuleGroup(c *gin.Context) {
	var req biz.AlertRuleGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ruleGroupRepo.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, req)
}

func (s *HTTPServer) getRuleGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	g, err := s.ruleGroupRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "规则分类不存在")
		return
	}
	response.Success(c, g)
}

func (s *HTTPServer) updateRuleGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req biz.AlertRuleGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	req.ID = uint(id)
	if err := s.ruleGroupRepo.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}
	response.Success(c, req)
}

func (s *HTTPServer) deleteRuleGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.ruleGroupRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}

// --- Rules ---

func (s *HTTPServer) listRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	assetGroupID, _ := strconv.ParseUint(c.Query("assetGroupId"), 10, 64)
	ruleGroupID, _ := strconv.ParseUint(c.Query("ruleGroupId"), 10, 64)
	keyword := c.Query("keyword")

	var enabled *bool
	if v := c.Query("enabled"); v != "" {
		b := v == "true"
		enabled = &b
	}

	list, total, err := s.ruleRepo.List(c.Request.Context(), page, pageSize, uint(assetGroupID), uint(ruleGroupID), keyword, enabled)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}

	// 从缓存回填评估时间
	if s.evalEngine != nil {
		if err := s.ruleRepo.FillEvalTimesFromCache(c.Request.Context(), list, s.evalEngine.GetEvalCache()); err != nil {
			// 回填失败不影响主流程，仅记录日志
			c.Error(err)
		}
	}

	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": list})
}

func (s *HTTPServer) createRule(c *gin.Context) {
	var req biz.AlertRule
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.EvalInterval <= 0 {
		req.EvalInterval = 15
	}
	if err := s.ruleRepo.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}

	// 发布规则重载事件
	if s.evalEngine != nil {
		if err := s.evalEngine.GetRuleCache().PublishReloadEvent(c.Request.Context()); err != nil {
			c.Error(err)
		}
	}

	response.Success(c, req)
}

func (s *HTTPServer) getRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rule, err := s.ruleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "规则不存在")
		return
	}
	response.Success(c, rule)
}

func (s *HTTPServer) updateRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// 先检查规则是否存在
	_, err := s.ruleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "规则不存在")
		return
	}

	// 解析请求体到 map，只更新传递的字段
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 使用 GORM 的 Updates 方法只更新传递的字段
	if err := s.ruleRepo.UpdateFields(c.Request.Context(), uint(id), updates); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}

	// 发布规则重载事件
	if s.evalEngine != nil {
		if err := s.evalEngine.GetRuleCache().PublishReloadEvent(c.Request.Context()); err != nil {
			c.Error(err)
		}
	}

	// 重新获取更新后的规则
	updatedRule, _ := s.ruleRepo.GetByID(c.Request.Context(), uint(id))
	response.Success(c, updatedRule)
}

func (s *HTTPServer) deleteRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// 先将该规则的所有告警中的告警标记为已恢复（与禁用规则逻辑相同）
	if err := s.eventRepo.ResolveActiveByRuleID(c.Request.Context(), uint(id)); err != nil {
		// 记录错误但不影响后续操作
		c.Error(err)
	}

	// 删除规则
	if err := s.ruleRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}

	// 发布规则重载事件
	if s.evalEngine != nil {
		if err := s.evalEngine.GetRuleCache().PublishReloadEvent(c.Request.Context()); err != nil {
			c.Error(err)
		}
	}

	response.Success(c, nil)
}

func (s *HTTPServer) toggleRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rule, err := s.ruleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "规则不存在")
		return
	}

	// 如果是禁用规则，将该规则的所有告警中的告警标记为已恢复
	if rule.Enabled {
		if err := s.eventRepo.ResolveActiveByRuleID(c.Request.Context(), uint(id)); err != nil {
			// 记录错误但不影响规则禁用操作
			c.Error(err)
		}
	}

	rule.Enabled = !rule.Enabled
	if err := s.ruleRepo.Update(c.Request.Context(), rule); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "操作失败")
		return
	}

	// 发布规则重载事件
	if s.evalEngine != nil {
		if err := s.evalEngine.GetRuleCache().PublishReloadEvent(c.Request.Context()); err != nil {
			c.Error(err)
		}
	}

	response.Success(c, gin.H{"enabled": rule.Enabled})
}

func (s *HTTPServer) testRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rule, err := s.ruleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "规则不存在")
		return
	}
	results, err := s.evalEngine.EvalRuleOnce(c.Request.Context(), rule)
	if err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "查询失败: "+err.Error())
		return
	}
	firing := len(results) > 0
	response.Success(c, gin.H{"firing": firing, "results": results})
}

func (s *HTTPServer) cloneRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rule, err := s.ruleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "规则不存在")
		return
	}
	newRule := *rule
	newRule.ID = 0
	newRule.Name = rule.Name + "_copy"
	newRule.Enabled = false
	newRule.LastEvalAt = nil
	if err := s.ruleRepo.Create(c.Request.Context(), &newRule); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "克隆失败")
		return
	}
	response.Success(c, newRule)
}

func (s *HTTPServer) importRules(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "请上传文件")
		return
	}
	f, err := file.Open()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "文件读取失败")
		return
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	f.Read(buf)

	var rules []biz.AlertRule
	// 尝试 JSON，失败则 YAML
	if err := json.Unmarshal(buf, &rules); err != nil {
		if err2 := yaml.Unmarshal(buf, &rules); err2 != nil {
			response.ErrorCode(c, http.StatusBadRequest, "文件格式错误，支持 JSON 和 YAML")
			return
		}
	}

	success := 0
	for i := range rules {
		rules[i].ID = 0
		rules[i].Enabled = false
		if rules[i].EvalInterval <= 0 {
			rules[i].EvalInterval = 15
		}
		if err := s.ruleRepo.Create(c.Request.Context(), &rules[i]); err == nil {
			success++
		}
	}

	// 发布规则重载事件
	if s.evalEngine != nil && success > 0 {
		if err := s.evalEngine.GetRuleCache().PublishReloadEvent(c.Request.Context()); err != nil {
			c.Error(err)
		}
	}

	response.Success(c, gin.H{"imported": success, "total": len(rules)})
}

func (s *HTTPServer) exportRules(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	idsStr := c.QueryArray("ids")

	var rules []*biz.AlertRule
	var err error
	if len(idsStr) > 0 {
		var ids []uint
		for _, s := range idsStr {
			id, _ := strconv.ParseUint(s, 10, 64)
			ids = append(ids, uint(id))
		}
		rules, err = s.ruleRepo.ListByIDs(c.Request.Context(), ids)
	} else {
		rules, _, err = s.ruleRepo.List(c.Request.Context(), 1, 10000, 0, 0, "", nil)
	}
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}

	switch format {
	case "yaml":
		data, _ := yaml.Marshal(rules)
		c.Header("Content-Disposition", "attachment; filename=alert_rules.yaml")
		c.Data(http.StatusOK, "application/x-yaml", data)
	default:
		data, _ := json.MarshalIndent(rules, "", "  ")
		c.Header("Content-Disposition", "attachment; filename=alert_rules.json")
		c.Data(http.StatusOK, "application/json", data)
	}
}

func (s *HTTPServer) adhocTestRule(c *gin.Context) {
	var req struct {
		DataSourceIDs []uint `json:"dataSourceIds"`
		DataSourceID  uint   `json:"dataSourceId"`
		Expr          string `json:"expr"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	dsIDs := req.DataSourceIDs
	if len(dsIDs) == 0 && req.DataSourceID > 0 {
		dsIDs = []uint{req.DataSourceID}
	}
	if len(dsIDs) == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请选择数据源")
		return
	}
	if req.Expr == "" {
		response.ErrorCode(c, http.StatusBadRequest, "请输入 PromQL 表达式")
		return
	}
	results, err := s.evalEngine.EvalExprOnDatasources(c.Request.Context(), dsIDs, req.Expr)
	if err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "查询失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"firing": len(results) > 0, "results": results})
}

func init() {
	_ = fmt.Sprintf
}
