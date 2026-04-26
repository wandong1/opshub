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

	// 标题（使用任务名称）
	f.SetCellValue(sheetName, "A"+strconv.Itoa(row), record.TaskName+" - 巡检执行报告")
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

// generateDetailsSheet 生成执行明细 Sheet（包含所有字段）
func (s *ExecutionRecordService) generateDetailsSheet(f *excelize.File, details []*inspectionmgmtdata.InspectionExecutionDetail) error {
	sheetName := "执行明细"

	// 设置列宽（17列）
	colWidths := map[string]float64{
		"A": 6,   // 序号
		"B": 15,  // 巡检组
		"C": 18,  // 巡检项
		"D": 10,  // 巡检级别
		"E": 10,  // 风险等级
		"F": 12,  // 主机
		"G": 15,  // 主机IP
		"H": 12,  // 业务分组
		"I": 10,  // 执行类型
		"J": 10,  // 执行方式
		"K": 35,  // 执行命令
		"L": 35,  // 输出内容
		"M": 10,  // 执行状态
		"N": 10,  // 断言结果
		"O": 10,  // 执行时长
		"P": 20,  // 执行时间
		"Q": 30,  // 断言详情
	}
	for col, width := range colWidths {
		f.SetColWidth(sheetName, col, col, width)
	}

	// 表头样式（深蓝色背景）
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#1F4E78"}, Pattern: 1},
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

	// 数据样式（基础）
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D3D3D3", Style: 1},
			{Type: "top", Color: "#D3D3D3", Style: 1},
			{Type: "bottom", Color: "#D3D3D3", Style: 1},
			{Type: "right", Color: "#D3D3D3", Style: 1},
		},
	})

	// 成功状态样式（绿色背景）
	successStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#006100"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D3D3D3", Style: 1},
			{Type: "top", Color: "#D3D3D3", Style: 1},
			{Type: "bottom", Color: "#D3D3D3", Style: 1},
			{Type: "right", Color: "#D3D3D3", Style: 1},
		},
	})

	// 失败状态样式（红色背景）
	failStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#9C0006"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFC7CE"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D3D3D3", Style: 1},
			{Type: "top", Color: "#D3D3D3", Style: 1},
			{Type: "bottom", Color: "#D3D3D3", Style: 1},
			{Type: "right", Color: "#D3D3D3", Style: 1},
		},
	})

	// 断言通过样式（浅绿色）
	assertPassStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D3D3D3", Style: 1},
			{Type: "top", Color: "#D3D3D3", Style: 1},
			{Type: "bottom", Color: "#D3D3D3", Style: 1},
			{Type: "right", Color: "#D3D3D3", Style: 1},
		},
	})

	// 断言失败样式（浅红色）
	assertFailStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#9C0006"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FCE4D6"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D3D3D3", Style: 1},
			{Type: "top", Color: "#D3D3D3", Style: 1},
			{Type: "bottom", Color: "#D3D3D3", Style: 1},
			{Type: "right", Color: "#D3D3D3", Style: 1},
		},
	})

	// 表头
	headers := []string{
		"序号", "巡检组", "巡检项", "巡检级别", "风险等级",
		"主机", "主机IP", "业务分组", "执行类型", "执行方式",
		"执行命令", "输出内容", "执行状态", "断言结果", "执行时长(秒)",
		"执行时间", "断言详情",
	}
	for i, header := range headers {
		cell := string(rune('A'+i)) + "1"
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetName, 1, 30)

	// 数据行
	for i, detail := range details {
		row := i + 2
		rowStr := strconv.Itoa(row)

		// 基础信息
		f.SetCellValue(sheetName, "A"+rowStr, i+1) // 序号
		f.SetCellValue(sheetName, "B"+rowStr, detail.GroupName)
		f.SetCellValue(sheetName, "C"+rowStr, detail.ItemName)
		f.SetCellValue(sheetName, "D"+rowStr, s.formatLevel(detail.InspectionLevel))
		f.SetCellValue(sheetName, "E"+rowStr, s.formatLevel(detail.RiskLevel))
		f.SetCellValue(sheetName, "F"+rowStr, detail.HostName)
		f.SetCellValue(sheetName, "G"+rowStr, detail.HostIP)
		f.SetCellValue(sheetName, "H"+rowStr, detail.BusinessGroup)

		// 执行配置
		f.SetCellValue(sheetName, "I"+rowStr, s.formatExecutionType(detail.ExecutionType))
		f.SetCellValue(sheetName, "J"+rowStr, detail.ExecutionMode)
		f.SetCellValue(sheetName, "K"+rowStr, s.getExecutionCommand(detail))
		f.SetCellValue(sheetName, "L"+rowStr, detail.Output)

		// 执行结果
		f.SetCellValue(sheetName, "M"+rowStr, s.formatStatus(detail.Status))
		f.SetCellValue(sheetName, "N"+rowStr, s.formatAssertionResult(detail.AssertionResult))
		f.SetCellValue(sheetName, "O"+rowStr, fmt.Sprintf("%.3f", detail.Duration))
		f.SetCellValue(sheetName, "P"+rowStr, detail.ExecutedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, "Q"+rowStr, s.formatAssertionDetails(detail.AssertionDetails))

		// 应用基础样式
		for col := 'A'; col <= 'Q'; col++ {
			cell := string(col) + rowStr
			f.SetCellStyle(sheetName, cell, cell, dataStyle)
		}

		// 应用状态样式
		statusCell := "M" + rowStr
		if detail.Status == "success" {
			f.SetCellStyle(sheetName, statusCell, statusCell, successStyle)
		} else if detail.Status == "failed" {
			f.SetCellStyle(sheetName, statusCell, statusCell, failStyle)
		}

		// 应用断言结果样式
		assertCell := "N" + rowStr
		if detail.AssertionResult == "pass" {
			f.SetCellStyle(sheetName, assertCell, assertCell, assertPassStyle)
		} else if detail.AssertionResult == "fail" {
			f.SetCellStyle(sheetName, assertCell, assertCell, assertFailStyle)
		}

		// 设置行高（根据内容自适应）
		f.SetRowHeight(sheetName, row, 25)
	}

	// 启用自动筛选
	lastRow := len(details) + 1
	f.AutoFilter(sheetName, fmt.Sprintf("A1:Q%d", lastRow), []excelize.AutoFilterOptions{})

	// 冻结首行
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	return nil
}

// formatLevel 格式化级别显示
func (s *ExecutionRecordService) formatLevel(level string) string {
	levelMap := map[string]string{
		"high":   "高",
		"medium": "中",
		"low":    "低",
	}
	if val, ok := levelMap[level]; ok {
		return val
	}
	return level
}

// formatExecutionType 格式化执行类型
func (s *ExecutionRecordService) formatExecutionType(execType string) string {
	typeMap := map[string]string{
		"command": "命令",
		"script":  "脚本",
		"probe":   "拨测",
		"promql":  "PromQL",
	}
	if val, ok := typeMap[execType]; ok {
		return val
	}
	return execType
}

// formatStatus 格式化状态
func (s *ExecutionRecordService) formatStatus(status string) string {
	statusMap := map[string]string{
		"success": "成功",
		"failed":  "失败",
		"running": "运行中",
	}
	if val, ok := statusMap[status]; ok {
		return val
	}
	return status
}

// formatAssertionResult 格式化断言结果
func (s *ExecutionRecordService) formatAssertionResult(result string) string {
	resultMap := map[string]string{
		"pass": "通过",
		"fail": "失败",
		"skip": "跳过",
	}
	if val, ok := resultMap[result]; ok {
		return val
	}
	return result
}

// getExecutionCommand 获取执行命令
func (s *ExecutionRecordService) getExecutionCommand(detail *inspectionmgmtdata.InspectionExecutionDetail) string {
	if detail.Command != "" {
		return detail.Command
	}
	if detail.ScriptContent != "" {
		return fmt.Sprintf("[%s脚本]\n%s", detail.ScriptType, detail.ScriptContent)
	}
	if detail.PromQL != "" {
		return detail.PromQL
	}
	return "-"
}

// formatAssertionDetails 格式化断言详情
func (s *ExecutionRecordService) formatAssertionDetails(details string) string {
	if details == "" {
		return "-"
	}

	// 尝试解析 JSON 并格式化
	var assertionData map[string]interface{}
	if err := json.Unmarshal([]byte(details), &assertionData); err == nil {
		if message, ok := assertionData["message"].(string); ok {
			if pass, ok := assertionData["pass"].(bool); ok {
				if pass {
					return fmt.Sprintf("✓ %s", message)
				}
				return fmt.Sprintf("✗ %s", message)
			}
			return message
		}
	}

	return details
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
	f.SetColWidth(sheetName, "C", "C", 12)
	f.SetColWidth(sheetName, "D", "D", 12)
	f.SetColWidth(sheetName, "E", "E", 20)
	f.SetColWidth(sheetName, "F", "F", 15)
	f.SetColWidth(sheetName, "G", "G", 50)

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
	headers := []string{"巡检组", "巡检项", "巡检级别", "风险等级", "主机", "IP", "错误信息"}
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
		f.SetCellValue(sheetName, "C"+rowStr, detail.InspectionLevel)
		f.SetCellValue(sheetName, "D"+rowStr, detail.RiskLevel)
		f.SetCellValue(sheetName, "E"+rowStr, detail.HostName)
		f.SetCellValue(sheetName, "F"+rowStr, detail.HostIP)
		f.SetCellValue(sheetName, "G"+rowStr, detail.ErrorMessage)

		// 应用样式
		for col := 'A'; col <= 'G'; col++ {
			cell := string(col) + rowStr
			f.SetCellStyle(sheetName, cell, cell, dataStyle)
		}

		f.SetRowHeight(sheetName, row, 30)
	}

	return nil
}
