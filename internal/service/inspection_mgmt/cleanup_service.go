package inspection_mgmt

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	systembiz "github.com/ydcloud-dy/opshub/internal/biz/system"
	"github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

// CleanupService 执行记录清理服务
type CleanupService struct {
	recordRepo    inspection_mgmt.RecordRepository
	configUseCase *systembiz.ConfigUseCase
	cron          *cron.Cron
}

// NewCleanupService 创建清理服务
func NewCleanupService(
	recordRepo inspection_mgmt.RecordRepository,
	configUseCase *systembiz.ConfigUseCase,
) *CleanupService {
	return &CleanupService{
		recordRepo:    recordRepo,
		configUseCase: configUseCase,
		cron:          cron.New(),
	}
}

// Start 启动定时清理任务（每天凌晨2点执行）
func (s *CleanupService) Start() error {
	// 每天凌晨2点执行清理
	_, err := s.cron.AddFunc("0 2 * * *", func() {
		s.cleanup()
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	appLogger.Info("智能巡检执行记录清理服务已启动，每天凌晨2点执行清理")
	return nil
}

// Stop 停止定时清理任务
func (s *CleanupService) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		appLogger.Info("智能巡检执行记录清理服务已停止")
	}
}

// cleanup 执行清理逻辑
func (s *CleanupService) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 获取配置的保留数量
	retention := s.configUseCase.GetInspectionRecordRetention(ctx)

	appLogger.Info("开始清理智能巡检执行记录",
		zap.Int("retention", retention),
	)

	// 先获取当前总记录数
	totalCount, err := s.recordRepo.GetTotalCount(ctx)
	if err != nil {
		appLogger.Error("获取记录总数失败", zap.Error(err))
		return
	}

	appLogger.Info("当前记录总数",
		zap.Int64("total", totalCount),
		zap.Int("retention", retention),
	)

	// 如果记录数未超过保留数量，无需清理
	if totalCount <= int64(retention) {
		appLogger.Info("记录数未超过保留数量，无需清理")
		return
	}

	// 执行清理
	if err := s.recordRepo.CleanupExcessRecords(ctx, retention); err != nil {
		appLogger.Error("清理执行记录失败", zap.Error(err))
		return
	}

	// 获取清理后的记录数
	afterCount, err := s.recordRepo.GetTotalCount(ctx)
	if err != nil {
		appLogger.Error("获取清理后记录总数失败", zap.Error(err))
		return
	}

	deletedCount := totalCount - afterCount
	appLogger.Info("智能巡检执行记录清理完成",
		zap.Int64("deleted", deletedCount),
		zap.Int64("remaining", afterCount),
	)
}

// CleanupNow 立即执行一次清理（用于手动触发）
func (s *CleanupService) CleanupNow() {
	s.cleanup()
}
