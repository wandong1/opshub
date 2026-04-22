package inspection_mgmt

import (
	"context"
	"encoding/json"
	"fmt"

	inspectionbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
	"github.com/ydcloud-dy/opshub/pkg/utils"
)

// VariableResolver 变量解析器，负责合并三个来源的变量
type VariableResolver struct {
	variableRepo inspectionbiz.ProbeVariableRepo
	groupRepo    inspectionmgmtdata.GroupRepository
}

// NewVariableResolver 创建变量解析器
func NewVariableResolver(variableRepo inspectionbiz.ProbeVariableRepo, groupRepo inspectionmgmtdata.GroupRepository) *VariableResolver {
	return &VariableResolver{
		variableRepo: variableRepo,
		groupRepo:    groupRepo,
	}
}

// ResolveVariables 解析变量，合并四个来源：预置变量、全局变量、巡检组自定义变量、运行时提取的变量
// 优先级：运行时提取的变量 > 巡检组自定义变量 > 全局变量 > 预置变量
func (r *VariableResolver) ResolveVariables(ctx context.Context, groupID uint, runtimeVars map[string]string, hostIP string) (map[string]string, error) {
	result := make(map[string]string)

	// 1. 生成系统预置变量（最低优先级）
	presetVars := utils.GeneratePresetVariables(hostIP)
	for k, v := range presetVars {
		result[k] = v
	}

	// 2. 加载全局变量
	globalVars, err := r.loadGlobalVariables(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载全局变量失败: %v", err)
	}
	for k, v := range globalVars {
		result[k] = v
	}

	// 3. 加载巡检组自定义变量（覆盖全局变量）
	groupVars, err := r.loadGroupVariables(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("加载巡检组变量失败: %v", err)
	}
	for k, v := range groupVars {
		result[k] = v
	}

	// 4. 合并运行时提取的变量（优先级最高）
	for k, v := range runtimeVars {
		result[k] = v
	}

	return result, nil
}

// loadGlobalVariables 加载全局环境变量
func (r *VariableResolver) loadGlobalVariables(ctx context.Context) (map[string]string, error) {
	// 获取所有变量（不分页）
	variables, _, err := r.variableRepo.List(ctx, 1, 10000, "", "", "")
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, v := range variables {
		result[v.Name] = v.Value
	}

	return result, nil
}

// loadGroupVariables 加载巡检组自定义变量
func (r *VariableResolver) loadGroupVariables(ctx context.Context, groupID uint) (map[string]string, error) {
	group, err := r.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)

	// 解析 JSON 格式的自定义变量
	if group.CustomVariables != "" && group.CustomVariables != "{}" {
		var customVars map[string]string
		if err := json.Unmarshal([]byte(group.CustomVariables), &customVars); err != nil {
			return nil, fmt.Errorf("解析自定义变量失败: %v", err)
		}
		result = customVars
	}

	return result, nil
}

// GetVariablesForGroup 获取指定巡检组可用的所有变量（用于前端下拉列表）
func (r *VariableResolver) GetVariablesForGroup(ctx context.Context, groupID uint) ([]VariableOption, error) {
	var options []VariableOption

	// 1. 加载全局变量
	globalVars, _, err := r.variableRepo.List(ctx, 1, 10000, "", "", "")
	if err != nil {
		return nil, err
	}
	for _, v := range globalVars {
		options = append(options, VariableOption{
			Name:   v.Name,
			Value:  v.Value,
			Source: "global",
			Type:   v.VarType,
		})
	}

	// 2. 加载巡检组自定义变量
	groupVars, err := r.loadGroupVariables(ctx, groupID)
	if err != nil {
		return nil, err
	}
	for name, value := range groupVars {
		options = append(options, VariableOption{
			Name:   name,
			Value:  value,
			Source: "group",
			Type:   "plain",
		})
	}

	return options, nil
}

// VariableOption 变量选项（用于前端下拉列表）
type VariableOption struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Source string `json:"source"` // global, group, runtime
	Type   string `json:"type"`   // plain, secret
}
