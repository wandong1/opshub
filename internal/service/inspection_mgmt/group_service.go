package inspection_mgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
	"gopkg.in/yaml.v3"
)

type GroupService struct {
	groupRepo inspectionmgmtdata.GroupRepository
	itemRepo  inspectionmgmtdata.ItemRepository
}

func NewGroupService(groupRepo inspectionmgmtdata.GroupRepository, itemRepo inspectionmgmtdata.ItemRepository) *GroupService {
	return &GroupService{
		groupRepo: groupRepo,
		itemRepo:  itemRepo,
	}
}

func (s *GroupService) Create(ctx context.Context, req *GroupCreateRequest) (uint, error) {
	group := &inspectionmgmtdata.InspectionGroup{
		Name:              req.Name,
		Description:       req.Description,
		Status:            req.Status,
		Sort:              req.Sort,
		PrometheusURL:     req.PrometheusURL,
		PrometheusUsername: req.PrometheusUsername,
		PrometheusPassword: req.PrometheusPassword,
		ExecutionMode:     req.ExecutionMode,
		ExecutionStrategy: req.ExecutionStrategy,
		Concurrency:       req.Concurrency,
		GroupIDs:          req.GroupIDs,
	}

	if group.Status == "" {
		group.Status = "enabled"
	}
	if group.ExecutionMode == "" {
		group.ExecutionMode = "auto"
	}
	if group.ExecutionStrategy == "" {
		group.ExecutionStrategy = "concurrent"
	}
	if group.Concurrency == 0 {
		group.Concurrency = 50
	}

	if err := s.groupRepo.Create(ctx, group); err != nil {
		return 0, err
	}
	return group.ID, nil
}

func (s *GroupService) Update(ctx context.Context, id uint, req *GroupUpdateRequest) error {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	group.Description = req.Description
	if req.Status != "" {
		group.Status = req.Status
	}
	group.Sort = req.Sort
	group.PrometheusURL = req.PrometheusURL
	group.PrometheusUsername = req.PrometheusUsername
	if req.PrometheusPassword != "" {
		group.PrometheusPassword = req.PrometheusPassword
	}
	if req.ExecutionMode != "" {
		group.ExecutionMode = req.ExecutionMode
	}
	if req.ExecutionStrategy != "" {
		group.ExecutionStrategy = req.ExecutionStrategy
	}
	if req.Concurrency > 0 {
		group.Concurrency = req.Concurrency
	}
	// 修复：即使是空字符串也要更新，因为前端可能清空了选择
	group.GroupIDs = req.GroupIDs

	return s.groupRepo.Update(ctx, group)
}

func (s *GroupService) Delete(ctx context.Context, id uint) error {
	return s.groupRepo.Delete(ctx, id)
}

func (s *GroupService) GetByID(ctx context.Context, id uint) (*GroupResponse, error) {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(group), nil
}

func (s *GroupService) List(ctx context.Context, req *GroupListRequest) ([]*GroupResponse, int64, error) {
	groups, total, err := s.groupRepo.List(ctx, req.Page, req.PageSize, req.Name, req.Status)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*GroupResponse, len(groups))
	for i, group := range groups {
		resp := s.toResponse(group)

		// 获取该巡检组的巡检项
		items, err := s.itemRepo.GetByGroupID(ctx, group.ID)
		if err == nil {
			resp.ItemCount = len(items)
			itemNames := make([]string, 0, len(items))
			for _, item := range items {
				itemNames = append(itemNames, item.Name)
			}
			resp.ItemNames = itemNames
		}

		responses[i] = resp
	}

	return responses, total, nil
}

func (s *GroupService) GetAll(ctx context.Context) ([]*GroupResponse, error) {
	groups, err := s.groupRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]*GroupResponse, len(groups))
	for i, group := range groups {
		// 查询该巡检组的巡检项数量
		items, err := s.itemRepo.GetByGroupID(ctx, group.ID)
		itemCount := 0
		if err == nil {
			itemCount = len(items)
		}

		responses[i] = s.toResponseWithCount(group, itemCount)
	}

	return responses, nil
}

func (s *GroupService) toResponse(group *inspectionmgmtdata.InspectionGroup) *GroupResponse {
	return &GroupResponse{
		ID:                group.ID,
		Name:              group.Name,
		Description:       group.Description,
		Status:            group.Status,
		Sort:              group.Sort,
		PrometheusURL:     group.PrometheusURL,
		PrometheusUsername: group.PrometheusUsername,
		ExecutionMode:     group.ExecutionMode,
		ExecutionStrategy: group.ExecutionStrategy,
		Concurrency:       group.Concurrency,
		GroupIDs:          group.GroupIDs,
		ItemCount:         0,
		ItemNames:         []string{},
		CreatedAt:         group.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         group.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *GroupService) toResponseWithCount(group *inspectionmgmtdata.InspectionGroup, itemCount int) *GroupResponse {
	return &GroupResponse{
		ID:                group.ID,
		Name:              group.Name,
		Description:       group.Description,
		Status:            group.Status,
		Sort:              group.Sort,
		PrometheusURL:     group.PrometheusURL,
		PrometheusUsername: group.PrometheusUsername,
		ExecutionMode:     group.ExecutionMode,
		ExecutionStrategy: group.ExecutionStrategy,
		Concurrency:       group.Concurrency,
		GroupIDs:          group.GroupIDs,
		ItemCount:         itemCount,
		ItemNames:         []string{},
		CreatedAt:         group.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         group.UpdatedAt.Format(time.RFC3339),
	}
}

// GetStats 获取统计数据
func (s *GroupService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats, err := s.groupRepo.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// Export 导出巡检组配置
func (s *GroupService) Export(ctx context.Context, id uint, format string) (string, error) {
	// 获取巡检组
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("获取巡检组失败: %v", err)
	}

	// 获取巡检项
	items, err := s.itemRepo.GetByGroupID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("获取巡检项失败: %v", err)
	}

	// 构建导出数据
	exportData := s.buildExportData(group, items)

	// 根据格式序列化
	return s.serializeExportData(exportData, format)
}

// ExportAll 导出所有巡检组配置
func (s *GroupService) ExportAll(ctx context.Context, format string) (string, error) {
	// 获取所有巡检组
	groups, err := s.groupRepo.GetAll(ctx)
	if err != nil {
		return "", fmt.Errorf("获取巡检组列表失败: %v", err)
	}

	// 构建所有巡检组的导出数据
	allExportData := make([]GroupExportData, 0, len(groups))
	for _, group := range groups {
		// 获取巡检项
		items, err := s.itemRepo.GetByGroupID(ctx, group.ID)
		if err != nil {
			continue // 跳过获取失败的
		}

		exportData := s.buildExportData(group, items)
		allExportData = append(allExportData, exportData)
	}

	// 根据格式序列化
	var result []byte
	if format == "yaml" {
		result, err = yaml.Marshal(allExportData)
	} else {
		result, err = json.MarshalIndent(allExportData, "", "  ")
	}

	if err != nil {
		return "", fmt.Errorf("序列化失败: %v", err)
	}

	return string(result), nil
}

// buildExportData 构建导出数据
func (s *GroupService) buildExportData(group *inspectionmgmtdata.InspectionGroup, items []*inspectionmgmtdata.InspectionItem) GroupExportData {
	exportData := GroupExportData{
		Name:              group.Name,
		Description:       group.Description,
		Status:            group.Status,
		PrometheusURL:     group.PrometheusURL,
		PrometheusUsername: group.PrometheusUsername,
		ExecutionMode:     group.ExecutionMode,
		ExecutionStrategy: group.ExecutionStrategy,
		Concurrency:       group.Concurrency,
		Items:             make([]ItemExportData, 0, len(items)),
	}

	// 解析 GroupIDs
	if group.GroupIDs != "" {
		var groupIDs []uint
		if err := json.Unmarshal([]byte(group.GroupIDs), &groupIDs); err == nil {
			exportData.GroupIDs = groupIDs
		}
	}

	// 转换巡检项
	for _, item := range items {
		itemData := ItemExportData{
			Name:              item.Name,
			Description:       item.Description,
			ExecutionType:     item.ExecutionType,
			ExecutionStrategy: item.ExecutionStrategy,
			Command:           item.Command,
			ScriptType:        item.ScriptType,
			ScriptContent:     item.ScriptContent,
			ScriptFile:        item.ScriptFile,
			PromQLQuery:       item.PromQLQuery,
			HostMatchType:     item.HostMatchType,
			AssertionType:     item.AssertionType,
			AssertionValue:    item.AssertionValue,
			VariableName:      item.VariableName,
			VariableRegex:     item.VariableRegex,
			Timeout:           item.Timeout,
			Status:            item.Status,
		}

		// 解析 HostTags
		if item.HostTags != "" {
			var hostTags []string
			if err := json.Unmarshal([]byte(item.HostTags), &hostTags); err == nil {
				itemData.HostTags = hostTags
			}
		}

		// 解析 HostIDs
		if item.HostIDs != "" {
			var hostIDs []uint
			if err := json.Unmarshal([]byte(item.HostIDs), &hostIDs); err == nil {
				itemData.HostIDs = hostIDs
			}
		}

		exportData.Items = append(exportData.Items, itemData)
	}

	return exportData
}

// serializeExportData 序列化导出数据
func (s *GroupService) serializeExportData(data interface{}, format string) (string, error) {
	var result []byte
	var err error

	if format == "yaml" {
		result, err = yaml.Marshal(data)
	} else {
		result, err = json.MarshalIndent(data, "", "  ")
	}

	if err != nil {
		return "", fmt.Errorf("序列化失败: %v", err)
	}

	return string(result), nil
}

// Import 导入巡检组配置（支持单个或多个）
func (s *GroupService) Import(ctx context.Context, req *GroupImportRequest) ([]uint, error) {
	// 尝试解析为数组
	var exportDataArray []GroupExportData
	var err error

	if req.Format == "yaml" {
		err = yaml.Unmarshal([]byte(req.Data), &exportDataArray)
	} else {
		err = json.Unmarshal([]byte(req.Data), &exportDataArray)
	}

	// 如果解析数组失败，尝试解析为单个对象
	if err != nil {
		var singleData GroupExportData
		if req.Format == "yaml" {
			err = yaml.Unmarshal([]byte(req.Data), &singleData)
		} else {
			err = json.Unmarshal([]byte(req.Data), &singleData)
		}

		if err != nil {
			return nil, fmt.Errorf("解析数据失败: %v", err)
		}

		exportDataArray = []GroupExportData{singleData}
	}

	// 批量导入
	importedIDs := make([]uint, 0, len(exportDataArray))
	for _, exportData := range exportDataArray {
		groupID, err := s.importSingleGroup(ctx, exportData)
		if err != nil {
			return importedIDs, fmt.Errorf("导入巡检组 '%s' 失败: %v", exportData.Name, err)
		}
		importedIDs = append(importedIDs, groupID)
	}

	return importedIDs, nil
}

// importSingleGroup 导入单个巡检组
func (s *GroupService) importSingleGroup(ctx context.Context, exportData GroupExportData) (uint, error) {
	// 序列化 GroupIDs
	groupIDsJSON := "[]"
	if len(exportData.GroupIDs) > 0 {
		groupIDsBytes, _ := json.Marshal(exportData.GroupIDs)
		groupIDsJSON = string(groupIDsBytes)
	}

	// 创建巡检组
	createReq := &GroupCreateRequest{
		Name:              exportData.Name,
		Description:       exportData.Description,
		Status:            exportData.Status,
		PrometheusURL:     exportData.PrometheusURL,
		PrometheusUsername: exportData.PrometheusUsername,
		ExecutionMode:     exportData.ExecutionMode,
		ExecutionStrategy: exportData.ExecutionStrategy,
		Concurrency:       exportData.Concurrency,
		GroupIDs:          groupIDsJSON,
	}

	groupID, err := s.Create(ctx, createReq)
	if err != nil {
		return 0, err
	}

	// 创建巡检项
	for i, itemData := range exportData.Items {
		// 序列化 HostTags
		hostTagsJSON := "[]"
		if len(itemData.HostTags) > 0 {
			hostTagsBytes, _ := json.Marshal(itemData.HostTags)
			hostTagsJSON = string(hostTagsBytes)
		}

		// 序列化 HostIDs
		hostIDsJSON := "[]"
		if len(itemData.HostIDs) > 0 {
			hostIDsBytes, _ := json.Marshal(itemData.HostIDs)
			hostIDsJSON = string(hostIDsBytes)
		}

		itemReq := &ItemCreateRequest{
			Name:              itemData.Name,
			Description:       itemData.Description,
			GroupID:           groupID,
			Sort:              i,
			Status:            itemData.Status,
			ExecutionStrategy: itemData.ExecutionStrategy,
			ExecutionType:     itemData.ExecutionType,
			Command:           itemData.Command,
			ScriptType:        itemData.ScriptType,
			ScriptContent:     itemData.ScriptContent,
			ScriptFile:        itemData.ScriptFile,
			PromQLQuery:       itemData.PromQLQuery,
			HostMatchType:     itemData.HostMatchType,
			HostTags:          hostTagsJSON,
			HostIDs:           hostIDsJSON,
			AssertionType:     itemData.AssertionType,
			AssertionValue:    itemData.AssertionValue,
			VariableName:      itemData.VariableName,
			VariableRegex:     itemData.VariableRegex,
			Timeout:           itemData.Timeout,
		}

		// 需要注入 ItemService 来创建巡检项
		itemService := NewItemService(s.itemRepo, s.groupRepo, nil, nil, nil, nil)
		if _, err := itemService.Create(ctx, itemReq); err != nil {
			// 如果创建巡检项失败，删除已创建的巡检组
			s.groupRepo.Delete(ctx, groupID)
			return 0, fmt.Errorf("创建巡检项失败: %v", err)
		}
	}

	return groupID, nil
}
