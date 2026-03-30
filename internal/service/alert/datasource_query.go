package alert

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
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
	baseURL := strings.TrimRight(ds.URL, "/")
	params := url.Values{}
	params.Set("query", expr)
	params.Set("time", fmt.Sprintf("%d", time.Now().Unix()))

	reqURL := fmt.Sprintf("%s/api/v1/query?%s", baseURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
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

func queryInfluxDB(ds *biz.AlertDataSource, expr string) ([]QueryResult, error) {
	baseURL := strings.TrimRight(ds.URL, "/")
	reqURL := fmt.Sprintf("%s/query", baseURL)

	params := url.Values{}
	params.Set("q", expr)

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if ds.Token != "" {
		req.Header.Set("Authorization", "Token "+ds.Token)
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
