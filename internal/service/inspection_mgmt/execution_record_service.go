package inspection_mgmt

import (
	"context"
	"encoding/json"
	"time"

	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
)

// ExecutionRecordService 巡检执行记录服务
type ExecutionRecordService struct {
	execRepo inspectionmgmtdata.ExecutionRecordRepository
}

// NewExecutionRecordService 创建执行记录服务
func NewExecutionRecordService(execRepo inspectionmgmtdata.ExecutionRecordRepository) *ExecutionRecordService {
	return &ExecutionRecordService{
		execRepo: execRepo,
	}
}

// ExecutionRecordResponse 执行记录响应
type ExecutionRecordResponse struct {
	ID                 uint      `json:"id"`
	TaskID             uint      `json:"taskId"`
	TaskName           string    `json:"taskName"`
	TotalItems         int       `json:"totalItems"`
	TotalHosts         int       `json:"totalHosts"`
	TotalExecutions    int       `json:"totalExecutions"`
	SuccessCount       int       `json:"successCount"`
	FailedCount        int       `json:"failedCount"`
	AssertionPassCount int       `json:"assertionPassCount"`
	AssertionFailCount int       `json:"assertionFailCount"`
	AssertionSkipCount int       `json:"assertionSkipCount"`
	Status             string    `json:"status"`
	Duration           float64   `json:"duration"`
	StartedAt          string    `json:"startedAt"`
	CompletedAt        string    `json:"completedAt,omitempty"`
	GroupNames         []string  `json:"groupNames"`
	CreatedAt          string    `json:"createdAt"`
}

// ExecutionDetailResponse 执行明细响应
type ExecutionDetailResponse struct {
	ID                 uint    `json:"id"`
	ExecutionID        uint    `json:"executionId"`
	GroupID            uint    `json:"groupId"`
	GroupName          string  `json:"groupName"`
	ItemID             uint    `json:"itemId"`
	ItemName           string  `json:"itemName"`
	InspectionLevel    string  `json:"inspectionLevel"` // 巡检级别
	RiskLevel          string  `json:"riskLevel"`       // 风险等级
	HostID             uint    `json:"hostId"`
	HostName           string  `json:"hostName"`
	HostIP             string  `json:"hostIp"`
	BusinessGroup      string  `json:"businessGroup"`
	ExecutionType      string  `json:"executionType"`
	ExecutionMode      string  `json:"executionMode"`
	Command            string  `json:"command"`
	ScriptType         string  `json:"scriptType"`
	ScriptContent      string  `json:"scriptContent"`
	AssertionType      string  `json:"assertionType"`
	AssertionValue     string  `json:"assertionValue"`
	Status             string  `json:"status"`
	Output             string  `json:"output"`
	ErrorMessage       string  `json:"errorMessage"`
	Duration           float64 `json:"duration"`
	AssertionResult    string  `json:"assertionResult"`
	AssertionDetails   string  `json:"assertionDetails"`
	ExtractedVariables string  `json:"extractedVariables"`
	ExecutedAt         string  `json:"executedAt"`
}

// ExecutionRecordListRequest 执行记录列表请求
type ExecutionRecordListRequest struct {
	Page      int    `form:"page" binding:"required,min=1"`
	PageSize  int    `form:"pageSize" binding:"required,min=1,max=100"`
	TaskID    uint   `form:"taskId"`
	Status    string `form:"status"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
}

// GetByID 获取执行记录详情
func (s *ExecutionRecordService) GetByID(ctx context.Context, id uint) (*ExecutionRecordResponse, error) {
	record, err := s.execRepo.GetRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(record), nil
}

// List 分页查询执行记录列表
func (s *ExecutionRecordService) List(ctx context.Context, req *ExecutionRecordListRequest) ([]*ExecutionRecordResponse, int64, error) {
	var startTime, endTime *time.Time

	// 解析时间参数
	if req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err == nil {
			startTime = &t
		}
	}
	if req.EndTime != "" {
		t, err := time.Parse(time.RFC3339, req.EndTime)
		if err == nil {
			endTime = &t
		}
	}

	records, total, err := s.execRepo.ListRecords(ctx, req.Page, req.PageSize, req.TaskID, req.Status, startTime, endTime)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*ExecutionRecordResponse, len(records))
	for i, record := range records {
		responses[i] = s.toResponse(record)
	}

	return responses, total, nil
}

// GetDetails 获取执行记录的所有明细
func (s *ExecutionRecordService) GetDetails(ctx context.Context, executionID uint) ([]*ExecutionDetailResponse, error) {
	details, err := s.execRepo.GetDetailsByExecutionID(ctx, executionID)
	if err != nil {
		return nil, err
	}

	responses := make([]*ExecutionDetailResponse, len(details))
	for i, detail := range details {
		responses[i] = s.toDetailResponse(detail)
	}

	return responses, nil
}

// Delete 删除执行记录
func (s *ExecutionRecordService) Delete(ctx context.Context, id uint) error {
	return s.execRepo.DeleteRecord(ctx, id)
}

// toResponse 转换为响应对象
func (s *ExecutionRecordService) toResponse(record *inspectionmgmtdata.InspectionExecutionRecord) *ExecutionRecordResponse {
	resp := &ExecutionRecordResponse{
		ID:                 record.ID,
		TaskID:             record.TaskID,
		TaskName:           record.TaskName,
		TotalItems:         record.TotalItems,
		TotalHosts:         record.TotalHosts,
		TotalExecutions:    record.TotalExecutions,
		SuccessCount:       record.SuccessCount,
		FailedCount:        record.FailedCount,
		AssertionPassCount: record.AssertionPassCount,
		AssertionFailCount: record.AssertionFailCount,
		AssertionSkipCount: record.AssertionSkipCount,
		Status:             record.Status,
		Duration:           record.Duration,
		StartedAt:          record.StartedAt.Format(time.RFC3339),
		CreatedAt:          record.CreatedAt.Format(time.RFC3339),
	}

	if record.CompletedAt != nil {
		resp.CompletedAt = record.CompletedAt.Format(time.RFC3339)
	}

	// 解析 GroupNames JSON
	if record.GroupNames != "" {
		var groupNames []string
		if err := json.Unmarshal([]byte(record.GroupNames), &groupNames); err == nil {
			resp.GroupNames = groupNames
		}
	}

	return resp
}

// toDetailResponse 转换明细为响应对象
func (s *ExecutionRecordService) toDetailResponse(detail *inspectionmgmtdata.InspectionExecutionDetail) *ExecutionDetailResponse {
	return &ExecutionDetailResponse{
		ID:                 detail.ID,
		ExecutionID:        detail.ExecutionID,
		GroupID:            detail.GroupID,
		GroupName:          detail.GroupName,
		ItemID:             detail.ItemID,
		ItemName:           detail.ItemName,
		InspectionLevel:    detail.InspectionLevel,
		RiskLevel:          detail.RiskLevel,
		HostID:             detail.HostID,
		HostName:           detail.HostName,
		HostIP:             detail.HostIP,
		BusinessGroup:      detail.BusinessGroup,
		ExecutionType:      detail.ExecutionType,
		ExecutionMode:      detail.ExecutionMode,
		Command:            detail.Command,
		ScriptType:         detail.ScriptType,
		ScriptContent:      detail.ScriptContent,
		AssertionType:      detail.AssertionType,
		AssertionValue:     detail.AssertionValue,
		Status:             detail.Status,
		Output:             detail.Output,
		ErrorMessage:       detail.ErrorMessage,
		Duration:           detail.Duration,
		AssertionResult:    detail.AssertionResult,
		AssertionDetails:   detail.AssertionDetails,
		ExtractedVariables: detail.ExtractedVariables,
		ExecutedAt:         detail.ExecutedAt.Format(time.RFC3339),
	}
}
