package inspection_mgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	inspectionmgmtbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection_mgmt"
	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
)

// ItemAssertionOverride 巡检项断言覆盖结构
type ItemAssertionOverride struct {
	ItemID         uint   `json:"item_id"`
	AssertionType  string `json:"assertion_type"`
	AssertionValue string `json:"assertion_value"`
}

type ItemService struct {
	itemRepo         inspectionmgmtdata.ItemRepository
	groupRepo        inspectionmgmtdata.GroupRepository
	recordRepo       inspectionmgmtdata.RecordRepository
	hostRepo         assetbiz.HostRepo
	serviceLabelRepo assetbiz.ServiceLabelRepo
	cmdExecutor      *inspectionmgmtbiz.CommandExecutor
	probeExecutor    *ProbeExecutor
	validator        *inspectionmgmtbiz.AssertionValidator
	extractor        *inspectionmgmtbiz.VariableExtractor
	replacer         *VariableReplacer
	variableResolver *VariableResolver
}

func NewItemService(
	itemRepo inspectionmgmtdata.ItemRepository,
	groupRepo inspectionmgmtdata.GroupRepository,
	recordRepo inspectionmgmtdata.RecordRepository,
	hostRepo assetbiz.HostRepo,
	serviceLabelRepo assetbiz.ServiceLabelRepo,
	cmdExecutor *inspectionmgmtbiz.CommandExecutor,
	probeExecutor *ProbeExecutor,
	variableResolver *VariableResolver,
) *ItemService {
	return &ItemService{
		itemRepo:         itemRepo,
		groupRepo:        groupRepo,
		recordRepo:       recordRepo,
		hostRepo:         hostRepo,
		serviceLabelRepo: serviceLabelRepo,
		cmdExecutor:      cmdExecutor,
		probeExecutor:    probeExecutor,
		validator:        &inspectionmgmtbiz.AssertionValidator{},
		extractor:        &inspectionmgmtbiz.VariableExtractor{},
		replacer:         NewVariableReplacer(),
		variableResolver: variableResolver,
	}
}

func (s *ItemService) Create(ctx context.Context, req *ItemCreateRequest) (uint, error) {
	// 添加日志
	fmt.Printf("[DEBUG] Create InspectionItem:\n")
	fmt.Printf("  Name: %s\n", req.Name)
	fmt.Printf("  ExecutionType: %s\n", req.ExecutionType)
	fmt.Printf("  ScriptContent: %q\n", req.ScriptContent)
	fmt.Printf("  ScriptType: %s\n", req.ScriptType)
	fmt.Printf("  ScriptFile: %s\n", req.ScriptFile)
	fmt.Printf("  HostMatchType: %s\n", req.HostMatchType)
	fmt.Printf("  HostTags: %s\n", req.HostTags)
	fmt.Printf("  HostIDs: %s\n", req.HostIDs)

	item := &inspectionmgmtdata.InspectionItem{
		Name:              req.Name,
		Description:       req.Description,
		GroupID:           req.GroupID,
		Sort:              req.Sort,
		Status:            req.Status,
		ExecutionStrategy: req.ExecutionStrategy,
		ExecutionType:     req.ExecutionType,
		Command:           req.Command,
		ScriptType:        req.ScriptType,
		ScriptContent:     req.ScriptContent,
		ScriptFile:        req.ScriptFile,
		ScriptArgs:        req.ScriptArgs,
		PromQLQuery:       req.PromQLQuery,
		ProbeCategory:     req.ProbeCategory,
		ProbeType:         req.ProbeType,
		ProbeConfigID:     req.ProbeConfigID,
		HostMatchType:     req.HostMatchType,
		HostTags:          req.HostTags,
		HostIDs:           req.HostIDs,
		AssertionType:     req.AssertionType,
		AssertionValue:    req.AssertionValue,
		VariableName:      req.VariableName,
		VariableRegex:     req.VariableRegex,
		Timeout:           req.Timeout,
		InspectionLevel:   req.InspectionLevel,
		RiskLevel:         req.RiskLevel,
	}

	if item.Status == "" {
		item.Status = "enabled"
	}
	if item.ExecutionStrategy == "" {
		item.ExecutionStrategy = "concurrent"
	}
	if item.Timeout == 0 {
		item.Timeout = 60
	}

	fmt.Printf("[DEBUG] Before Create - item.HostTags: %s\n", item.HostTags)
	fmt.Printf("[DEBUG] Before Create - item.HostIDs: %s\n", item.HostIDs)

	if err := s.itemRepo.Create(ctx, item); err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (s *ItemService) Update(ctx context.Context, id uint, req *ItemUpdateRequest) error {
	item, err := s.itemRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if req.Name != "" {
		item.Name = req.Name
	}
	item.Description = req.Description
	if req.GroupID > 0 {
		item.GroupID = req.GroupID
	}
	item.Sort = req.Sort
	if req.Status != "" {
		item.Status = req.Status
	}
	if req.ExecutionStrategy != "" {
		item.ExecutionStrategy = req.ExecutionStrategy
	}
	if req.ExecutionType != "" {
		item.ExecutionType = req.ExecutionType
		// 切换执行类型时清空其他类型的字段
		switch req.ExecutionType {
		case "command":
			// 切换到命令：清空脚本、PromQL 和拨测字段
			item.ScriptType = ""
			item.ScriptContent = ""
			item.ScriptFile = ""
			item.ScriptArgs = ""
			item.PromQLQuery = ""
			item.ProbeCategory = ""
			item.ProbeType = ""
			item.ProbeConfigID = 0
		case "script":
			// 切换到脚本：清空命令、PromQL 和拨测字段
			item.Command = ""
			item.PromQLQuery = ""
			item.ProbeCategory = ""
			item.ProbeType = ""
			item.ProbeConfigID = 0
		case "promql":
			// 切换到 PromQL：清空命令、脚本和拨测字段
			item.Command = ""
			item.ScriptType = ""
			item.ScriptContent = ""
			item.ScriptFile = ""
			item.ScriptArgs = ""
			item.ProbeCategory = ""
			item.ProbeType = ""
			item.ProbeConfigID = 0
		case "probe":
			// 切换到拨测：清空命令、脚本和 PromQL 字段
			item.Command = ""
			item.ScriptType = ""
			item.ScriptContent = ""
			item.ScriptFile = ""
			item.ScriptArgs = ""
			item.PromQLQuery = ""
		}
	}
	// 只有在执行类型匹配时才更新对应字段
	if item.ExecutionType == "command" {
		item.Command = req.Command
	} else if item.ExecutionType == "script" {
		item.ScriptType = req.ScriptType
		item.ScriptContent = req.ScriptContent
		item.ScriptFile = req.ScriptFile
		item.ScriptArgs = req.ScriptArgs
	} else if item.ExecutionType == "promql" {
		item.PromQLQuery = req.PromQLQuery
	} else if item.ExecutionType == "probe" {
		item.ProbeCategory = req.ProbeCategory
		item.ProbeType = req.ProbeType
		item.ProbeConfigID = req.ProbeConfigID
	}
	item.HostMatchType = req.HostMatchType
	// 修复：即使是空字符串也要更新
	item.HostTags = req.HostTags
	item.HostIDs = req.HostIDs
	item.AssertionType = req.AssertionType
	item.AssertionValue = req.AssertionValue
	item.VariableName = req.VariableName
	item.VariableRegex = req.VariableRegex
	if req.Timeout > 0 {
		item.Timeout = req.Timeout
	}
	if req.InspectionLevel != "" {
		item.InspectionLevel = req.InspectionLevel
	}
	if req.RiskLevel != "" {
		item.RiskLevel = req.RiskLevel
	}

	return s.itemRepo.Update(ctx, item)
}

func (s *ItemService) Delete(ctx context.Context, id uint) error {
	return s.itemRepo.Delete(ctx, id)
}

func (s *ItemService) GetByID(ctx context.Context, id uint) (*ItemResponse, error) {
	item, err := s.itemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(item), nil
}

func (s *ItemService) List(ctx context.Context, req *ItemListRequest) ([]*ItemResponse, int64, error) {
	items, total, err := s.itemRepo.List(ctx, req.Page, req.PageSize, req.GroupID, req.Name, req.Status)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*ItemResponse, len(items))
	for i, item := range items {
		responses[i] = s.toResponse(item)
	}

	return responses, total, nil
}

func (s *ItemService) TestRun(ctx context.Context, req *TestRunRequest) ([]*TestRunResponse, error) {
	var results []*TestRunResponse

	for _, itemID := range req.ItemIDs {
		item, err := s.itemRepo.GetByID(ctx, itemID)
		if err != nil {
			continue
		}

		// 获取巡检组
		group, err := s.groupRepo.GetByID(ctx, item.GroupID)
		if err != nil {
			continue
		}

		// 拨测类型不需要主机匹配，直接执行
		if item.ExecutionType == "probe" {
			result := s.executeItem(ctx, item, group, nil, nil, nil)
			results = append(results, result)
			continue
		}

		// 匹配主机（非拨测类型）
		hosts, err := s.matchHosts(ctx, item, group)
		if err != nil {
			continue
		}

		// 预加载所有启用的服务标签
		enabledLabels, err := s.serviceLabelRepo.GetAllEnabled(ctx)
		if err != nil {
			continue
		}
		labelMap := make(map[string]*assetbiz.ServiceLabel)
		for _, label := range enabledLabels {
			labelMap[label.Name] = label
		}

		// 对每个主机执行巡检
		for _, host := range hosts {
			// 为每个主机创建独立的变量上下文
			variables := make(map[string]string)

			// 生成 instance 预设变量
			exporterPort := 9100
			if host.ExporterPort > 0 {
				exporterPort = host.ExporterPort
			}
			variables["instance"] = fmt.Sprintf("%s:%d", host.IP, exporterPort)

			// 生成 {label}_instance 预设变量
			if host.Tags != "" {
				hostTags := strings.Split(host.Tags, ",")

				// 解析主机级端口覆盖配置
				var labelPortOverrides map[string]int
				if host.LabelPortOverrides != "" {
					json.Unmarshal([]byte(host.LabelPortOverrides), &labelPortOverrides)
				}

				for _, tag := range hostTags {
					tag = strings.TrimSpace(tag)
					if label, exists := labelMap[tag]; exists && label.ExporterPort > 0 {
						labelInstanceVar := fmt.Sprintf("%s_instance", label.Name)

						// 优先级：主机级覆盖端口 > 服务标签默认端口
						port := label.ExporterPort
						if labelPortOverrides != nil {
							if overridePort, hasOverride := labelPortOverrides[label.Name]; hasOverride && overridePort > 0 {
								port = overridePort
							}
						}

						variables[labelInstanceVar] = fmt.Sprintf("%s:%d", host.IP, port)
					}
				}
			}

			result := s.executeItem(ctx, item, group, host, variables, nil)
			results = append(results, result)
		}
	}

	return results, nil
}

// TestRunWithoutSave 测试执行巡检项（不保存到数据库，用于前端测试）
func (s *ItemService) TestRunWithoutSave(ctx context.Context, groupID uint, items []ItemCreateRequest) ([]*TestRunResponse, error) {
	var results []*TestRunResponse

	// 获取巡检组
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("获取巡检组失败: %v", err)
	}

	for _, itemReq := range items {
		// 构造临时巡检项（不保存到数据库）
		item := &inspectionmgmtdata.InspectionItem{
			Name:              itemReq.Name,
			Description:       itemReq.Description,
			GroupID:           groupID,
			Sort:              itemReq.Sort,
			Status:            itemReq.Status,
			ExecutionStrategy: itemReq.ExecutionStrategy,
			ExecutionType:     itemReq.ExecutionType,
			Command:           itemReq.Command,
			ScriptType:        itemReq.ScriptType,
			ScriptContent:     itemReq.ScriptContent,
			ScriptFile:        itemReq.ScriptFile,
			ScriptArgs:        itemReq.ScriptArgs,
			PromQLQuery:       itemReq.PromQLQuery,
			ProbeCategory:     itemReq.ProbeCategory,
			ProbeType:         itemReq.ProbeType,
			ProbeConfigID:     itemReq.ProbeConfigID,
			HostMatchType:     itemReq.HostMatchType,
			HostTags:          itemReq.HostTags,
			HostIDs:           itemReq.HostIDs,
			AssertionType:     itemReq.AssertionType,
			AssertionValue:    itemReq.AssertionValue,
			VariableName:      itemReq.VariableName,
			VariableRegex:     itemReq.VariableRegex,
			Timeout:           itemReq.Timeout,
			InspectionLevel:   itemReq.InspectionLevel,
			RiskLevel:         itemReq.RiskLevel,
		}

		// 设置默认值
		if item.Status == "" {
			item.Status = "enabled"
		}
		if item.ExecutionStrategy == "" {
			item.ExecutionStrategy = "concurrent"
		}
		if item.Timeout == 0 {
			item.Timeout = 60
		}

		// 拨测类型不需要主机匹配，直接执行
		if item.ExecutionType == "probe" {
			result := s.executeItem(ctx, item, group, nil, nil, nil)
			results = append(results, result)
			continue
		}

		// 匹配主机（非拨测类型）
		hosts, err := s.matchHosts(ctx, item, group)
		if err != nil {
			continue
		}

		// 预加载所有启用的服务标签
		enabledLabels, err := s.serviceLabelRepo.GetAllEnabled(ctx)
		if err != nil {
			continue
		}
		labelMap := make(map[string]*assetbiz.ServiceLabel)
		for _, label := range enabledLabels {
			labelMap[label.Name] = label
		}

		// 对每个主机执行巡检
		for _, host := range hosts {
			// 为每个主机创建独立的变量上下文
			variables := make(map[string]string)

			// 生成 instance 预设变量
			exporterPort := 9100
			if host.ExporterPort > 0 {
				exporterPort = host.ExporterPort
			}
			variables["instance"] = fmt.Sprintf("%s:%d", host.IP, exporterPort)

			// 生成 {label}_instance 预设变量
			if host.Tags != "" {
				hostTags := strings.Split(host.Tags, ",")

				// 解析主机级端口覆盖配置
				var labelPortOverrides map[string]int
				if host.LabelPortOverrides != "" {
					json.Unmarshal([]byte(host.LabelPortOverrides), &labelPortOverrides)
				}

				for _, tag := range hostTags {
					tag = strings.TrimSpace(tag)
					if label, exists := labelMap[tag]; exists && label.ExporterPort > 0 {
						labelInstanceVar := fmt.Sprintf("%s_instance", label.Name)

						// 优先级：主机级覆盖端口 > 服务标签默认端口
						port := label.ExporterPort
						if labelPortOverrides != nil {
							if overridePort, hasOverride := labelPortOverrides[label.Name]; hasOverride && overridePort > 0 {
								port = overridePort
							}
						}

						variables[labelInstanceVar] = fmt.Sprintf("%s:%d", host.IP, port)
					}
				}
			}

			result := s.executeItem(ctx, item, group, host, variables, nil)
			results = append(results, result)
		}
	}

	return results, nil
}

func (s *ItemService) executeItem(
	ctx context.Context,
	item *inspectionmgmtdata.InspectionItem,
	group *inspectionmgmtdata.InspectionGroup,
	host *assetbiz.Host,
	runtimeVariables map[string]string,
	assertionOverride *ItemAssertionOverride,
) *TestRunResponse {
	result := &TestRunResponse{
		ItemID:          item.ID,
		ItemName:        item.Name,
		InspectionLevel: item.InspectionLevel,
		RiskLevel:       item.RiskLevel,
	}

	// 拨测类型不需要主机信息
	if host != nil {
		result.HostID = host.ID
		result.HostName = host.Name
		result.HostIp = host.IP
		fmt.Printf("[ItemService] executeItem - itemID: %d, itemName: %s, executionType: %s, hostID: %d\n",
			item.ID, item.Name, item.ExecutionType, host.ID)
	} else {
		fmt.Printf("[ItemService] executeItem - itemID: %d, itemName: %s, executionType: %s (no host)\n",
			item.ID, item.Name, item.ExecutionType)
	}

	// 解析变量（合并全局变量、巡检组自定义变量、运行时变量）
	var variables map[string]string
	if runtimeVariables == nil {
		variables = make(map[string]string)
	} else {
		variables = runtimeVariables
	}

	fmt.Printf("[ItemService] Runtime variables before resolve: %+v\n", runtimeVariables)

	// 尝试解析变量，如果失败则使用空变量继续执行
	if s.variableResolver != nil {
		resolvedVars, err := s.variableResolver.ResolveVariables(ctx, group.ID, runtimeVariables)
		if err != nil {
			fmt.Printf("[ItemService] Failed to resolve variables: %v, continuing with empty variables\n", err)
		} else {
			variables = resolvedVars
			fmt.Printf("[ItemService] Variables after resolve: %+v\n", variables)
		}
	}

	var execResult *inspectionmgmtbiz.ExecuteResult

	// 根据执行类型选择执行方式
	switch item.ExecutionType {
	case "command":
		// 命令执行 - 替换变量
		fmt.Printf("[ItemService] Executing command: %s\n", item.Command)
		command := s.replacer.ReplaceCommand(item.Command, variables)
		fmt.Printf("[ItemService] Command after variable replacement: %s\n", command)
		execResult = s.cmdExecutor.Execute(ctx, host, command, group.ExecutionMode, item.Timeout)

	case "script":
		// 脚本执行 - 替换脚本内容中的变量
		fmt.Printf("[ItemService] Executing script - type: %s, hasContent: %v, hasFile: %v, args: %s\n",
			item.ScriptType, item.ScriptContent != "", item.ScriptFile != "", item.ScriptArgs)

		scriptContent := s.replacer.ReplaceScriptContent(item.ScriptContent, variables)
		scriptArgs := s.replacer.Replace(item.ScriptArgs, variables)

		scriptReq := &inspectionmgmtbiz.ScriptExecuteRequest{
			ScriptType:    item.ScriptType,
			ScriptContent: scriptContent,
			ScriptFile:    item.ScriptFile,
			ScriptArgs:    scriptArgs,
		}
		execResult = s.cmdExecutor.ExecuteScript(ctx, host, scriptReq, group.ExecutionMode, item.Timeout)

	case "promql":
		// PromQL 查询 - 替换查询中的变量
		fmt.Printf("[ItemService] Executing PromQL: %s\n", item.PromQLQuery)
		promql := s.replacer.ReplacePromQL(item.PromQLQuery, variables)
		fmt.Printf("[ItemService] PromQL after variable replacement: %s\n", promql)

		// PromQL 执行（暂不实现）
		execResult = &inspectionmgmtbiz.ExecuteResult{
			Output:   "",
			Error:    fmt.Errorf("PromQL 执行暂未实现"),
			Duration: 0,
		}

	case "probe":
		// 拨测执行
		fmt.Printf("[ItemService] Executing probe - configID: %d\n", item.ProbeConfigID)
		execResult = s.probeExecutor.Execute(ctx, item.ProbeConfigID, item.Timeout)

	default:
		fmt.Printf("[ItemService] Unknown execution type: %s\n", item.ExecutionType)
		execResult = &inspectionmgmtbiz.ExecuteResult{
			Output:   "",
			Error:    fmt.Errorf("未知的执行类型: %s", item.ExecutionType),
			Duration: 0,
		}
	}

	result.Duration = execResult.Duration
	result.Output = execResult.Output

	if execResult.Error != nil {
		result.Status = "failed"
		result.ErrorMessage = execResult.Error.Error()
		result.AssertionResult = "skip"
		fmt.Printf("[ItemService] Execution failed - error: %v\n", execResult.Error)
		return result
	}

	result.Status = "success"
	fmt.Printf("[ItemService] Execution success - output length: %d\n", len(execResult.Output))

	// 应用断言覆盖（如果提供）
	effectiveAssertionType := item.AssertionType
	effectiveAssertionValue := item.AssertionValue
	if assertionOverride != nil {
		effectiveAssertionType = assertionOverride.AssertionType
		effectiveAssertionValue = assertionOverride.AssertionValue
		fmt.Printf("[ItemService] 应用断言覆盖 - item_id: %d, override_type: %s, override_value: %s\n",
			item.ID, effectiveAssertionType, effectiveAssertionValue)
	}

	// 断言校验
	assertionResult := s.validator.Validate(effectiveAssertionType, effectiveAssertionValue, execResult.Output)
	if assertionResult.Pass {
		result.AssertionResult = "pass"
	} else {
		result.AssertionResult = "fail"
	}
	result.AssertionDetails = map[string]interface{}{
		"pass":    assertionResult.Pass,
		"message": assertionResult.Message,
	}

	return result
}

func (s *ItemService) matchHosts(ctx context.Context, item *inspectionmgmtdata.InspectionItem, group *inspectionmgmtdata.InspectionGroup) ([]*assetbiz.Host, error) {
	// 解析关联分组 ID
	var groupIDs []uint
	if group.GroupIDs != "" {
		if err := json.Unmarshal([]byte(group.GroupIDs), &groupIDs); err != nil {
			return nil, fmt.Errorf("解析分组 ID 失败: %v", err)
		}
	}

	if len(groupIDs) == 0 {
		return nil, fmt.Errorf("未配置关联分组")
	}

	// 获取所有分组的主机
	var allHosts []*assetbiz.Host
	for _, gid := range groupIDs {
		hosts, err := s.hostRepo.GetByGroupID(ctx, gid)
		if err != nil {
			continue
		}
		allHosts = append(allHosts, hosts...)
	}

	// 根据匹配类型过滤主机
	if item.HostMatchType == "id" && item.HostIDs != "" {
		var hostIDs []uint
		if err := json.Unmarshal([]byte(item.HostIDs), &hostIDs); err != nil {
			return nil, fmt.Errorf("解析主机 ID 失败: %v", err)
		}
		return s.filterHostsByIDs(allHosts, hostIDs), nil
	}

	if item.HostMatchType == "tag" && item.HostTags != "" {
		var tags []string
		if err := json.Unmarshal([]byte(item.HostTags), &tags); err != nil {
			return nil, fmt.Errorf("解析主机标签失败: %v", err)
		}
		return s.filterHostsByTags(allHosts, tags), nil
	}

	if item.HostMatchType == "name" && item.HostTags != "" {
		var keywords []string
		if err := json.Unmarshal([]byte(item.HostTags), &keywords); err != nil {
			return nil, fmt.Errorf("解析主机名关键词失败: %v", err)
		}
		return s.filterHostsByNames(allHosts, keywords), nil
	}

	return allHosts, nil
}

func (s *ItemService) filterHostsByIDs(hosts []*assetbiz.Host, ids []uint) []*assetbiz.Host {
	idMap := make(map[uint]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	var filtered []*assetbiz.Host
	for _, host := range hosts {
		if idMap[host.ID] {
			filtered = append(filtered, host)
		}
	}
	return filtered
}

func (s *ItemService) filterHostsByTags(hosts []*assetbiz.Host, tags []string) []*assetbiz.Host {
	var filtered []*assetbiz.Host
	for _, host := range hosts {
		if s.hostHasTags(host, tags) {
			filtered = append(filtered, host)
		}
	}
	return filtered
}

func (s *ItemService) filterHostsByNames(hosts []*assetbiz.Host, keywords []string) []*assetbiz.Host {
	var filtered []*assetbiz.Host
	for _, host := range hosts {
		if s.hostMatchesKeywords(host, keywords) {
			filtered = append(filtered, host)
		}
	}
	return filtered
}

func (s *ItemService) hostMatchesKeywords(host *assetbiz.Host, keywords []string) bool {
	if host.Name == "" {
		return false
	}

	for _, keyword := range keywords {
		if keyword != "" && containsIgnoreCase(host.Name, keyword) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *ItemService) hostHasTags(host *assetbiz.Host, tags []string) bool {
	if host.Tags == "" {
		return false
	}

	hostTags := make(map[string]bool)
	for _, tag := range splitTags(host.Tags) {
		hostTags[tag] = true
	}

	for _, tag := range tags {
		if hostTags[tag] {
			return true
		}
	}
	return false
}

func splitTags(tags string) []string {
	var result []string
	for _, tag := range splitByComma(tags) {
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

func splitByComma(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func (s *ItemService) toResponse(item *inspectionmgmtdata.InspectionItem) *ItemResponse {
	return &ItemResponse{
		ID:                item.ID,
		Name:              item.Name,
		Description:       item.Description,
		GroupID:           item.GroupID,
		Sort:              item.Sort,
		Status:            item.Status,
		ExecutionStrategy: item.ExecutionStrategy,
		ExecutionType:     item.ExecutionType,
		Command:           item.Command,
		ScriptType:        item.ScriptType,
		ScriptContent:     item.ScriptContent,
		ScriptFile:        item.ScriptFile,
		ScriptArgs:        item.ScriptArgs,
		PromQLQuery:       item.PromQLQuery,
		ProbeCategory:     item.ProbeCategory,
		ProbeType:         item.ProbeType,
		ProbeConfigID:     item.ProbeConfigID,
		HostMatchType:     item.HostMatchType,
		HostTags:          item.HostTags,
		HostIDs:           item.HostIDs,
		AssertionType:     item.AssertionType,
		AssertionValue:    item.AssertionValue,
		VariableName:      item.VariableName,
		VariableRegex:     item.VariableRegex,
		Timeout:           item.Timeout,
		InspectionLevel:   item.InspectionLevel,
		RiskLevel:         item.RiskLevel,
		CreatedAt:         item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         item.UpdatedAt.Format(time.RFC3339),
	}
}

// BatchSave 批量保存巡检项
func (s *ItemService) BatchSave(ctx context.Context, groupID uint, items []ItemCreateRequest) error {
	// 1. 获取该组现有的所有巡检项
	existingItems, err := s.itemRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("获取现有巡检项失败: %v", err)
	}

	// 构建现有巡检项 ID 集合
	existingIDs := make(map[uint]bool)
	for _, item := range existingItems {
		existingIDs[item.ID] = true
	}

	// 2. 处理前端传来的巡检项（更新或创建）
	submittedIDs := make(map[uint]bool)
	for _, req := range items {
		req.GroupID = groupID // 确保 GroupID 正确

		if req.ID != nil && *req.ID > 0 && existingIDs[*req.ID] {
			// 更新现有巡检项（保留 ID）
			submittedIDs[*req.ID] = true
			updateReq := &ItemUpdateRequest{
				Name:              req.Name,
				Description:       req.Description,
				GroupID:           req.GroupID,
				Sort:              req.Sort,
				Status:            req.Status,
				ExecutionStrategy: req.ExecutionStrategy,
				ExecutionType:     req.ExecutionType,
				Command:           req.Command,
				ScriptType:        req.ScriptType,
				ScriptContent:     req.ScriptContent,
				ScriptFile:        req.ScriptFile,
				ScriptArgs:        req.ScriptArgs,
				PromQLQuery:       req.PromQLQuery,
				ProbeCategory:     req.ProbeCategory,
				ProbeType:         req.ProbeType,
				ProbeConfigID:     req.ProbeConfigID,
				HostMatchType:     req.HostMatchType,
				HostTags:          req.HostTags,
				HostIDs:           req.HostIDs,
				AssertionType:     req.AssertionType,
				AssertionValue:    req.AssertionValue,
				VariableName:      req.VariableName,
				VariableRegex:     req.VariableRegex,
				Timeout:           req.Timeout,
				InspectionLevel:   req.InspectionLevel,
				RiskLevel:         req.RiskLevel,
			}
			if err := s.Update(ctx, *req.ID, updateReq); err != nil {
				return fmt.Errorf("更新巡检项 %d 失败: %v", *req.ID, err)
			}
		} else {
			// 创建新巡检项
			if _, err := s.Create(ctx, &req); err != nil {
				return fmt.Errorf("创建巡检项失败: %v", err)
			}
		}
	}

	// 3. 删除前端未传递的巡检项（用户已删除）
	for id := range existingIDs {
		if !submittedIDs[id] {
			if err := s.itemRepo.Delete(ctx, id); err != nil {
				return fmt.Errorf("删除巡检项 %d 失败: %v", id, err)
			}
		}
	}

	return nil
}

// ExecuteItemByID 根据巡检项ID执行巡检（用于调度器）
func (s *ItemService) ExecuteItemByID(ctx context.Context, itemID uint) ([]*inspectionmgmtdata.InspectionRecord, error) {
	// 获取巡检项
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get inspection item failed: %w", err)
	}

	// 获取巡检组
	group, err := s.groupRepo.GetByID(ctx, item.GroupID)
	if err != nil {
		return nil, fmt.Errorf("get inspection group failed: %w", err)
	}

	// 拨测类型不需要主机匹配，直接执行
	if item.ExecutionType == "probe" {
		variables := make(map[string]string)
		result := s.executeItem(ctx, item, group, nil, variables, nil)

		// 转换 AssertionDetails 为 JSON 字符串
		assertionDetailsJSON := ""
		if result.AssertionDetails != nil {
			if data, err := json.Marshal(result.AssertionDetails); err == nil {
				assertionDetailsJSON = string(data)
			}
		}

		// 保存执行记录（拨测类型 HostID 为 0）
		record := &inspectionmgmtdata.InspectionRecord{
			GroupID:          item.GroupID,
			ItemID:           item.ID,
			HostID:           0, // 拨测类型不关联主机
			Status:           result.Status,
			Output:           result.Output,
			ErrorMessage:     result.ErrorMessage,
			Duration:         result.Duration,
			AssertionResult:  result.AssertionResult,
			AssertionDetails: assertionDetailsJSON,
			ExecutedAt:       time.Now(),
		}

		if err := s.recordRepo.Create(ctx, record); err != nil {
			return nil, fmt.Errorf("save inspection record failed: %w", err)
		}

		return []*inspectionmgmtdata.InspectionRecord{record}, nil
	}

	// 获取目标主机列表（非拨测类型）
	hosts, err := s.matchHosts(ctx, item, group)
	if err != nil {
		return nil, fmt.Errorf("match hosts failed: %w", err)
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("no target hosts found for item %d", itemID)
	}

	// 预加载所有启用的服务标签
	enabledLabels, err := s.serviceLabelRepo.GetAllEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("load service labels failed: %w", err)
	}
	labelMap := make(map[string]*assetbiz.ServiceLabel)
	for _, label := range enabledLabels {
		labelMap[label.Name] = label
	}

	// 执行巡检并保存记录
	records := make([]*inspectionmgmtdata.InspectionRecord, 0, len(hosts))

	for _, host := range hosts {
		// 为每个主机创建独立的变量上下文
		variables := make(map[string]string)

		// 生成 instance 预设变量
		exporterPort := 9100
		if host.ExporterPort > 0 {
			exporterPort = host.ExporterPort
		}
		variables["instance"] = fmt.Sprintf("%s:%d", host.IP, exporterPort)

		// 生成 {label}_instance 预设变量
		if host.Tags != "" {
			hostTags := strings.Split(host.Tags, ",")

			// 解析主机级端口覆盖配置
			var labelPortOverrides map[string]int
			if host.LabelPortOverrides != "" {
				json.Unmarshal([]byte(host.LabelPortOverrides), &labelPortOverrides)
			}

			for _, tag := range hostTags {
				tag = strings.TrimSpace(tag)
				if label, exists := labelMap[tag]; exists && label.ExporterPort > 0 {
					labelInstanceVar := fmt.Sprintf("%s_instance", label.Name)

					// 优先级：主机级覆盖端口 > 服务标签默认端口
					port := label.ExporterPort
					if labelPortOverrides != nil {
						if overridePort, hasOverride := labelPortOverrides[label.Name]; hasOverride && overridePort > 0 {
							port = overridePort
						}
					}

					variables[labelInstanceVar] = fmt.Sprintf("%s:%d", host.IP, port)
				}
			}
		}

		result := s.executeItem(ctx, item, group, host, variables, nil)

		// 转换 AssertionDetails 为 JSON 字符串
		assertionDetailsJSON := ""
		if result.AssertionDetails != nil {
			if data, err := json.Marshal(result.AssertionDetails); err == nil {
				assertionDetailsJSON = string(data)
			}
		}

		// 保存执行记录
		record := &inspectionmgmtdata.InspectionRecord{
			GroupID:          item.GroupID,
			ItemID:           item.ID,
			HostID:           host.ID,
			Status:           result.Status,
			Output:           result.Output,
			ErrorMessage:     result.ErrorMessage,
			Duration:         result.Duration,
			AssertionResult:  result.AssertionResult,
			AssertionDetails: assertionDetailsJSON,
			ExecutedAt:       time.Now(),
		}

		if err := s.recordRepo.Create(ctx, record); err != nil {
			return nil, fmt.Errorf("save inspection record failed: %w", err)
		}

		records = append(records, record)
	}

	return records, nil
}

// ExecuteItemByIDWithOverride 执行巡检项，支持任务调度级覆盖（需求一、需求三、断言覆盖）
// executionModeOverride: 覆盖执行方式，空=沿用巡检组配置
// businessGroupIDOverride: 覆盖业务分组，0=沿用巡检组配置
// extraVars: 任务调度级自定义变量（优先级最高，合并到运行时变量）
// assertionOverride: 断言覆盖，nil=沿用巡检项配置
func (s *ItemService) ExecuteItemByIDWithOverride(
	ctx context.Context,
	itemID uint,
	executionModeOverride string,
	businessGroupIDOverride uint,
	extraVars map[string]string,
	assertionOverride *ItemAssertionOverride,
) ([]*inspectionmgmtdata.InspectionRecord, error) {
	// 获取巡检项
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get inspection item failed: %w", err)
	}

	// 获取巡检组
	group, err := s.groupRepo.GetByID(ctx, item.GroupID)
	if err != nil {
		return nil, fmt.Errorf("get inspection group failed: %w", err)
	}

	// 需求一：运行时覆盖巡检组配置（不修改数据库）
	effectiveGroup := *group
	if executionModeOverride != "" {
		effectiveGroup.ExecutionMode = executionModeOverride
	}
	if businessGroupIDOverride > 0 {
		// 将业务分组 ID 序列化为 JSON 数组格式
		groupIDsJSON, _ := json.Marshal([]uint{businessGroupIDOverride})
		effectiveGroup.GroupIDs = string(groupIDsJSON)
	}

	// 需求三：将任务调度自定义变量（最高优先级）作为初始运行时变量
	baseVars := make(map[string]string)
	for k, v := range extraVars {
		baseVars[k] = v
	}

	// 拨测类型不需要主机匹配，直接执行
	if item.ExecutionType == "probe" {
		variables := make(map[string]string)
		for k, v := range baseVars {
			variables[k] = v
		}
		result := s.executeItem(ctx, item, &effectiveGroup, nil, variables, assertionOverride)

		assertionDetailsJSON := ""
		if result.AssertionDetails != nil {
			if data, err := json.Marshal(result.AssertionDetails); err == nil {
				assertionDetailsJSON = string(data)
			}
		}

		record := &inspectionmgmtdata.InspectionRecord{
			GroupID:          item.GroupID,
			ItemID:           item.ID,
			HostID:           0,
			Status:           result.Status,
			Output:           result.Output,
			ErrorMessage:     result.ErrorMessage,
			Duration:         result.Duration,
			AssertionResult:  result.AssertionResult,
			AssertionDetails: assertionDetailsJSON,
			ExecutedAt:       time.Now(),
		}
		if err := s.recordRepo.Create(ctx, record); err != nil {
			return nil, fmt.Errorf("save inspection record failed: %w", err)
		}
		return []*inspectionmgmtdata.InspectionRecord{record}, nil
	}

	// 获取目标主机列表（使用覆盖后的 group）
	hosts, err := s.matchHosts(ctx, item, &effectiveGroup)
	if err != nil {
		return nil, fmt.Errorf("match hosts failed: %w", err)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no target hosts found for item %d", itemID)
	}

	// 预加载所有启用的服务标签
	enabledLabels, err := s.serviceLabelRepo.GetAllEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("load service labels failed: %w", err)
	}
	labelMap := make(map[string]*assetbiz.ServiceLabel)
	for _, label := range enabledLabels {
		labelMap[label.Name] = label
	}

	records := make([]*inspectionmgmtdata.InspectionRecord, 0, len(hosts))
	for _, host := range hosts {
		// 为每个主机创建独立的变量上下文
		variables := make(map[string]string)

		// 复制任务变量
		for k, v := range baseVars {
			variables[k] = v
		}

		// 生成 instance 预设变量
		exporterPort := 9100
		if host.ExporterPort > 0 {
			exporterPort = host.ExporterPort
		}
		variables["instance"] = fmt.Sprintf("%s:%d", host.IP, exporterPort)

		// 生成 {label}_instance 预设变量
		if host.Tags != "" {
			hostTags := strings.Split(host.Tags, ",")

			// 解析主机级端口覆盖配置
			var labelPortOverrides map[string]int
			if host.LabelPortOverrides != "" {
				json.Unmarshal([]byte(host.LabelPortOverrides), &labelPortOverrides)
			}

			for _, tag := range hostTags {
				tag = strings.TrimSpace(tag)
				if label, exists := labelMap[tag]; exists && label.ExporterPort > 0 {
					labelInstanceVar := fmt.Sprintf("%s_instance", label.Name)

					// 优先级：主机级覆盖端口 > 服务标签默认端口
					port := label.ExporterPort
					if labelPortOverrides != nil {
						if overridePort, hasOverride := labelPortOverrides[label.Name]; hasOverride && overridePort > 0 {
							port = overridePort
						}
					}

					variables[labelInstanceVar] = fmt.Sprintf("%s:%d", host.IP, port)
				}
			}
		}

		result := s.executeItem(ctx, item, &effectiveGroup, host, variables, assertionOverride)

		assertionDetailsJSON := ""
		if result.AssertionDetails != nil {
			if data, err := json.Marshal(result.AssertionDetails); err == nil {
				assertionDetailsJSON = string(data)
			}
		}

		record := &inspectionmgmtdata.InspectionRecord{
			GroupID:          item.GroupID,
			ItemID:           item.ID,
			HostID:           host.ID,
			HostName:         host.Name,
			HostIP:           host.IP,
			Status:           result.Status,
			Output:           result.Output,
			ErrorMessage:     result.ErrorMessage,
			Duration:         result.Duration,
			AssertionResult:  result.AssertionResult,
			AssertionDetails: assertionDetailsJSON,
			ExecutedAt:       time.Now(),
		}
		if err := s.recordRepo.Create(ctx, record); err != nil {
			return nil, fmt.Errorf("save inspection record failed: %w", err)
		}
		records = append(records, record)
	}
	return records, nil
}
