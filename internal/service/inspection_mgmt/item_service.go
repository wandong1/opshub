package inspection_mgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	inspectionmgmtbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection_mgmt"
	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
)

type ItemService struct {
	itemRepo    inspectionmgmtdata.ItemRepository
	groupRepo   inspectionmgmtdata.GroupRepository
	recordRepo  inspectionmgmtdata.RecordRepository
	hostRepo    assetbiz.HostRepo
	cmdExecutor *inspectionmgmtbiz.CommandExecutor
	validator   *inspectionmgmtbiz.AssertionValidator
	extractor   *inspectionmgmtbiz.VariableExtractor
	replacer    *inspectionmgmtbiz.VariableReplacer
}

func NewItemService(
	itemRepo inspectionmgmtdata.ItemRepository,
	groupRepo inspectionmgmtdata.GroupRepository,
	recordRepo inspectionmgmtdata.RecordRepository,
	hostRepo assetbiz.HostRepo,
	cmdExecutor *inspectionmgmtbiz.CommandExecutor,
) *ItemService {
	return &ItemService{
		itemRepo:    itemRepo,
		groupRepo:   groupRepo,
		recordRepo:  recordRepo,
		hostRepo:    hostRepo,
		cmdExecutor: cmdExecutor,
		validator:   &inspectionmgmtbiz.AssertionValidator{},
		extractor:   &inspectionmgmtbiz.VariableExtractor{},
		replacer:    &inspectionmgmtbiz.VariableReplacer{},
	}
}

func (s *ItemService) Create(ctx context.Context, req *ItemCreateRequest) (uint, error) {
	// 添加日志
	fmt.Printf("[DEBUG] Create InspectionItem:\n")
	fmt.Printf("  Name: %s\n", req.Name)
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
		PromQLQuery:       req.PromQLQuery,
		HostMatchType:     req.HostMatchType,
		HostTags:          req.HostTags,
		HostIDs:           req.HostIDs,
		AssertionType:     req.AssertionType,
		AssertionValue:    req.AssertionValue,
		VariableName:      req.VariableName,
		VariableRegex:     req.VariableRegex,
		Timeout:           req.Timeout,
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
	}
	item.Command = req.Command
	item.ScriptType = req.ScriptType
	item.ScriptContent = req.ScriptContent
	item.ScriptFile = req.ScriptFile
	item.PromQLQuery = req.PromQLQuery
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

		// 匹配主机
		hosts, err := s.matchHosts(ctx, item, group)
		if err != nil {
			continue
		}

		// 对每个主机执行巡检
		for _, host := range hosts {
			result := s.executeItem(ctx, item, group, host, nil)
			results = append(results, result)
		}
	}

	return results, nil
}

func (s *ItemService) executeItem(ctx context.Context, item *inspectionmgmtdata.InspectionItem, group *inspectionmgmtdata.InspectionGroup, host *assetbiz.Host, variables map[string]string) *TestRunResponse {
	result := &TestRunResponse{
		ItemID:   item.ID,
		ItemName: item.Name,
		HostID:   host.ID,
		HostName: host.Name,
		HostIp:   host.IP,
	}

	// 替换变量
	command := s.replacer.Replace(item.Command, variables)

	// 执行命令
	execResult := s.cmdExecutor.Execute(ctx, host, command, group.ExecutionMode, item.Timeout)
	result.Duration = execResult.Duration
	result.Output = execResult.Output

	if execResult.Error != nil {
		result.Status = "failed"
		result.ErrorMessage = execResult.Error.Error()
		result.AssertionResult = "skip"
		return result
	}

	result.Status = "success"

	// 断言校验
	assertionResult := s.validator.Validate(item.AssertionType, item.AssertionValue, execResult.Output)
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
		PromQLQuery:       item.PromQLQuery,
		HostMatchType:     item.HostMatchType,
		HostTags:          item.HostTags,
		HostIDs:           item.HostIDs,
		AssertionType:     item.AssertionType,
		AssertionValue:    item.AssertionValue,
		VariableName:      item.VariableName,
		VariableRegex:     item.VariableRegex,
		Timeout:           item.Timeout,
		CreatedAt:         item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         item.UpdatedAt.Format(time.RFC3339),
	}
}

// BatchSave 批量保存巡检项
func (s *ItemService) BatchSave(ctx context.Context, groupID uint, items []ItemCreateRequest) error {
	// 先删除该组下的所有巡检项
	if err := s.itemRepo.DeleteByGroupID(ctx, groupID); err != nil {
		return fmt.Errorf("删除旧巡检项失败: %v", err)
	}

	// 批量创建新巡检项
	for _, req := range items {
		req.GroupID = groupID
		if _, err := s.Create(ctx, &req); err != nil {
			return fmt.Errorf("创建巡检项失败: %v", err)
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

	// 获取目标主机列表
	hosts, err := s.matchHosts(ctx, item, group)
	if err != nil {
		return nil, fmt.Errorf("match hosts failed: %w", err)
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("no target hosts found for item %d", itemID)
	}

	// 执行巡检并保存记录
	records := make([]*inspectionmgmtdata.InspectionRecord, 0, len(hosts))
	variables := make(map[string]string)

	for _, host := range hosts {
		result := s.executeItem(ctx, item, group, host, variables)

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
