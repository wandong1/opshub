package inspection_mgmt

import (
	"context"
	"time"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
)

type RecordService struct {
	recordRepo inspectionmgmtdata.RecordRepository
	itemRepo   inspectionmgmtdata.ItemRepository
	groupRepo  inspectionmgmtdata.GroupRepository
	hostRepo   assetbiz.HostRepo
}

func NewRecordService(
	recordRepo inspectionmgmtdata.RecordRepository,
	itemRepo inspectionmgmtdata.ItemRepository,
	groupRepo inspectionmgmtdata.GroupRepository,
) *RecordService {
	return &RecordService{
		recordRepo: recordRepo,
		itemRepo:   itemRepo,
		groupRepo:  groupRepo,
	}
}

// SetHostRepo 设置主机仓库（依赖注入）
func (s *RecordService) SetHostRepo(hostRepo assetbiz.HostRepo) {
	s.hostRepo = hostRepo
}

func (s *RecordService) GetByID(ctx context.Context, id uint) (*RecordResponse, error) {
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(ctx, record), nil
}

func (s *RecordService) List(ctx context.Context, req *RecordListRequest) ([]*RecordResponse, int64, error) {
	var startTime, endTime *time.Time
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

	records, total, err := s.recordRepo.List(
		ctx,
		req.Page,
		req.PageSize,
		req.TaskID,
		req.GroupID,
		req.ItemID,
		req.HostID,
		req.Status,
		startTime,
		endTime,
	)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*RecordResponse, len(records))
	for i, record := range records {
		responses[i] = s.toResponse(ctx, record)
	}

	return responses, total, nil
}

func (s *RecordService) toResponse(ctx context.Context, record *inspectionmgmtdata.InspectionRecord) *RecordResponse {
	resp := &RecordResponse{
		ID:                 record.ID,
		TaskID:             record.TaskID,
		GroupID:            record.GroupID,
		ItemID:             record.ItemID,
		HostID:             record.HostID,
		Status:             record.Status,
		Output:             record.Output,
		ErrorMessage:       record.ErrorMessage,
		Duration:           record.Duration,
		AssertionResult:    record.AssertionResult,
		AssertionDetails:   record.AssertionDetails,
		ExtractedVariables: record.ExtractedVariables,
		ExecutedAt:         record.ExecutedAt.Format(time.RFC3339),
		CreatedAt:          record.CreatedAt.Format(time.RFC3339),
	}

	// 获取巡检项名称
	if item, err := s.itemRepo.GetByID(ctx, record.ItemID); err == nil {
		resp.ItemName = item.Name
	}

	// 获取巡检组名称
	if record.GroupID > 0 {
		if group, err := s.groupRepo.GetByID(ctx, record.GroupID); err == nil {
			resp.GroupName = group.Name
		}
	}

	// 获取主机名称
	if record.HostID > 0 && s.hostRepo != nil {
		if host, err := s.hostRepo.GetByID(ctx, record.HostID); err == nil && host != nil {
			resp.HostName = host.Name
		}
	}

	return resp
}
