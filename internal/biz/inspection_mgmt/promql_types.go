package inspection_mgmt

// PromQLQueryResult Prometheus 查询结果
type PromQLQueryResult struct {
	Status string         `json:"status"`
	Data   PromQLDataBody `json:"data"`
}

// PromQLDataBody Prometheus 数据体
type PromQLDataBody struct {
	ResultType string             `json:"resultType"`
	Result     []PromQLResultItem `json:"result"`
}

// PromQLResultItem Prometheus 结果项
type PromQLResultItem struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`  // Instant Query: [timestamp, value]
	Values [][]interface{}   `json:"values"` // Range Query: [[timestamp, value], ...]
}
