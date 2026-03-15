package inspection_mgmt

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// ExportRecordToExcel 导出巡检记录为 Excel
func (s *RecordService) ExportRecordToExcel(ctx context.Context, id uint) (*excelize.File, error) {
	// 获取记录详情
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取记录失败: %w", err)
	}

	// 获取关联信息
	var itemName, groupName, hostName string
	if item, err := s.itemRepo.GetByID(ctx, record.ItemID); err == nil {
		itemName = item.Name
	}
	if record.GroupID > 0 {
		if group, err := s.groupRepo.GetByID(ctx, record.GroupID); err == nil {
			groupName = group.Name
		}
	}
	if record.HostID > 0 && s.hostRepo != nil {
		if host, err := s.hostRepo.GetByID(ctx, record.HostID); err == nil && host != nil {
			hostName = host.Name
		}
	}

	// 创建 Excel 文件
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			// 忽略关闭错误
		}
	}()

	// 创建汇总信息 Sheet
	summarySheet := "巡检汇总"
	f.SetSheetName("Sheet1", summarySheet)

	// 设置汇总信息标题样式
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
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

	contentStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// 设置列宽
	f.SetColWidth(summarySheet, "A", "A", 20)
	f.SetColWidth(summarySheet, "B", "B", 50)

	// 写入标题
	f.SetCellValue(summarySheet, "A1", "巡检记录汇总报告")
	f.MergeCell(summarySheet, "A1", "B1")
	f.SetCellStyle(summarySheet, "A1", "B1", titleStyle)
	f.SetRowHeight(summarySheet, 1, 30)

	// 写入汇总信息
	row := 3
	summaryData := [][]string{
		{"记录ID", strconv.FormatUint(uint64(record.ID), 10)},
		{"巡检组", groupName},
		{"巡检项", itemName},
		{"目标主机", hostName},
		{"执行状态", getStatusText(record.Status)},
		{"断言结果", getAssertionResultText(record.AssertionResult)},
		{"执行时长", fmt.Sprintf("%.2f 秒", record.Duration)},
		{"执行时间", record.ExecutedAt.Format("2006-01-02 15:04:05")},
	}

	for _, data := range summaryData {
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), data[0])
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), data[1])
		f.SetCellStyle(summarySheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), headerStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), contentStyle)
		f.SetRowHeight(summarySheet, row, 25)
		row++
	}

	// 创建详细信息 Sheet
	detailSheet := "巡检详情"
	f.NewSheet(detailSheet)

	// 设置详情 Sheet 列宽
	f.SetColWidth(detailSheet, "A", "A", 20)
	f.SetColWidth(detailSheet, "B", "B", 80)

	// 写入详情标题
	f.SetCellValue(detailSheet, "A1", "巡检详细信息")
	f.MergeCell(detailSheet, "A1", "B1")
	f.SetCellStyle(detailSheet, "A1", "B1", titleStyle)
	f.SetRowHeight(detailSheet, 1, 30)

	// 写入详细信息
	detailRow := 3
	detailData := [][]string{
		{"执行输出", record.Output},
		{"错误信息", record.ErrorMessage},
		{"断言详情", record.AssertionDetails},
		{"提取变量", record.ExtractedVariables},
	}

	for _, data := range detailData {
		if data[1] != "" {
			f.SetCellValue(detailSheet, fmt.Sprintf("A%d", detailRow), data[0])
			f.SetCellValue(detailSheet, fmt.Sprintf("B%d", detailRow), data[1])
			f.SetCellStyle(detailSheet, fmt.Sprintf("A%d", detailRow), fmt.Sprintf("A%d", detailRow), headerStyle)
			f.SetCellStyle(detailSheet, fmt.Sprintf("B%d", detailRow), fmt.Sprintf("B%d", detailRow), contentStyle)

			// 根据内容长度设置行高
			lines := len(data[1])/80 + 1
			if lines > 50 {
				lines = 50
			}
			f.SetRowHeight(detailSheet, detailRow, float64(20*lines))
			detailRow++
		}
	}

	return f, nil
}

func getStatusText(status string) string {
	switch status {
	case "success":
		return "✓ 成功"
	case "failed":
		return "✗ 失败"
	default:
		return status
	}
}

func getAssertionResultText(result string) string {
	switch result {
	case "pass":
		return "✓ 通过"
	case "fail":
		return "✗ 失败"
	case "skip":
		return "- 跳过"
	default:
		return result
	}
}
