package inspection_mgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
)

// ExportReport 导出巡检执行报告
func (s *ExecutionRecordService) ExportReport(ctx context.Context, executionID uint) (*excelize.File, error) {
	// 获取执行记录
	record, err := s.execRepo.GetRecordByID(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("获取执行记录失败: %w", err)
	}

	// 获取执行明细
	details, err := s.execRepo.GetDetailsByExecutionID(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("获取执行明细失败: %w", err)
	}

	// 创建 Excel 文件
	f := excelize.NewFile()
	defer func() {
		if err != nil {
			f.Close()
		}
	}()

	// 创建三个 Sheet
	f.SetSheetName("Sheet1", "执行概览")
	f.NewSheet("执行明细")
	if record.FailedCount > 0 {
		f.NewSheet("失败项分析")
	}

	// 生成各个 Sheet
	if err := s.generateOverviewSheet(f, record, details); err != nil {
		return nil, err
	}
	if err := s.generateDetailsSheet(f, details); err != nil {
		return nil, err
	}
	if record.FailedCount > 0 {
		if err := s.generateFailureAnalysisSheet(f, details); err != nil {
			return nil, err
		}
	}

	// 设置默认 Sheet
	sheetIndex, _ := f.GetSheetIndex("执行概览")
	f.SetActiveSheet(sheetIndex)

	return f, nil
}

// generateOverviewSheet 生成执行概览 Sheet
func (s *ExecutionRecordService) generateOverviewSheet(f *excelize.File, record *inspectionmgmtdata.InspectionExecutionRecord, details []*inspectionmgmtdata.InspectionExecutionDetail) error {
	sheetName := "执行概览"

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "B", 30)

	// 标题样式
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// 表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})

	// 数据样式
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})

	// 成功样式（绿色）
	successStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#00B050"},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// 失败样式（红色）
	failStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FF0000"},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	row := 1

	// 标题
	f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "巡检执行报告")
	f.MergeCell(sheetName, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row))
	f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), titleStyle)
	f.SetRowHeight(sheetName, row, 30)
	row += 2

	// 任务信息
	f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "任务信息")
	f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), headerStyle)
	row++

	taskInfo := [][]string{
		{"任务名称", record.TaskName},
		{"任务ID", fmt.Sprintf("%d", record.TaskID)},
		{"执行状态", record.Status},
		{"开始时间", record.StartedAt.Format("2006-01-02 15:04:05")},
	}
	if record.CompletedAt != nil {
		taskInfo = append(taskInfo, []string{"完成时间", record.CompletedAt.Format("2006-01-02 15:04:05")})
	}
	taskInfo = append(taskInfo, []string{"执行时长", fmt.Sprintf("%.2f 秒", record.Duration)})

	// 解析巡检组名称
	var groupNames []string
	if record.GroupNames != "" {
		json.Unmarshal([]byte(record.GroupNames), &groupNames)
	}
	if len(groupNames) > 0 {
		taskInfo = append(taskInfo, []string{"巡检组", fmt.Sprintf("%v", groupNames)})
	}

	for _, info := range taskInfo {
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), info[0])
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), info[1])
		f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), dataStyle)
		row++
	}
	row++

	// 执行统计
	f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "执行统计")
	f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), headerStyle)
	row++

	stats := [][]interface{}{
		{"总巡检项数", record.TotalItems},
		{"总主机数", record.TotalHosts},
		{"总执行次数", record.TotalExecutions},
		{"成功次数", record.SuccessCount},
		{"失败次数", record.FailedCount},
	}

	for _, stat := range stats {
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), stat[0])
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), stat[1])

		// 成功/失败数使用不同颜色
		if stat[0] == "成功次数" && record.SuccessCount > 0 {
			f.SetCellStyle(sheetName, "B"+strconv.Itoa(row), "B"+strconv.Itoa(row), successStyle)
		} else if stat[0] == "失败次数" && record.FailedCount > 0 {
			f.SetCellStyle(sheetName, "B"+strconv.Itoa(row), "B"+strconv.Itoa(row), failStyle)
		} else {
			f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), dataStyle)
		}
		row++
	}
	row++

	// 断言统计
	f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "断言统计")
	f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), headerStyle)
	row++

	assertionStats := [][]interface{}{
		{"断言通过", record.AssertionPassCount},
		{"断言失败", record.AssertionFailCount},
		{"断言跳过", record.AssertionSkipCount},
	}

	for _, stat := range assertionStats {
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), stat[0])
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), stat[1])

		if stat[0] == "断言通过" && record.AssertionPassCount > 0 {
			f.SetCellStyle(sheetName, "B"+strconv.Itoa(row), "B"+strconv.Itoa(row), successStyle)
		} else if stat[0] == "断言失败" && record.AssertionFailCount > 0 {
			f.SetCellStyle(sheetName, "B"+strconv.Itoa(row), "B"+strconv.Itoa(row), failStyle)
		} else {
			f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), dataStyle)
		}
		row++
	}

	return nil
}

// generateDetailsSheet 生成执行明细 Sheet
func (s *ExecutionRecordService) generateDetailsSheet(f *excelize.File, details []*inspectionmgmtdata.InspectionExecutionDetail) error {
	sheetName := "执行明细"

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 12)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 20)
	f.SetColWidth(sheetName, "D", "D", 20)
	f.SetColWidth(sheetName, "E", "E", 15)
	f.SetColWidth(sheetName, "F", "F", 10)
	f.SetColWidth(sheetName, "G", "G", 12)
	f.SetColWidth(sheetName, "H", "H", 12)
	f.SetColWidth(sheetName, "I", "I", 20)
	f.SetColWidth(sheetName, "J", "J", 40)

	// 表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})

	// 数据样式
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D3D3D3", Style: 1},
			{Type: "top", Color: "#D3D3D3", Style: 1},
			{Type: "bottom", Color: "#D3D3D3", Style: 1},
			{Type: "right", Color: "#D3D3D3", Style: 1},
		},
	})

	// 表头
	headers := []string{"ID", "巡检组", "巡检项", "主机", "IP", "状态", "断言结果", "时长(秒)", "执行时间", "错误信息"}
	for i, header := range headers {
		cell := string(rune('A'+i)) + "1"
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetName, 1, 25)

	// 数据行
	for i, detail := range details {
		row := i + 2
		rowStr := strconv.Itoa(row)

		f.SetCellValue(sheetName, "A"+rowStr, detail.ID)
		f.SetCellValue(sheetName, "B"+rowStr, detail.GroupName)
		f.SetCellValue(sheetName, "C"+rowStr, detail.ItemName)
		f.SetCellValue(sheetName, "D"+rowStr, detail.HostName)
		f.SetCellValue(sheetName, "E"+rowStr, detail.HostIP)
		f.SetCellValue(sheetName, "F"+rowStr, detail.Status)
		f.SetCellValue(sheetName, "G"+rowStr, detail.AssertionResult)
		f.SetCellValue(sheetName, "H"+rowStr, fmt.Sprintf("%.2f", detail.Duration))
		f.SetCellValue(sheetName, "I"+rowStr, detail.ExecutedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, "J"+rowStr, detail.ErrorMessage)

		// 应用样式
		for col := 'A'; col <= 'J'; col++ {
			cell := string(col) + rowStr
			f.SetCellStyle(sheetName, cell, cell, dataStyle)
		}

		f.SetRowHeight(sheetName, row, 20)
	}

	// 启用自动筛选
	lastRow := len(details) + 1
	f.AutoFilter(sheetName, fmt.Sprintf("A1:J%d", lastRow), []excelize.AutoFilterOptions{})

	return nil
}

// generateFailureAnalysisSheet 生成失败项分析 Sheet
func (s *ExecutionRecordService) generateFailureAnalysisSheet(f *excelize.File, details []*inspectionmgmtdata.InspectionExecutionDetail) error {
	sheetName := "失败项分析"

	// 筛选失败项
	var failedDetails []*inspectionmgmtdata.InspectionExecutionDetail
	for _, detail := range details {
		if detail.Status == "failed" {
			failedDetails = append(failedDetails, detail)
		}
	}

	if len(failedDetails) == 0 {
		return nil
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 20)
	f.SetColWidth(sheetName, "D", "D", 15)
	f.SetColWidth(sheetName, "E", "E", 50)

	// 表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FF0000"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})

	// 数据样式
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D3D3D3", Style: 1},
			{Type: "top", Color: "#D3D3D3", Style: 1},
			{Type: "bottom", Color: "#D3D3D3", Style: 1},
			{Type: "right", Color: "#D3D3D3", Style: 1},
		},
	})

	// 表头
	headers := []string{"巡检组", "巡检项", "主机", "IP", "错误信息"}
	for i, header := range headers {
		cell := string(rune('A'+i)) + "1"
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetName, 1, 25)

	// 数据行
	for i, detail := range failedDetails {
		row := i + 2
		rowStr := strconv.Itoa(row)

		f.SetCellValue(sheetName, "A"+rowStr, detail.GroupName)
		f.SetCellValue(sheetName, "B"+rowStr, detail.ItemName)
		f.SetCellValue(sheetName, "C"+rowStr, detail.HostName)
		f.SetCellValue(sheetName, "D"+rowStr, detail.HostIP)
		f.SetCellValue(sheetName, "E"+rowStr, detail.ErrorMessage)

		// 应用样式
		for col := 'A'; col <= 'E'; col++ {
			cell := string(col) + rowStr
			f.SetCellStyle(sheetName, cell, cell, dataStyle)
		}

		f.SetRowHeight(sheetName, row, 30)
	}

	return nil
}
