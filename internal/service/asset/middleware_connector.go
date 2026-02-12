// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package asset

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/xdg-go/scram"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	milvusclient "github.com/milvus-io/milvus-sdk-go/v2/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectResult 连接测试结果
type ConnectResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Version string `json:"version"`
	Latency int64  `json:"latency"` // 毫秒
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	Command  string `json:"command" binding:"required"`
	Database string `json:"database"`
	Limit    int    `json:"limit"`
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	Columns      []string        `json:"columns,omitempty"`
	Rows         [][]interface{} `json:"rows,omitempty"`
	AffectedRows int64           `json:"affectedRows,omitempty"`
	Message      string          `json:"message,omitempty"`
	RawResult    interface{}     `json:"rawResult,omitempty"`
	Duration     int64           `json:"duration,omitempty"` // 执行耗时(ms)
}

// MiddlewareConnector 中间件连接器接口
type MiddlewareConnector interface {
	TestConnection(ctx context.Context, mw *assetbiz.Middleware) (*ConnectResult, error)
	Execute(ctx context.Context, mw *assetbiz.Middleware, req *ExecuteRequest) (*ExecuteResult, error)
}

// GetConnector 根据中间件类型获取连接器
func GetConnector(mwType string) (MiddlewareConnector, error) {
	switch mwType {
	case assetbiz.MiddlewareTypeMySQL:
		return &MySQLConnector{}, nil
	case assetbiz.MiddlewareTypeRedis:
		return &RedisConnector{}, nil
	case assetbiz.MiddlewareTypeClickHouse:
		return &ClickHouseConnector{}, nil
	case assetbiz.MiddlewareTypeMongoDB:
		return &MongoDBConnector{}, nil
	case assetbiz.MiddlewareTypeKafka:
		return &KafkaConnector{}, nil
	case assetbiz.MiddlewareTypeMilvus:
		return &MilvusConnector{}, nil
	default:
		return nil, fmt.Errorf("不支持的中间件类型: %s", mwType)
	}
}

// ===== MySQL Connector =====

type MySQLConnector struct{}

func (c *MySQLConnector) buildDSN(mw *assetbiz.Middleware) string {
	dbName := mw.DatabaseName
	if dbName == "" {
		dbName = "information_schema"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=10s&readTimeout=30s&parseTime=true",
		mw.Username, mw.Password, mw.Host, mw.Port, dbName)
	// 解析额外连接参数
	if mw.ConnectionParams != "" {
		var params map[string]string
		if err := json.Unmarshal([]byte(mw.ConnectionParams), &params); err == nil {
			for k, v := range params {
				dsn += "&" + k + "=" + v
			}
		}
	}
	return dsn
}

func (c *MySQLConnector) TestConnection(ctx context.Context, mw *assetbiz.Middleware) (*ConnectResult, error) {
	start := time.Now()
	db, err := sql.Open("mysql", c.buildDSN(mw))
	if err != nil {
		return &ConnectResult{Success: false, Message: fmt.Sprintf("连接失败: %v", err)}, nil
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return &ConnectResult{Success: false, Message: fmt.Sprintf("连接失败: %v", err)}, nil
	}

	var version string
	db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)

	return &ConnectResult{
		Success: true,
		Message: "连接成功",
		Version: version,
		Latency: time.Since(start).Milliseconds(),
	}, nil
}

func (c *MySQLConnector) Execute(ctx context.Context, mw *assetbiz.Middleware, req *ExecuteRequest) (*ExecuteResult, error) {
	db, err := sql.Open("mysql", c.buildDSN(mw))
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	defer db.Close()

	command := strings.TrimSpace(req.Command)
	upperCmd := strings.ToUpper(command)

	start := time.Now()

	// SELECT 查询
	if strings.HasPrefix(upperCmd, "SELECT") || strings.HasPrefix(upperCmd, "SHOW") || strings.HasPrefix(upperCmd, "DESC") || strings.HasPrefix(upperCmd, "EXPLAIN") {
		limit := req.Limit
		if limit <= 0 {
			limit = 100
		}
		// 对 SELECT 语句自动添加 LIMIT（如果没有的话）
		if strings.HasPrefix(upperCmd, "SELECT") && !strings.Contains(upperCmd, "LIMIT") {
			command = fmt.Sprintf("%s LIMIT %d", command, limit)
		}

		rows, err := db.QueryContext(ctx, command)
		if err != nil {
			return nil, fmt.Errorf("查询失败: %w", err)
		}
		defer rows.Close()

		columns, _ := rows.Columns()
		var resultRows [][]interface{}

		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			if err := rows.Scan(valuePtrs...); err != nil {
				continue
			}
			row := make([]interface{}, len(columns))
			for i, v := range values {
				if b, ok := v.([]byte); ok {
					row[i] = string(b)
				} else {
					row[i] = v
				}
			}
			resultRows = append(resultRows, row)
		}

		return &ExecuteResult{Columns: columns, Rows: resultRows, Duration: time.Since(start).Milliseconds()}, nil
	}

	// 非查询语句
	result, err := db.ExecContext(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("执行失败: %w", err)
	}
	affected, _ := result.RowsAffected()
	return &ExecuteResult{AffectedRows: affected, Message: fmt.Sprintf("执行成功，影响 %d 行", affected), Duration: time.Since(start).Milliseconds()}, nil
}

// ===== Redis Connector =====

type RedisConnector struct{}

func (c *RedisConnector) buildOptions(mw *assetbiz.Middleware) *redis.Options {
	opts := &redis.Options{
		Addr:        fmt.Sprintf("%s:%d", mw.Host, mw.Port),
		Password:    mw.Password,
		DialTimeout: 10 * time.Second,
		ReadTimeout: 30 * time.Second,
	}
	if mw.DatabaseName != "" {
		var db int
		fmt.Sscanf(mw.DatabaseName, "%d", &db)
		opts.DB = db
	}
	if mw.Username != "" {
		opts.Username = mw.Username
	}
	return opts
}

func (c *RedisConnector) TestConnection(ctx context.Context, mw *assetbiz.Middleware) (*ConnectResult, error) {
	start := time.Now()
	rdb := redis.NewClient(c.buildOptions(mw))
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return &ConnectResult{Success: false, Message: fmt.Sprintf("连接失败: %v", err)}, nil
	}

	info, _ := rdb.Info(ctx, "server").Result()
	var version string
	for _, line := range strings.Split(info, "\n") {
		if strings.HasPrefix(line, "redis_version:") {
			version = strings.TrimPrefix(line, "redis_version:")
			version = strings.TrimSpace(version)
			break
		}
	}

	return &ConnectResult{
		Success: true,
		Message: "连接成功",
		Version: version,
		Latency: time.Since(start).Milliseconds(),
	}, nil
}

func (c *RedisConnector) Execute(ctx context.Context, mw *assetbiz.Middleware, req *ExecuteRequest) (*ExecuteResult, error) {
	rdb := redis.NewClient(c.buildOptions(mw))
	defer rdb.Close()

	parts := parseRedisCommand(req.Command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("命令不能为空")
	}

	start := time.Now()

	args := make([]interface{}, len(parts))
	for i, p := range parts {
		args[i] = p
	}

	result, err := rdb.Do(ctx, args...).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("执行失败: %w", err)
	}

	return &ExecuteResult{
		RawResult: formatRedisResult(result, err),
		Message:   formatRedisResultString(result, err),
		Duration:  time.Since(start).Milliseconds(),
	}, nil
}

// parseRedisCommand 解析 Redis 命令，支持引号包裹的参数
// 例如: SET key "hello world" -> ["SET", "key", "hello world"]
func parseRedisCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for _, ch := range cmd {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' && (inSingle || inDouble) {
			escaped = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if (ch == ' ' || ch == '\t') && !inSingle && !inDouble {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(ch)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// formatRedisResult 格式化 Redis 结果为结构化数据
func formatRedisResult(result interface{}, err error) interface{} {
	if err == redis.Nil {
		return nil
	}
	switch v := result.(type) {
	case []interface{}:
		formatted := make([]interface{}, len(v))
		for i, item := range v {
			formatted[i] = formatRedisResult(item, nil)
		}
		return formatted
	default:
		return v
	}
}

// formatRedisResultString 格式化 Redis 结果为可读字符串
func formatRedisResultString(result interface{}, err error) string {
	if err == redis.Nil {
		return "(nil)"
	}
	switch v := result.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case int64:
		return fmt.Sprintf("(integer) %d", v)
	case []interface{}:
		if len(v) == 0 {
			return "(empty array)"
		}
		var sb strings.Builder
		for i, item := range v {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("%d) %s", i+1, formatRedisResultString(item, nil)))
		}
		return sb.String()
	case nil:
		return "(nil)"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ===== ClickHouse Connector (基于 database/sql) =====

type ClickHouseConnector struct{}

func (c *ClickHouseConnector) buildDSN(mw *assetbiz.Middleware) string {
	dbName := mw.DatabaseName
	if dbName == "" {
		dbName = "default"
	}
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?dial_timeout=10s&read_timeout=30s",
		mw.Username, mw.Password, mw.Host, mw.Port, dbName)
	if mw.ConnectionParams != "" {
		var params map[string]string
		if err := json.Unmarshal([]byte(mw.ConnectionParams), &params); err == nil {
			for k, v := range params {
				dsn += "&" + k + "=" + v
			}
		}
	}
	return dsn
}

func (c *ClickHouseConnector) TestConnection(ctx context.Context, mw *assetbiz.Middleware) (*ConnectResult, error) {
	start := time.Now()
	db, err := sql.Open("clickhouse", c.buildDSN(mw))
	if err != nil {
		return &ConnectResult{Success: false, Message: fmt.Sprintf("连接失败: %v", err)}, nil
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return &ConnectResult{Success: false, Message: fmt.Sprintf("连接失败: %v", err)}, nil
	}

	var version string
	db.QueryRowContext(ctx, "SELECT version()").Scan(&version)

	return &ConnectResult{
		Success: true,
		Message: "连接成功",
		Version: version,
		Latency: time.Since(start).Milliseconds(),
	}, nil
}

func (c *ClickHouseConnector) Execute(ctx context.Context, mw *assetbiz.Middleware, req *ExecuteRequest) (*ExecuteResult, error) {
	db, err := sql.Open("clickhouse", c.buildDSN(mw))
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	defer db.Close()

	command := strings.TrimSpace(req.Command)
	upperCmd := strings.ToUpper(command)

	start := time.Now()

	if strings.HasPrefix(upperCmd, "SELECT") || strings.HasPrefix(upperCmd, "SHOW") ||
		strings.HasPrefix(upperCmd, "DESC") || strings.HasPrefix(upperCmd, "EXPLAIN") ||
		strings.HasPrefix(upperCmd, "EXISTS") {
		limit := req.Limit
		if limit <= 0 {
			limit = 100
		}
		if strings.HasPrefix(upperCmd, "SELECT") && !strings.Contains(upperCmd, "LIMIT") {
			command = fmt.Sprintf("%s LIMIT %d", command, limit)
		}

		rows, err := db.QueryContext(ctx, command)
		if err != nil {
			return nil, fmt.Errorf("查询失败: %w", err)
		}
		defer rows.Close()

		columns, _ := rows.Columns()
		var resultRows [][]interface{}

		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			if err := rows.Scan(valuePtrs...); err != nil {
				continue
			}
			row := make([]interface{}, len(columns))
			for i, v := range values {
				switch val := v.(type) {
				case []byte:
					row[i] = string(val)
				case time.Time:
					row[i] = val.Format("2006-01-02 15:04:05")
				default:
					row[i] = val
				}
			}
			resultRows = append(resultRows, row)
		}

		return &ExecuteResult{Columns: columns, Rows: resultRows, Duration: time.Since(start).Milliseconds()}, nil
	}

	// 非查询语句
	result, err := db.ExecContext(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("执行失败: %w", err)
	}
	affected, _ := result.RowsAffected()
	return &ExecuteResult{AffectedRows: affected, Message: fmt.Sprintf("执行成功，影响 %d 行", affected), Duration: time.Since(start).Milliseconds()}, nil
}

// ===== MongoDB Connector =====

type MongoDBConnector struct{}

func (c *MongoDBConnector) buildURI(mw *assetbiz.Middleware) string {
	uri := "mongodb://"
	if mw.Username != "" {
		uri += url.QueryEscape(mw.Username) + ":" + url.QueryEscape(mw.Password) + "@"
	}
	uri += fmt.Sprintf("%s:%d/?connectTimeoutMS=10000", mw.Host, mw.Port)
	if mw.ConnectionParams != "" {
		var params map[string]string
		if err := json.Unmarshal([]byte(mw.ConnectionParams), &params); err == nil {
			for k, v := range params {
				uri += "&" + k + "=" + v
			}
		}
	}
	return uri
}

func (c *MongoDBConnector) connect(ctx context.Context, mw *assetbiz.Middleware) (*mongo.Client, error) {
	clientOpts := options.Client().ApplyURI(c.buildURI(mw))
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, err
	}
	return client, nil
}

func (c *MongoDBConnector) TestConnection(ctx context.Context, mw *assetbiz.Middleware) (*ConnectResult, error) {
	start := time.Now()
	client, err := c.connect(ctx, mw)
	if err != nil {
		return &ConnectResult{Success: false, Message: fmt.Sprintf("连接失败: %v", err)}, nil
	}
	defer client.Disconnect(ctx)

	var buildInfo bson.M
	err = client.Database("admin").RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&buildInfo)
	version := ""
	if err == nil {
		if v, ok := buildInfo["version"]; ok {
			version = fmt.Sprintf("%v", v)
		}
	}

	return &ConnectResult{
		Success: true,
		Message: "连接成功",
		Version: version,
		Latency: time.Since(start).Milliseconds(),
	}, nil
}

func (c *MongoDBConnector) Execute(ctx context.Context, mw *assetbiz.Middleware, req *ExecuteRequest) (*ExecuteResult, error) {
	client, err := c.connect(ctx, mw)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	defer client.Disconnect(ctx)

	// 解析 JSON 命令
	var cmd map[string]interface{}
	if err := json.Unmarshal([]byte(req.Command), &cmd); err != nil {
		return nil, fmt.Errorf("命令格式错误，需要 JSON 格式: %w", err)
	}

	action, _ := cmd["action"].(string)
	dbName, _ := cmd["database"].(string)
	if dbName == "" {
		dbName = req.Database
	}
	if dbName == "" {
		dbName = mw.DatabaseName
	}
	if dbName == "" {
		return nil, fmt.Errorf("请指定数据库名称")
	}

	collName, _ := cmd["collection"].(string)
	start := time.Now()

	switch strings.ToLower(action) {
	case "find":
		if collName == "" {
			return nil, fmt.Errorf("请指定集合名称")
		}
		filter := toBsonDoc(cmd["filter"])
		sortDoc := toBsonDoc(cmd["sort"])
		limit := int64(100)
		if l, ok := cmd["limit"].(float64); ok && l > 0 {
			limit = int64(l)
		}

		opts := options.Find().SetLimit(limit)
		if sortDoc != nil {
			opts.SetSort(sortDoc)
		}

		cursor, err := client.Database(dbName).Collection(collName).Find(ctx, filter, opts)
		if err != nil {
			return nil, fmt.Errorf("查询失败: %w", err)
		}
		defer cursor.Close(ctx)

		var results []bson.M
		if err := cursor.All(ctx, &results); err != nil {
			return nil, fmt.Errorf("读取结果失败: %w", err)
		}

		jsonBytes, _ := json.Marshal(results)
		return &ExecuteResult{
			RawResult: json.RawMessage(jsonBytes),
			Message:   fmt.Sprintf("查询成功，返回 %d 条文档", len(results)),
			Duration:  time.Since(start).Milliseconds(),
		}, nil

	case "insertone":
		if collName == "" {
			return nil, fmt.Errorf("请指定集合名称")
		}
		doc := toBsonDoc(cmd["document"])
		if doc == nil {
			return nil, fmt.Errorf("请提供要插入的文档 (document)")
		}
		result, err := client.Database(dbName).Collection(collName).InsertOne(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("插入失败: %w", err)
		}
		return &ExecuteResult{
			RawResult: map[string]interface{}{"insertedId": result.InsertedID},
			Message:   fmt.Sprintf("插入成功，ID: %v", result.InsertedID),
			Duration:  time.Since(start).Milliseconds(),
		}, nil

	case "updatemany":
		if collName == "" {
			return nil, fmt.Errorf("请指定集合名称")
		}
		filter := toBsonDoc(cmd["filter"])
		update := toBsonDoc(cmd["update"])
		if update == nil {
			return nil, fmt.Errorf("请提供更新操作 (update)")
		}
		result, err := client.Database(dbName).Collection(collName).UpdateMany(ctx, filter, update)
		if err != nil {
			return nil, fmt.Errorf("更新失败: %w", err)
		}
		return &ExecuteResult{
			AffectedRows: result.ModifiedCount,
			Message:      fmt.Sprintf("更新成功，匹配 %d 条，修改 %d 条", result.MatchedCount, result.ModifiedCount),
			Duration:     time.Since(start).Milliseconds(),
		}, nil

	case "deletemany":
		if collName == "" {
			return nil, fmt.Errorf("请指定集合名称")
		}
		filter := toBsonDoc(cmd["filter"])
		if filter == nil {
			return nil, fmt.Errorf("请提供过滤条件 (filter)，不允许空条件删除")
		}
		result, err := client.Database(dbName).Collection(collName).DeleteMany(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("删除失败: %w", err)
		}
		return &ExecuteResult{
			AffectedRows: result.DeletedCount,
			Message:      fmt.Sprintf("删除成功，删除 %d 条文档", result.DeletedCount),
			Duration:     time.Since(start).Milliseconds(),
		}, nil

	case "aggregate":
		if collName == "" {
			return nil, fmt.Errorf("请指定集合名称")
		}
		pipeline, ok := cmd["pipeline"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("请提供聚合管道 (pipeline)")
		}
		var bsonPipeline []bson.M
		for _, stage := range pipeline {
			if m, ok := stage.(map[string]interface{}); ok {
				bsonPipeline = append(bsonPipeline, bson.M(m))
			}
		}
		cursor, err := client.Database(dbName).Collection(collName).Aggregate(ctx, bsonPipeline)
		if err != nil {
			return nil, fmt.Errorf("聚合失败: %w", err)
		}
		defer cursor.Close(ctx)

		var results []bson.M
		if err := cursor.All(ctx, &results); err != nil {
			return nil, fmt.Errorf("读取结果失败: %w", err)
		}

		jsonBytes, _ := json.Marshal(results)
		return &ExecuteResult{
			RawResult: json.RawMessage(jsonBytes),
			Message:   fmt.Sprintf("聚合成功，返回 %d 条结果", len(results)),
			Duration:  time.Since(start).Milliseconds(),
		}, nil

	case "count":
		if collName == "" {
			return nil, fmt.Errorf("请指定集合名称")
		}
		filter := toBsonDoc(cmd["filter"])
		count, err := client.Database(dbName).Collection(collName).CountDocuments(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("计数失败: %w", err)
		}
		return &ExecuteResult{
			RawResult: map[string]interface{}{"count": count},
			Message:   fmt.Sprintf("文档数量: %d", count),
			Duration:  time.Since(start).Milliseconds(),
		}, nil

	case "runcommand":
		cmdDoc := toBsonDoc(cmd["command"])
		if cmdDoc == nil {
			return nil, fmt.Errorf("请提供要执行的命令 (command)")
		}
		var result bson.M
		err := client.Database(dbName).RunCommand(ctx, cmdDoc).Decode(&result)
		if err != nil {
			return nil, fmt.Errorf("执行命令失败: %w", err)
		}
		jsonBytes, _ := json.Marshal(result)
		return &ExecuteResult{
			RawResult: json.RawMessage(jsonBytes),
			Message:   "命令执行成功",
			Duration:  time.Since(start).Milliseconds(),
		}, nil

	default:
		return nil, fmt.Errorf("不支持的操作: %s，支持: find/insertOne/updateMany/deleteMany/aggregate/count/runCommand", action)
	}
}

// toBsonDoc 将 interface{} 转换为 bson.M，nil 输入返回空 bson.M
func toBsonDoc(v interface{}) bson.M {
	if v == nil {
		return bson.M{}
	}
	if m, ok := v.(map[string]interface{}); ok {
		return bson.M(m)
	}
	return bson.M{}
}

// ===== Kafka Connector =====

// KafkaConnectionParams Kafka 连接认证参数
type KafkaConnectionParams struct {
	AuthMode            string `json:"authMode"`            // "none"/"sasl_plain"/"sasl_scram256"/"sasl_scram512"/"kerberos"
	KerberosServiceName string `json:"kerberosServiceName"` // 默认 "kafka"
	KerberosRealm       string `json:"kerberosRealm"`
	KerberosKeytab      string `json:"kerberosKeytab"`     // keytab 文件路径
	KerberosKeytabData  string `json:"kerberosKeytabData"` // keytab Base64 编码内容
	KerberosPrincipal   string `json:"kerberosPrincipal"`
	KerberosKrb5Conf    string `json:"kerberosKrb5Conf"`   // krb5.conf 路径
	KerberosKrb5Data    string `json:"kerberosKrb5Data"`   // krb5.conf 文本内容
	UseTLS              bool   `json:"useTLS"`
}

// XDGSCRAMClient implements sarama.SCRAMClient for SCRAM-SHA-256/512
type XDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	HashGeneratorFcn scram.HashGeneratorFcn
}

func (x *XDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

func (x *XDGSCRAMClient) Step(challenge string) (string, error) {
	return x.ClientConversation.Step(challenge)
}

func (x *XDGSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}

// SHA256 hash generator for SCRAM
func SHA256HashGenerator() hash.Hash { return sha256.New() }

// SHA512 hash generator for SCRAM
func SHA512HashGenerator() hash.Hash { return sha512.New() }

type KafkaConnector struct{}

func (c *KafkaConnector) parseConnectionParams(mw *assetbiz.Middleware) *KafkaConnectionParams {
	params := &KafkaConnectionParams{AuthMode: "none"}
	if mw.ConnectionParams != "" {
		_ = json.Unmarshal([]byte(mw.ConnectionParams), params)
	}
	return params
}

func (c *KafkaConnector) getBrokers(mw *assetbiz.Middleware) []string {
	hosts := strings.Split(mw.Host, ",")
	var brokers []string
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if !strings.Contains(h, ":") {
			h = fmt.Sprintf("%s:%d", h, mw.Port)
		}
		brokers = append(brokers, h)
	}
	if len(brokers) == 0 {
		brokers = []string{fmt.Sprintf("%s:%d", mw.Host, mw.Port)}
	}
	return brokers
}

// buildConfig constructs a sarama.Config based on middleware auth settings.
// Returns the config and a cleanup function for any temp files created.
func (c *KafkaConnector) buildConfig(mw *assetbiz.Middleware) (*sarama.Config, func(), error) {
	params := c.parseConnectionParams(mw)
	config := sarama.NewConfig()
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 30 * time.Second
	config.Net.WriteTimeout = 30 * time.Second
	config.Metadata.Timeout = 10 * time.Second

	var tempFiles []string
	cleanup := func() {
		for _, f := range tempFiles {
			os.Remove(f)
		}
	}

	if params.UseTLS {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{InsecureSkipVerify: true}
	}

	switch params.AuthMode {
	case "sasl_plain":
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		config.Net.SASL.User = mw.Username
		config.Net.SASL.Password = mw.Password

	case "sasl_scram256":
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		config.Net.SASL.User = mw.Username
		config.Net.SASL.Password = mw.Password
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA256HashGenerator}
		}

	case "sasl_scram512":
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		config.Net.SASL.User = mw.Username
		config.Net.SASL.Password = mw.Password
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA512HashGenerator}
		}

	case "kerberos":
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
		config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_KEYTAB_AUTH
		config.Net.SASL.GSSAPI.ServiceName = params.KerberosServiceName
		if config.Net.SASL.GSSAPI.ServiceName == "" {
			config.Net.SASL.GSSAPI.ServiceName = "kafka"
		}
		config.Net.SASL.GSSAPI.Realm = params.KerberosRealm
		principal := params.KerberosPrincipal
		if principal == "" {
			principal = mw.Username
		}
		config.Net.SASL.GSSAPI.Username = principal

		// Handle keytab: prefer Data field, fall back to path
		if params.KerberosKeytabData != "" {
			data, err := base64.StdEncoding.DecodeString(params.KerberosKeytabData)
			if err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("解码 keytab 数据失败: %w", err)
			}
			tmpFile, err := os.CreateTemp("", "kafka-keytab-*.keytab")
			if err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("创建临时 keytab 文件失败: %w", err)
			}
			if _, err := tmpFile.Write(data); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				cleanup()
				return nil, nil, fmt.Errorf("写入临时 keytab 文件失败: %w", err)
			}
			tmpFile.Close()
			tempFiles = append(tempFiles, tmpFile.Name())
			config.Net.SASL.GSSAPI.KeyTabPath = tmpFile.Name()
		} else if params.KerberosKeytab != "" {
			config.Net.SASL.GSSAPI.KeyTabPath = params.KerberosKeytab
		}

		// Handle krb5.conf: prefer Data field, fall back to path
		if params.KerberosKrb5Data != "" {
			tmpFile, err := os.CreateTemp("", "kafka-krb5-*.conf")
			if err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("创建临时 krb5.conf 文件失败: %w", err)
			}
			if _, err := tmpFile.WriteString(params.KerberosKrb5Data); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				cleanup()
				return nil, nil, fmt.Errorf("写入临时 krb5.conf 文件失败: %w", err)
			}
			tmpFile.Close()
			tempFiles = append(tempFiles, tmpFile.Name())
			config.Net.SASL.GSSAPI.KerberosConfigPath = tmpFile.Name()
		} else if params.KerberosKrb5Conf != "" {
			config.Net.SASL.GSSAPI.KerberosConfigPath = params.KerberosKrb5Conf
		}
	}

	return config, cleanup, nil
}

func (c *KafkaConnector) TestConnection(ctx context.Context, mw *assetbiz.Middleware) (*ConnectResult, error) {
	start := time.Now()
	brokers := c.getBrokers(mw)

	config, cleanup, err := c.buildConfig(mw)
	if err != nil {
		return &ConnectResult{Success: false, Message: fmt.Sprintf("配置错误: %v", err)}, nil
	}
	defer cleanup()

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return &ConnectResult{Success: false, Message: fmt.Sprintf("连接失败: %v", err)}, nil
	}
	defer client.Close()

	brokerList := client.Brokers()
	version := fmt.Sprintf("%d brokers", len(brokerList))

	return &ConnectResult{
		Success: true,
		Message: "连接成功",
		Version: version,
		Latency: time.Since(start).Milliseconds(),
	}, nil
}

func (c *KafkaConnector) Execute(ctx context.Context, mw *assetbiz.Middleware, req *ExecuteRequest) (*ExecuteResult, error) {
	return &ExecuteResult{
		Message: "Kafka 不支持通用 Execute，请使用专用 API",
	}, nil
}

// ===== Milvus Connector =====

type MilvusConnector struct{}

func (c *MilvusConnector) connect(_ context.Context, mw *assetbiz.Middleware) (milvusclient.Client, error) {
	addr := fmt.Sprintf("%s:%d", mw.Host, mw.Port)
	cfg := milvusclient.Config{
		Address: addr,
	}
	if mw.Username != "" {
		cfg.Username = mw.Username
		cfg.Password = mw.Password
	}
	if mw.DatabaseName != "" {
		cfg.DBName = mw.DatabaseName
	}
	// 使用独立的超时 context 建立 gRPC 连接，避免绑定到 HTTP 请求生命周期
	dialCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return milvusclient.NewClient(dialCtx, cfg)
}

func (c *MilvusConnector) TestConnection(ctx context.Context, mw *assetbiz.Middleware) (*ConnectResult, error) {
	start := time.Now()
	client, err := c.connect(ctx, mw)
	if err != nil {
		return &ConnectResult{Success: false, Message: fmt.Sprintf("连接失败: %v", err)}, nil
	}
	defer client.Close()

	version, err := client.GetVersion(ctx)
	if err != nil {
		version = "unknown"
	}

	return &ConnectResult{
		Success: true,
		Message: "连接成功",
		Version: version,
		Latency: time.Since(start).Milliseconds(),
	}, nil
}

func (c *MilvusConnector) Execute(ctx context.Context, mw *assetbiz.Middleware, req *ExecuteRequest) (*ExecuteResult, error) {
	return &ExecuteResult{
		Message: "Milvus 不支持通用 Execute，请使用专用 API",
	}, nil
}
