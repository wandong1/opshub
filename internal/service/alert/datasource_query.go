package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	agentserver "github.com/ydcloud-dy/opshub/internal/server/agent"
)

// QueryResult 数据源查询结果
type QueryResult struct {
	Labels map[string]string
	Value  float64
}

// QueryDataSource 根据数据源类型执行 instant query
func QueryDataSource(ds *biz.AlertDataSource, expr string) ([]QueryResult, error) {
	switch ds.Type {
	case "prometheus", "victoriametrics":
		return queryPrometheus(ds, expr)
	case "influxdb":
		return queryInfluxDB(ds, expr)
	default:
		return nil, fmt.Errorf("unsupported datasource type: %s", ds.Type)
	}
}

func queryPrometheus(ds *biz.AlertDataSource, expr string) ([]QueryResult, error) {
	// 根据接入方式构建 URL
	var baseURL string
	if ds.AccessMode == "agent" {
		// Agent 代理模式：使用代理转发 URL
		// 格式：http://localhost:9876/api/v1/alert/proxy/datasource/{token}
		baseURL = "http://localhost:9876" + ds.ProxyURL
	} else {
		// 直连模式：直接使用数据源 URL
		baseURL = strings.TrimRight(ds.URL, "/")
	}

	params := url.Values{}
	params.Set("query", expr)
	params.Set("time", fmt.Sprintf("%d", time.Now().Unix()))

	reqURL := fmt.Sprintf("%s/api/v1/query?%s", baseURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 直连模式才需要添加认证（Agent 模式由代理处理器添加）
	if ds.AccessMode == "direct" {
		if ds.Token != "" {
			req.Header.Set("Authorization", "Bearer "+ds.Token)
		} else if ds.Username != "" {
			req.SetBasicAuth(ds.Username, ds.Password)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parsePrometheusResponse(body)
}

func queryInfluxDB(ds *biz.AlertDataSource, expr string) ([]QueryResult, error) {
	// 根据接入方式构建 URL
	var baseURL string
	if ds.AccessMode == "agent" {
		// Agent 代理模式：使用代理转发 URL
		baseURL = "http://localhost:9876" + ds.ProxyURL
	} else {
		// 直连模式：直接使用数据源 URL
		baseURL = strings.TrimRight(ds.URL, "/")
	}

	reqURL := fmt.Sprintf("%s/query", baseURL)

	params := url.Values{}
	params.Set("q", expr)

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 直连模式才需要添加认证（Agent 模式由代理处理器添加）
	if ds.AccessMode == "direct" {
		if ds.Token != "" {
			req.Header.Set("Authorization", "Token "+ds.Token)
		} else if ds.Username != "" {
			req.SetBasicAuth(ds.Username, ds.Password)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var influxResp struct {
		Results []struct {
			Series []struct {
				Columns []string        `json:"columns"`
				Values  [][]interface{} `json:"values"`
			} `json:"series"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &influxResp); err != nil {
		return nil, err
	}

	var results []QueryResult
	for _, r := range influxResp.Results {
		for _, s := range r.Series {
			if len(s.Values) == 0 {
				continue
			}
			row := s.Values[len(s.Values)-1]
			for i, col := range s.Columns {
				if col == "value" || col == "_value" {
					var val float64
					switch v := row[i].(type) {
					case float64:
						val = v
					case json.Number:
						val, _ = v.Float64()
					}
					results = append(results, QueryResult{
						Labels: map[string]string{"__name__": expr},
						Value:  val,
					})
				}
			}
		}
	}
	return results, nil
}

// TestDataSource 测试数据源连通性
func TestDataSource(ds *biz.AlertDataSource) error {
	var testExpr string
	switch ds.Type {
	case "prometheus", "victoriametrics":
		testExpr = "up"
	case "influxdb":
		testExpr = "SHOW MEASUREMENTS LIMIT 1"
	default:
		return fmt.Errorf("unsupported datasource type: %s", ds.Type)
	}
	_, err := QueryDataSource(ds, testExpr)
	return err
}

// QueryDataSourceWithAgent 支持Agent代理的数据源查询
func QueryDataSourceWithAgent(ctx context.Context, ds *biz.AlertDataSource, expr string,
	agentRelationRepo biz.DataSourceAgentRelationRepo, agentHub *agentserver.AgentHub) ([]QueryResult, error) {

	if ds.AccessMode == "agent" {
		return queryViaAgent(ctx, ds, expr, agentRelationRepo, agentHub)
	}
	return QueryDataSource(ds, expr)
}

// queryViaAgent 通过Agent代理查询数据源
func queryViaAgent(ctx context.Context, ds *biz.AlertDataSource, expr string,
	agentRelationRepo biz.DataSourceAgentRelationRepo, agentHub *agentserver.AgentHub) ([]QueryResult, error) {

	// 获取该数据源关联的所有Agent
	rels, err := agentRelationRepo.ListByDataSourceID(ctx, ds.ID)
	if err != nil || len(rels) == 0 {
		return nil, fmt.Errorf("no online agent available for datasource: %d", ds.ID)
	}

	// 选择第一个在线的Agent
	var selectedRel *biz.DataSourceAgentRelation
	if agentHub != nil {
		for _, rel := range rels {
			if agentHub.IsOnline(rel.AgentHostID) {
				selectedRel = rel
				break
			}
		}
	}

	if selectedRel == nil {
		return nil, fmt.Errorf("no online agent available for datasource: %d", ds.ID)
	}

	// 构建目标URL并查询
	params := url.Values{}
	params.Set("query", expr)
	params.Set("time", fmt.Sprintf("%d", time.Now().Unix()))

	targetURL := fmt.Sprintf("http://%s:%d/api/v1/query?%s", ds.Host, ds.Port, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	if ds.Token != "" {
		req.Header.Set("Authorization", "Bearer "+ds.Token)
	} else if ds.Username != "" {
		req.SetBasicAuth(ds.Username, ds.Password)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parsePrometheusResponse(body)
}

// parsePrometheusResponse 解析Prometheus响应
func parsePrometheusResponse(body []byte) ([]QueryResult, error) {
	var promResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, err
	}
	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query error: %s", promResp.Error)
	}

	var results []QueryResult
	for _, item := range promResp.Data.Result {
		if len(item.Value) < 2 {
			continue
		}
		valStr, _ := item.Value[1].(string)
		var val float64
		fmt.Sscanf(valStr, "%f", &val)
		results = append(results, QueryResult{Labels: item.Metric, Value: val})
	}
	return results, nil
}
