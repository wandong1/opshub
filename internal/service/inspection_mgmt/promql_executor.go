package inspection_mgmt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"

	alertbiz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	inspectionmgmtbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection_mgmt"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
)

// PromQLExecutor PromQL 执行器
type PromQLExecutor struct {
	datasourceRepo *alertdata.DataSourceRepo
	httpClient     *http.Client
	semaphore      chan struct{} // 并发控制信号量
	platformURL    string        // 平台地址（用于拼接 Agent 代理 URL）
}

// NewPromQLExecutor 创建 PromQL 执行器
func NewPromQLExecutor(datasourceRepo *alertdata.DataSourceRepo, maxConcurrency int) *PromQLExecutor {
	if maxConcurrency <= 0 {
		maxConcurrency = 50 // 默认并发数
	}

	return &PromQLExecutor{
		datasourceRepo: datasourceRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		semaphore:   make(chan struct{}, maxConcurrency),
		platformURL: "http://localhost:9876", // 默认平台地址
	}
}

// PromQLExecutionResult PromQL 执行结果
type PromQLExecutionResult struct {
	HostID       uint
	HostName     string
	HostIP       string
	DataSourceID uint
	PromQL       string
	PromQLResult string
	MetricValue  float64
	MetricLabels map[string]string
	Status       string // success/error
	ErrorMessage string
	Duration     float64 // 执行耗时（秒）
}

// ExecuteBatch 批量执行 PromQL 查询（带并发控制）
func (e *PromQLExecutor) ExecuteBatch(ctx context.Context, promql string, hosts []*assetbiz.Host, datasourceID uint, queryType string) []*PromQLExecutionResult {
	var wg sync.WaitGroup
	results := make([]*PromQLExecutionResult, len(hosts))

	for i, host := range hosts {
		wg.Add(1)
		go func(index int, h *assetbiz.Host) {
			defer wg.Done()

			// 获取信号量（控制并发）
			e.semaphore <- struct{}{}
			defer func() { <-e.semaphore }()

			// 执行单个查询
			results[index] = e.Execute(ctx, promql, h, datasourceID, queryType)
		}(i, host)
	}

	wg.Wait()
	return results
}

// Execute 执行单个 PromQL 查询
func (e *PromQLExecutor) Execute(ctx context.Context, promql string, host *assetbiz.Host, datasourceID uint, queryType string) *PromQLExecutionResult {
	startTime := time.Now()
	result := &PromQLExecutionResult{
		HostID:       host.ID,
		HostName:     host.Name,
		HostIP:       host.IP,
		DataSourceID: datasourceID,
		Status:       "error",
	}

	// 1. 渲染 PromQL 模板（注入预置变量）
	renderedPromQL, err := e.renderPromQLTemplate(promql, host)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("PromQL 模板渲染失败: %v", err)
		result.Duration = time.Since(startTime).Seconds()
		return result
	}
	result.PromQL = renderedPromQL

	// 2. 获取数据源配置
	ds, err := e.datasourceRepo.GetByID(ctx, datasourceID)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("获取数据源失败: %v", err)
		result.Duration = time.Since(startTime).Seconds()
		return result
	}

	// 3. 执行查询
	queryResult, err := e.queryDataSource(ctx, ds, renderedPromQL, queryType)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("查询失败: %v", err)
		result.Duration = time.Since(startTime).Seconds()
		return result
	}

	result.PromQLResult = queryResult
	result.Duration = time.Since(startTime).Seconds()

	// 4. 解析查询结果（提取指标值和标签）
	metricValue, metricLabels, err := e.parseQueryResult(queryResult, queryType)
	if err != nil {
		// 解析失败视为错误，不设置 success 状态
		result.Status = "error"
		result.ErrorMessage = fmt.Sprintf("结果解析失败: %v", err)
	} else {
		result.Status = "success"
		result.MetricValue = metricValue
		result.MetricLabels = metricLabels
	}

	return result
}

// renderPromQLTemplate 渲染 PromQL 模板
func (e *PromQLExecutor) renderPromQLTemplate(promql string, host *assetbiz.Host) (string, error) {
	tmpl, err := template.New("promql").Parse(promql)
	if err != nil {
		return "", err
	}

	// 构建预置变量
	data := map[string]interface{}{
		"Instance": e.getInstanceLabel(host),
		"IP":       host.IP,
		"Hostname": host.Name,
		"Labels":   e.parseHostLabels(host.Tags),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// getInstanceLabel 获取 instance 标签（格式：IP:Port）
func (e *PromQLExecutor) getInstanceLabel(host *assetbiz.Host) string {
	// 使用主机的 ExporterPort 字段
	if host.ExporterPort > 0 {
		return fmt.Sprintf("%s:%d", host.IP, host.ExporterPort)
	}

	// 默认使用 9100（node_exporter 默认端口）
	return fmt.Sprintf("%s:9100", host.IP)
}

// parseHostLabels 解析主机标签
func (e *PromQLExecutor) parseHostLabels(tags string) map[string]string {
	labels := make(map[string]string)
	if tags == "" {
		return labels
	}

	// 假设标签格式为 "key1:value1,key2:value2"
	for _, tag := range strings.Split(tags, ",") {
		parts := strings.SplitN(strings.TrimSpace(tag), ":", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}

	return labels
}

// queryDataSource 查询数据源
func (e *PromQLExecutor) queryDataSource(ctx context.Context, ds *alertbiz.AlertDataSource, promql string, queryType string) (string, error) {
	// 构建查询 URL
	var apiPath string
	params := url.Values{}
	params.Set("query", promql)

	if queryType == "range" {
		apiPath = "/api/v1/query_range"
		// Range Query 参数
		endTime := time.Now()
		startTime := endTime.Add(-1 * time.Hour) // 默认查询过去 1 小时
		params.Set("start", fmt.Sprintf("%d", startTime.Unix()))
		params.Set("end", fmt.Sprintf("%d", endTime.Unix()))
		params.Set("step", "60") // 默认步长 60 秒
	} else {
		apiPath = "/api/v1/query"
		// Instant Query 参数
		params.Set("time", fmt.Sprintf("%d", time.Now().Unix()))
	}

	// 根据接入方式选择查询路径
	var queryURL string
	if ds.AccessMode == "agent" && ds.ProxyURL != "" {
		// Agent 代理模式：拼接完整的代理 URL
		// ProxyURL 格式：/api/v1/alert/proxy/datasource/{token}
		// 拼接后格式：http://localhost:9876/api/v1/alert/proxy/datasource/{token}/api/v1/query?...
		queryURL = strings.TrimRight(e.platformURL, "/") + strings.TrimRight(ds.ProxyURL, "/") + apiPath + "?" + params.Encode()
	} else {
		// 直连模式：使用 URL
		queryURL = strings.TrimRight(ds.URL, "/") + apiPath + "?" + params.Encode()
	}

	fmt.Printf("PromQL 查询: %s\n", queryURL)
	// 发起 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return "", err
	}

	// 添加认证
	if ds.Username != "" && ds.Password != "" {
		req.SetBasicAuth(ds.Username, ds.Password)
	}
	if ds.Token != "" {
		req.Header.Set("Authorization", "Bearer "+ds.Token)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("查询失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// parseQueryResult 解析查询结果
func (e *PromQLExecutor) parseQueryResult(queryResult string, queryType string) (float64, map[string]string, error) {
	var result inspectionmgmtbiz.PromQLQueryResult
	if err := json.Unmarshal([]byte(queryResult), &result); err != nil {
		return 0, nil, fmt.Errorf("JSON 解析失败: %v", err)
	}

	if result.Status != "success" {
		return 0, nil, fmt.Errorf("查询状态异常: %s", result.Status)
	}

	if len(result.Data.Result) == 0 {
		return 0, nil, fmt.Errorf("查询无结果")
	}

	// 提取第一个结果的指标值和标签
	firstResult := result.Data.Result[0]
	metricLabels := firstResult.Metric

	var metricValue float64
	var err error

	if queryType == "range" {
		// Range Query: 取最后一个时间点的值
		if len(firstResult.Values) > 0 {
			lastValue := firstResult.Values[len(firstResult.Values)-1]
			if len(lastValue) >= 2 {
				if valueStr, ok := lastValue[1].(string); ok {
					metricValue, err = parseFloat(valueStr)
				}
			}
		}
	} else {
		// Instant Query: 直接取 value[1]
		if len(firstResult.Value) >= 2 {
			if valueStr, ok := firstResult.Value[1].(string); ok {
				metricValue, err = parseFloat(valueStr)
			}
		}
	}

	if err != nil {
		return 0, metricLabels, fmt.Errorf("指标值解析失败: %v", err)
	}

	return metricValue, metricLabels, nil
}

// parseFloat 解析浮点数（支持特殊值）
func parseFloat(s string) (float64, error) {
	// 处理特殊值
	switch s {
	case "NaN":
		return 0, fmt.Errorf("值为 NaN")
	case "+Inf":
		return 0, fmt.Errorf("值为 +Inf")
	case "-Inf":
		return 0, fmt.Errorf("值为 -Inf")
	}

	var value float64
	_, err := fmt.Sscanf(s, "%f", &value)
	return value, err
}
