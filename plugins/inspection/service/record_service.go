package service

import (
	"context"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/inspection/dto"
	"github.com/ydcloud-dy/opshub/plugins/inspection/model"
	"github.com/ydcloud-dy/opshub/plugins/inspection/repository"
)

type RecordService struct {
	recordRepo repository.RecordRepository
	itemRepo   repository.ItemRepository
	hostRepo   interface {
		GetByID(ctx context.Context, id uint) (interface{}, error)
	}
}

func NewRecordService(
	recordRepo repository.RecordRepository,
	itemRepo repository.ItemRepository,
) *RecordService {
	return &RecordService{
		recordRepo: recordRepo,
		itemRepo:   itemRepo,
	}
}

func (s *RecordService) GetByID(ctx context.Context, id uint) (*dto.RecordResponse, error) {
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(ctx, record), nil
}

func (s *RecordService) List(ctx context.Context, req *dto.RecordListRequest) ([]*dto.RecordResponse, int64, error) {
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

	responses := make([]*dto.RecordResponse, len(records))
	for i, record := range records {
		responses[i] = s.toResponse(ctx, record)
	}

	return responses, total, nil
}

func (s *RecordService) toResponse(ctx context.Context, record *model.InspectionRecord) *dto.RecordResponse {
	resp := &dto.RecordResponse{
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

	return resp
}
