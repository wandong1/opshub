package inspection_mgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
	"github.com/xuri/excelize/v2"
)

// ExportTaskToExcel 导出巡检任务报告为 Excel（包含任务下所有巡检组和巡检项的执行记录）
func (s *TaskService) ExportTaskToExcel(ctx context.Context, taskID uint, hostRepo assetbiz.HostRepo, recordRepo inspectionmgmtdata.RecordRepository, itemRepo inspectionmgmtdata.ItemRepository, groupRepo inspectionmgmtdata.GroupRepository) (*excelize.File, error) {
	// 获取任务详情
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	// 解析任务配置的巡检组ID
	var groupIDs []uint
	if task.GroupIDs != "" {
		if err := json.Unmarshal([]byte(task.GroupIDs), &groupIDs); err != nil {
			return nil, fmt.Errorf("解析巡检组ID失败: %w", err)
		}
	}

	// 解析任务配置的巡检项ID
	var itemIDs []uint
	if task.ItemIDs != "" {
		if err := json.Unmarshal([]byte(task.ItemIDs), &itemIDs); err != nil {
			return nil, fmt.Errorf("解析巡检项ID失败: %w", err)
		}
	}

	// 获取该任务的所有执行记录
	records, err := recordRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("获取执行记录失败: %w", err)
	}

	// 创建 Excel 文件
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			// 忽略关闭错误
		}
	}()

	// 定义样式
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"5B9BD5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	contentStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	summaryHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"D9E1F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Sheet 1: 任务汇总信息
	summarySheet := "任务汇总"
	f.SetSheetName("Sheet1", summarySheet)

	// 设置列宽
	f.SetColWidth(summarySheet, "A", "A", 20)
	f.SetColWidth(summarySheet, "B", "B", 60)

	// 写入标题
	f.SetCellValue(summarySheet, "A1", "巡检任务报告")
	f.MergeCell(summarySheet, "A1", "B1")
	f.SetCellStyle(summarySheet, "A1", "B1", titleStyle)
	f.SetRowHeight(summarySheet, 1, 35)

	// 统计数据
	totalRecords := len(records)
	successCount := 0
	failCount := 0
	passCount := 0
	failAssertionCount := 0

	for _, record := range records {
		if record.Status == "success" {
			successCount++
		} else {
			failCount++
		}
		if record.AssertionResult == "pass" {
			passCount++
		} else if record.AssertionResult == "fail" {
			failAssertionCount++
		}
	}

	// 写入任务信息
	row := 3
	taskInfo := [][]string{
		{"任务名称", task.Name},
		{"任务描述", task.Description},
		{"任务类型", getTaskTypeText(task.TaskType)},
		{"调度表达式", task.CronExpr},
		{"配置巡检组数", strconv.Itoa(len(groupIDs))},
		{"配置巡检项数", strconv.Itoa(len(itemIDs))},
		{"执行记录总数", strconv.Itoa(totalRecords)},
		{"执行成功数", strconv.Itoa(successCount)},
		{"执行失败数", strconv.Itoa(failCount)},
		{"断言通过数", strconv.Itoa(passCount)},
		{"断言失败数", strconv.Itoa(failAssertionCount)},
	}

	if task.LastRunAt != nil {
		taskInfo = append(taskInfo, []string{"最后执行时间", task.LastRunAt.Format("2006-01-02 15:04:05")})
	}

	for _, data := range taskInfo {
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), data[0])
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), data[1])
		f.SetCellStyle(summarySheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), summaryHeaderStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), contentStyle)
		f.SetRowHeight(summarySheet, row, 25)
		row++
	}

	// Sheet 2: 执行记录详情
	detailSheet := "执行记录详情"
	f.NewSheet(detailSheet)

	// 设置列宽
	f.SetColWidth(detailSheet, "A", "A", 10)
	f.SetColWidth(detailSheet, "B", "B", 20)
	f.SetColWidth(detailSheet, "C", "C", 20)
	f.SetColWidth(detailSheet, "D", "D", 20)
	f.SetColWidth(detailSheet, "E", "E", 12)
	f.SetColWidth(detailSheet, "F", "F", 12)
	f.SetColWidth(detailSheet, "G", "G", 12)
	f.SetColWidth(detailSheet, "H", "H", 20)
	f.SetColWidth(detailSheet, "I", "I", 50)

	// 写入标题
	f.SetCellValue(detailSheet, "A1", "执行记录详情")
	f.MergeCell(detailSheet, "A1", "I1")
	f.SetCellStyle(detailSheet, "A1", "I1", titleStyle)
	f.SetRowHeight(detailSheet, 1, 35)

	// 写入表头
	headers := []string{"记录ID", "巡检组", "巡检项", "目标主机", "执行状态", "断言结果", "执行时长(秒)", "执行时间", "输出内容"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(detailSheet, cell, header)
		f.SetCellStyle(detailSheet, cell, cell, headerStyle)
	}
	f.SetRowHeight(detailSheet, 3, 30)

	// 写入记录数据
	detailRow := 4
	for _, record := range records {
		// 获取关联信息
		var itemName, groupName, hostName string
		if item, err := itemRepo.GetByID(ctx, record.ItemID); err == nil {
			itemName = item.Name
		}
		if record.GroupID > 0 {
			if group, err := groupRepo.GetByID(ctx, record.GroupID); err == nil {
				groupName = group.Name
			}
		}
		if record.HostID > 0 && hostRepo != nil {
			if host, err := hostRepo.GetByID(ctx, record.HostID); err == nil && host != nil {
				hostName = host.Name
			}
		}

		// 写入数据
		f.SetCellValue(detailSheet, fmt.Sprintf("A%d", detailRow), record.ID)
		f.SetCellValue(detailSheet, fmt.Sprintf("B%d", detailRow), groupName)
		f.SetCellValue(detailSheet, fmt.Sprintf("C%d", detailRow), itemName)
		f.SetCellValue(detailSheet, fmt.Sprintf("D%d", detailRow), hostName)
		f.SetCellValue(detailSheet, fmt.Sprintf("E%d", detailRow), getStatusText(record.Status))
		f.SetCellValue(detailSheet, fmt.Sprintf("F%d", detailRow), getAssertionResultText(record.AssertionResult))
		f.SetCellValue(detailSheet, fmt.Sprintf("G%d", detailRow), fmt.Sprintf("%.2f", record.Duration))
		f.SetCellValue(detailSheet, fmt.Sprintf("H%d", detailRow), record.ExecutedAt.Format("2006-01-02 15:04:05"))

		// 输出内容截取前500字符
		output := record.Output
		if len(output) > 500 {
			output = output[:500] + "..."
		}
		f.SetCellValue(detailSheet, fmt.Sprintf("I%d", detailRow), output)

		// 设置样式
		for col := 'A'; col <= 'I'; col++ {
			cell := fmt.Sprintf("%c%d", col, detailRow)
			f.SetCellStyle(detailSheet, cell, cell, contentStyle)
		}

		f.SetRowHeight(detailSheet, detailRow, 25)
		detailRow++
	}

	return f, nil
}

func getTaskTypeText(taskType string) string {
	switch taskType {
	case "inspection":
		return "巡检任务"
	case "probe":
		return "拨测任务"
	default:
		return taskType
	}
}
