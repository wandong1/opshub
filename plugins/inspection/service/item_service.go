package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/plugins/inspection/dto"
	"github.com/ydcloud-dy/opshub/plugins/inspection/executor"
	"github.com/ydcloud-dy/opshub/plugins/inspection/model"
	"github.com/ydcloud-dy/opshub/plugins/inspection/repository"
)

type ItemService struct {
	itemRepo    repository.ItemRepository
	groupRepo   repository.GroupRepository
	recordRepo  repository.RecordRepository
	hostRepo    assetbiz.HostRepo
	cmdExecutor *executor.CommandExecutor
	validator   *executor.AssertionValidator
	extractor   *executor.VariableExtractor
	replacer    *executor.VariableReplacer
}

func NewItemService(
	itemRepo repository.ItemRepository,
	groupRepo repository.GroupRepository,
	recordRepo repository.RecordRepository,
	hostRepo assetbiz.HostRepo,
	cmdExecutor *executor.CommandExecutor,
) *ItemService {
	return &ItemService{
		itemRepo:    itemRepo,
		groupRepo:   groupRepo,
		recordRepo:  recordRepo,
		hostRepo:    hostRepo,
		cmdExecutor: cmdExecutor,
		validator:   &executor.AssertionValidator{},
		extractor:   &executor.VariableExtractor{},
		replacer:    &executor.VariableReplacer{},
	}
}

func (s *ItemService) Create(ctx context.Context, req *dto.ItemCreateRequest) error {
	item := &model.InspectionItem{
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

	return s.itemRepo.Create(ctx, item)
}

func (s *ItemService) Update(ctx context.Context, id uint, req *dto.ItemUpdateRequest) error {
	item, err := s.itemRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if req.Name != "" {
		item.Name = req.Name
	}
	if req.Description != "" {
		item.Description = req.Description
	}
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

func (s *ItemService) GetByID(ctx context.Context, id uint) (*dto.ItemResponse, error) {
	item, err := s.itemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(item), nil
}

func (s *ItemService) List(ctx context.Context, req *dto.ItemListRequest) ([]*dto.ItemResponse, int64, error) {
	items, total, err := s.itemRepo.List(ctx, req.Page, req.PageSize, req.GroupID, req.Name, req.Status)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.ItemResponse, len(items))
	for i, item := range items {
		responses[i] = s.toResponse(item)
	}

	return responses, total, nil
}

func (s *ItemService) TestRun(ctx context.Context, req *dto.TestRunRequest) ([]*dto.TestRunResponse, error) {
	var results []*dto.TestRunResponse

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

func (s *ItemService) executeItem(ctx context.Context, item *model.InspectionItem, group *model.InspectionGroup, host *assetbiz.Host, variables map[string]string) *dto.TestRunResponse {
	result := &dto.TestRunResponse{
		ItemID:   item.ID,
		ItemName: item.Name,
		HostID:   host.ID,
		HostName: host.Name,
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

func (s *ItemService) matchHosts(ctx context.Context, item *model.InspectionItem, group *model.InspectionGroup) ([]*assetbiz.Host, error) {
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

func (s *ItemService) toResponse(item *model.InspectionItem) *dto.ItemResponse {
	return &dto.ItemResponse{
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
func (s *ItemService) BatchSave(ctx context.Context, groupID uint, items []dto.ItemCreateRequest) error {
	// 先删除该组下的所有巡检项
	if err := s.itemRepo.DeleteByGroupID(ctx, groupID); err != nil {
		return fmt.Errorf("删除旧巡检项失败: %v", err)
	}

	// 批量创建新巡检项
	for _, req := range items {
		req.GroupID = groupID
		if err := s.Create(ctx, &req); err != nil {
			return fmt.Errorf("创建巡检项失败: %v", err)
		}
	}

	return nil
}
