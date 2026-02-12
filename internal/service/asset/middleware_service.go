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
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
	milvusclient "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/redis/go-redis/v9"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	auditmodel "github.com/ydcloud-dy/opshub/internal/biz/audit"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	rbacService "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

// MiddlewareService 中间件HTTP服务
type MiddlewareService struct {
	middlewareUseCase   *assetbiz.MiddlewareUseCase
	mwPermissionUseCase *rbac.MiddlewarePermissionUseCase
	db                  *gorm.DB
}

// NewMiddlewareService 创建中间件服务
func NewMiddlewareService(
	middlewareUseCase *assetbiz.MiddlewareUseCase,
	mwPermissionUseCase *rbac.MiddlewarePermissionUseCase,
	db *gorm.DB,
) *MiddlewareService {
	return &MiddlewareService{
		middlewareUseCase:   middlewareUseCase,
		mwPermissionUseCase: mwPermissionUseCase,
		db:                  db,
	}
}

// CreateMiddleware 创建中间件
func (s *MiddlewareService) CreateMiddleware(c *gin.Context) {
	var req assetbiz.MiddlewareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	mw, err := s.middlewareUseCase.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, mw)
}

// UpdateMiddleware 更新中间件
func (s *MiddlewareService) UpdateMiddleware(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return
	}

	var req assetbiz.MiddlewareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	req.ID = uint(id)

	if err := s.middlewareUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteMiddleware 删除中间件
func (s *MiddlewareService) DeleteMiddleware(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return
	}

	if err := s.middlewareUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetMiddleware 获取中间件详情
func (s *MiddlewareService) GetMiddleware(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return
	}

	mw, err := s.middlewareUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, mw)
}

// ListMiddlewares 中间件列表
func (s *MiddlewareService) ListMiddlewares(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	mwType := c.Query("type")

	var groupID *uint
	if gidStr := c.Query("groupId"); gidStr != "" {
		gid, err := strconv.ParseUint(gidStr, 10, 32)
		if err == nil {
			gidVal := uint(gid)
			groupID = &gidVal
		}
	}

	var status *int
	if statusStr := c.Query("status"); statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err == nil {
			status = &s
		}
	}

	// 获取用户可访问的中间件ID列表
	userID := rbacService.GetUserID(c)
	var accessibleIDs []uint
	if userID > 0 && s.mwPermissionUseCase != nil {
		ids, err := s.mwPermissionUseCase.GetUserAccessibleMiddlewareIDs(c.Request.Context(), userID)
		if err == nil {
			accessibleIDs = ids
		}
	}

	list, total, err := s.middlewareUseCase.List(c.Request.Context(), page, pageSize, keyword, mwType, groupID, status, accessibleIDs)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"total": total,
		"list":  list,
	})
}

// BatchDeleteMiddlewares 批量删除中间件
func (s *MiddlewareService) BatchDeleteMiddlewares(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.middlewareUseCase.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "批量删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "批量删除成功", nil)
}

// TestMiddlewareConnection 测试中间件连接
func (s *MiddlewareService) TestMiddlewareConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return
	}

	mw, err := s.middlewareUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取中间件信息失败: "+err.Error())
		return
	}

	connector, err := GetConnector(mw.Type)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := connector.TestConnection(c.Request.Context(), mw)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "测试连接失败: "+err.Error())
		return
	}

	// 更新状态
	status := 0
	if result.Success {
		status = 1
	}
	_ = s.middlewareUseCase.UpdateStatus(c.Request.Context(), uint(id), status, result.Version)

	response.Success(c, result)
}

// ExecuteMiddleware 执行中间件数据操作
func (s *MiddlewareService) ExecuteMiddleware(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return
	}

	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	mw, err := s.middlewareUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取中间件信息失败: "+err.Error())
		return
	}

	// 细粒度权限校验：根据命令类型检查 MWPermQuery 或 MWPermModify
	cmdType := classifyCommand(mw.Type, req.Command)
	userID := rbacService.GetUserID(c)
	requiredPerm := uint(rbac.MWPermQuery)
	if cmdType != "query" {
		requiredPerm = uint(rbac.MWPermModify)
	}
	hasPermission, _ := s.mwPermissionUseCase.CheckMiddlewarePermission(c.Request.Context(), userID, uint(id), requiredPerm)
	if !hasPermission {
		// 回退检查旧的 MWPermExecute 权限（兼容已有配置）
		hasPermission, _ = s.mwPermissionUseCase.CheckMiddlewarePermission(c.Request.Context(), userID, uint(id), uint(rbac.MWPermExecute))
	}
	if !hasPermission {
		response.ErrorCode(c, http.StatusForbidden, "权限不足")
		return
	}

	// 如果请求中指定了数据库，覆盖默认数据库
	if req.Database != "" {
		mw.DatabaseName = req.Database
	}

	connector, err := GetConnector(mw.Type)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	result, execErr := connector.Execute(c.Request.Context(), mw, &req)

	// 异步写入审计日志
	auditLog := &auditmodel.SysMiddlewareAuditLog{
		UserID:         userID,
		Username:       rbacService.GetUsername(c),
		MiddlewareID:   uint(id),
		MiddlewareName: mw.Name,
		MiddlewareType: mw.Type,
		Database:       req.Database,
		Command:        req.Command,
		CommandType:    cmdType,
		Status:         "success",
		IP:             c.ClientIP(),
	}
	if result != nil {
		auditLog.Duration = result.Duration
		auditLog.AffectedRows = result.AffectedRows
	}
	if execErr != nil {
		auditLog.Status = "failed"
		auditLog.ErrorMsg = execErr.Error()
	}
	go s.db.Create(auditLog)

	if execErr != nil {
		response.ErrorCode(c, http.StatusInternalServerError, execErr.Error())
		return
	}

	response.Success(c, result)
}

// writeAuditLog 写入中间件审计日志的通用辅助方法
func (s *MiddlewareService) writeAuditLog(c *gin.Context, mw *assetbiz.Middleware, database, command, commandType, status string, execErr error, duration, affectedRows int64) {
	auditLog := &auditmodel.SysMiddlewareAuditLog{
		UserID:         rbacService.GetUserID(c),
		Username:       rbacService.GetUsername(c),
		MiddlewareID:   mw.ID,
		MiddlewareName: mw.Name,
		MiddlewareType: mw.Type,
		Database:       database,
		Command:        command,
		CommandType:    commandType,
		Status:         status,
		Duration:       duration,
		AffectedRows:   affectedRows,
		IP:             c.ClientIP(),
	}
	if execErr != nil {
		auditLog.Status = "failed"
		auditLog.ErrorMsg = execErr.Error()
	}
	go s.db.Create(auditLog)
}

// classifyCommand 根据中间件类型和命令内容推断命令类型
func classifyCommand(mwType, command string) string {
	cmd := strings.TrimSpace(command)
	upper := strings.ToUpper(cmd)

	switch mwType {
	case "mysql", "clickhouse":
		return classifySQLCommand(upper)
	case "redis":
		return classifyRedisCommand(upper)
	case "mongodb":
		return classifyMongoCommand(cmd)
	case "kafka":
		return classifyKafkaCommand(upper)
	case "milvus":
		return classifyMilvusCommand(cmd)
	default:
		return "other"
	}
}

var sqlSelectRe = regexp.MustCompile(`^(SELECT|SHOW|DESC|DESCRIBE|EXPLAIN)\b`)
var sqlInsertRe = regexp.MustCompile(`^INSERT\b`)
var sqlUpdateRe = regexp.MustCompile(`^UPDATE\b`)
var sqlDeleteRe = regexp.MustCompile(`^DELETE\b`)
var sqlDDLRe = regexp.MustCompile(`^(CREATE|ALTER|DROP|TRUNCATE|RENAME)\b`)

func classifySQLCommand(upper string) string {
	switch {
	case sqlSelectRe.MatchString(upper):
		return "query"
	case sqlInsertRe.MatchString(upper):
		return "insert"
	case sqlUpdateRe.MatchString(upper):
		return "update"
	case sqlDeleteRe.MatchString(upper):
		return "delete"
	case sqlDDLRe.MatchString(upper):
		return "ddl"
	default:
		return "other"
	}
}

func classifyRedisCommand(upper string) string {
	parts := strings.Fields(upper)
	if len(parts) == 0 {
		return "other"
	}
	switch parts[0] {
	case "GET", "MGET", "HGET", "HGETALL", "HMGET", "KEYS", "SCAN", "HSCAN", "SSCAN", "ZSCAN",
		"LRANGE", "LINDEX", "LLEN", "SCARD", "SMEMBERS", "SISMEMBER", "ZRANGE", "ZCARD",
		"ZRANGEBYSCORE", "ZSCORE", "TTL", "PTTL", "TYPE", "EXISTS", "DBSIZE", "INFO", "STRLEN":
		return "query"
	case "SET", "MSET", "HSET", "HMSET", "LPUSH", "RPUSH", "SADD", "ZADD", "SETNX", "SETEX",
		"APPEND", "INCR", "INCRBY", "DECR", "DECRBY":
		return "insert"
	case "DEL", "HDEL", "LREM", "SREM", "ZREM", "UNLINK":
		return "delete"
	default:
		return "other"
	}
}

func classifyMongoCommand(cmd string) string {
	lower := strings.ToLower(cmd)
	switch {
	case strings.Contains(lower, "find") || strings.Contains(lower, "count") || strings.Contains(lower, "aggregate"):
		return "query"
	case strings.Contains(lower, "insertone") || strings.Contains(lower, "insertmany"):
		return "insert"
	case strings.Contains(lower, "updateone") || strings.Contains(lower, "updatemany"):
		return "update"
	case strings.Contains(lower, "deleteone") || strings.Contains(lower, "deletemany"):
		return "delete"
	default:
		return "other"
	}
}

func classifyKafkaCommand(upper string) string {
	switch {
	case strings.Contains(upper, "CONSUME") || strings.Contains(upper, "POLL") || strings.Contains(upper, "LIST"):
		return "query"
	case strings.Contains(upper, "PRODUCE") || strings.Contains(upper, "CREATE"):
		return "insert"
	case strings.Contains(upper, "DELETE"):
		return "delete"
	default:
		return "other"
	}
}

func classifyMilvusCommand(cmd string) string {
	lower := strings.ToLower(cmd)
	switch {
	case strings.Contains(lower, "search") || strings.Contains(lower, "query") || strings.Contains(lower, "describe") || strings.Contains(lower, "list"):
		return "query"
	case strings.Contains(lower, "insert"):
		return "insert"
	case strings.Contains(lower, "delete"):
		return "delete"
	default:
		return "other"
	}
}

// getMiddlewareMySQL 获取中间件并建立MySQL连接（内部复用方法）
func (s *MiddlewareService) getMiddlewareMySQL(c *gin.Context) (*assetbiz.Middleware, *sql.DB, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return nil, nil, false
	}

	mw, err := s.middlewareUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取中间件信息失败: "+err.Error())
		return nil, nil, false
	}

	if mw.Type != assetbiz.MiddlewareTypeMySQL {
		response.ErrorCode(c, http.StatusBadRequest, "该接口仅支持 MySQL 类型中间件")
		return nil, nil, false
	}

	connector := &MySQLConnector{}
	db, err := sql.Open("mysql", connector.buildDSN(mw))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("连接失败: %v", err))
		return nil, nil, false
	}

	if err := db.PingContext(c.Request.Context()); err != nil {
		db.Close()
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("连接失败: %v", err))
		return nil, nil, false
	}

	return mw, db, true
}

// ListDatabases 获取数据库列表
func (s *MiddlewareService) ListDatabases(c *gin.Context) {
	_, db, ok := s.getMiddlewareMySQL(c)
	if !ok {
		return
	}
	defer db.Close()

	rows, err := db.QueryContext(c.Request.Context(), "SHOW DATABASES")
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			databases = append(databases, name)
		}
	}

	response.Success(c, databases)
}

// ListTables 获取指定数据库的表列表
func (s *MiddlewareService) ListTables(c *gin.Context) {
	_, db, ok := s.getMiddlewareMySQL(c)
	if !ok {
		return
	}
	defer db.Close()

	database := c.Query("database")
	if database == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 database 参数")
		return
	}

	ctx := c.Request.Context()
	if _, err := db.ExecContext(ctx, "USE `"+database+"`"); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("切换数据库失败: %v", err))
		return
	}

	rows, err := db.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}

	response.Success(c, tables)
}

// ListColumns 获取表的列信息
func (s *MiddlewareService) ListColumns(c *gin.Context) {
	_, db, ok := s.getMiddlewareMySQL(c)
	if !ok {
		return
	}
	defer db.Close()

	database := c.Query("database")
	table := c.Query("table")
	if database == "" || table == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 database 或 table 参数")
		return
	}

	ctx := c.Request.Context()
	query := fmt.Sprintf("SHOW COLUMNS FROM `%s`.`%s`", database, table)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}
	defer rows.Close()

	type ColumnInfo struct {
		Field   string      `json:"field"`
		Type    string      `json:"type"`
		Null    string      `json:"null"`
		Key     string      `json:"key"`
		Default interface{} `json:"default"`
		Extra   string      `json:"extra"`
	}

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var defaultVal sql.NullString
		if err := rows.Scan(&col.Field, &col.Type, &col.Null, &col.Key, &defaultVal, &col.Extra); err == nil {
			if defaultVal.Valid {
				col.Default = defaultVal.String
			}
			columns = append(columns, col)
		}
	}

	response.Success(c, columns)
}

// ===== Redis 专用 API =====

// getMiddlewareRedis 获取中间件并建立Redis连接（内部复用方法）
func (s *MiddlewareService) getMiddlewareRedis(c *gin.Context) (*assetbiz.Middleware, *redis.Client, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return nil, nil, false
	}

	mw, err := s.middlewareUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取中间件信息失败: "+err.Error())
		return nil, nil, false
	}

	if mw.Type != assetbiz.MiddlewareTypeRedis {
		response.ErrorCode(c, http.StatusBadRequest, "该接口仅支持 Redis 类型中间件")
		return nil, nil, false
	}

	connector := &RedisConnector{}
	opts := connector.buildOptions(mw)

	// 如果请求中指定了 db，覆盖默认值
	if dbStr := c.Query("db"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			opts.DB = db
		}
	}

	rdb := redis.NewClient(opts)
	if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
		rdb.Close()
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("连接失败: %v", err))
		return nil, nil, false
	}

	return mw, rdb, true
}

// GetRedisInfo 获取 Redis 服务器信息
func (s *MiddlewareService) GetRedisInfo(c *gin.Context) {
	_, rdb, ok := s.getMiddlewareRedis(c)
	if !ok {
		return
	}
	defer rdb.Close()

	info, err := rdb.Info(c.Request.Context()).Result()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取信息失败: %v", err))
		return
	}

	// 解析 INFO 为结构化数据
	sections := make(map[string]map[string]string)
	var currentSection string
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			currentSection = strings.TrimPrefix(line, "# ")
			sections[currentSection] = make(map[string]string)
			continue
		}
		if currentSection != "" {
			if idx := strings.Index(line, ":"); idx > 0 {
				sections[currentSection][line[:idx]] = line[idx+1:]
			}
		}
	}

	response.Success(c, sections)
}

// GetRedisDatabases 获取 Redis 数据库列表及 key 数量
func (s *MiddlewareService) GetRedisDatabases(c *gin.Context) {
	_, rdb, ok := s.getMiddlewareRedis(c)
	if !ok {
		return
	}
	defer rdb.Close()

	// 获取 keyspace 信息
	info, err := rdb.Info(c.Request.Context(), "keyspace").Result()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取信息失败: %v", err))
		return
	}

	// 获取数据库数量配置（部分云 Redis 禁用了 CONFIG 命令，失败时使用默认值）
	dbCount := 16 // 默认16个数据库
	configResult, err := rdb.ConfigGet(c.Request.Context(), "databases").Result()
	if err == nil && len(configResult) >= 2 {
		if count, err := strconv.Atoi(configResult["databases"]); err == nil {
			dbCount = count
		}
	}

	// 解析 keyspace 信息
	keyspaceMap := make(map[int]int64)
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "db") {
			continue
		}
		// 格式: db0:keys=123,expires=0,avg_ttl=0
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		dbNum, err := strconv.Atoi(strings.TrimPrefix(parts[0], "db"))
		if err != nil {
			continue
		}
		for _, field := range strings.Split(parts[1], ",") {
			kv := strings.SplitN(field, "=", 2)
			if len(kv) == 2 && kv[0] == "keys" {
				if keys, err := strconv.ParseInt(kv[1], 10, 64); err == nil {
					keyspaceMap[dbNum] = keys
				}
			}
		}
	}

	type DBInfo struct {
		DB   int   `json:"db"`
		Keys int64 `json:"keys"`
	}

	var databases []DBInfo
	for i := 0; i < dbCount; i++ {
		databases = append(databases, DBInfo{
			DB:   i,
			Keys: keyspaceMap[i],
		})
	}

	response.Success(c, databases)
}

// PLACEHOLDER_REDIS_HANDLERS_CONTINUE

// ScanRedisKeys 扫描 Redis 键
func (s *MiddlewareService) ScanRedisKeys(c *gin.Context) {
	_, rdb, ok := s.getMiddlewareRedis(c)
	if !ok {
		return
	}
	defer rdb.Close()

	cursor, _ := strconv.ParseUint(c.DefaultQuery("cursor", "0"), 10, 64)
	count, _ := strconv.ParseInt(c.DefaultQuery("count", "50"), 10, 64)
	pattern := c.DefaultQuery("pattern", "*")

	keys, nextCursor, err := rdb.Scan(c.Request.Context(), cursor, pattern, count).Result()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("扫描失败: %v", err))
		return
	}

	// 获取每个 key 的类型和 TTL
	type KeyInfo struct {
		Key  string `json:"key"`
		Type string `json:"type"`
		TTL  int64  `json:"ttl"` // -1=永不过期, -2=不存在, 其他=秒数
	}

	ctx := c.Request.Context()
	pipe := rdb.Pipeline()
	typeCmds := make([]*redis.StatusCmd, len(keys))
	ttlCmds := make([]*redis.DurationCmd, len(keys))
	for i, key := range keys {
		typeCmds[i] = pipe.Type(ctx, key)
		ttlCmds[i] = pipe.TTL(ctx, key)
	}
	pipe.Exec(ctx)

	var keyInfos []KeyInfo
	for i, key := range keys {
		ttlVal := int64(-1)
		if d := ttlCmds[i].Val(); d >= 0 {
			ttlVal = int64(d.Seconds())
		} else if d == -2 {
			ttlVal = -2
		}
		keyInfos = append(keyInfos, KeyInfo{
			Key:  key,
			Type: typeCmds[i].Val(),
			TTL:  ttlVal,
		})
	}

	// 按键名排序
	sort.Slice(keyInfos, func(i, j int) bool {
		return keyInfos[i].Key < keyInfos[j].Key
	})

	response.Success(c, gin.H{
		"keys":   keyInfos,
		"cursor": nextCursor,
	})
}

// GetRedisKeyDetail 获取键的类型、TTL、值
func (s *MiddlewareService) GetRedisKeyDetail(c *gin.Context) {
	_, rdb, ok := s.getMiddlewareRedis(c)
	if !ok {
		return
	}
	defer rdb.Close()

	key := c.Query("key")
	if key == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 key 参数")
		return
	}

	ctx := c.Request.Context()

	// 获取类型
	keyType, err := rdb.Type(ctx, key).Result()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取类型失败: %v", err))
		return
	}
	if keyType == "none" {
		response.ErrorCode(c, http.StatusNotFound, "键不存在")
		return
	}

	// 获取 TTL
	ttl, _ := rdb.TTL(ctx, key).Result()
	ttlSeconds := int64(-1)
	if ttl >= 0 {
		ttlSeconds = int64(ttl.Seconds())
	}

	// 获取值
	var value interface{}
	switch keyType {
	case "string":
		value, err = rdb.Get(ctx, key).Result()
	case "hash":
		value, err = rdb.HGetAll(ctx, key).Result()
	case "list":
		value, err = rdb.LRange(ctx, key, 0, -1).Result()
	case "set":
		value, err = rdb.SMembers(ctx, key).Result()
	case "zset":
		var zSlice []redis.Z
		zSlice, err = rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
		if err == nil {
			type ZMember struct {
				Member string  `json:"member"`
				Score  float64 `json:"score"`
			}
			var members []ZMember
			for _, z := range zSlice {
				members = append(members, ZMember{
					Member: fmt.Sprintf("%v", z.Member),
					Score:  z.Score,
				})
			}
			value = members
		}
	default:
		value = fmt.Sprintf("不支持的类型: %s", keyType)
	}

	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取值失败: %v", err))
		return
	}

	// 获取内存占用（可选，忽略错误）
	size, _ := rdb.MemoryUsage(ctx, key).Result()

	response.Success(c, gin.H{
		"key":   key,
		"type":  keyType,
		"ttl":   ttlSeconds,
		"value": value,
		"size":  size,
	})
}

// PLACEHOLDER_REDIS_HANDLERS_PART2

// SetRedisKey 创建/更新键值
func (s *MiddlewareService) SetRedisKey(c *gin.Context) {
	mw, rdb, ok := s.getMiddlewareRedis(c)
	if !ok {
		return
	}
	defer rdb.Close()

	var req struct {
		Key   string      `json:"key" binding:"required"`
		Type  string      `json:"type" binding:"required"`
		Value interface{} `json:"value" binding:"required"`
		TTL   int64       `json:"ttl"` // -1=永不过期, >0=秒数
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	var expiration time.Duration
	if req.TTL > 0 {
		expiration = time.Duration(req.TTL) * time.Second
	}

	cmdDesc := fmt.Sprintf("SET %s ...", req.Key)
	var err error
	switch req.Type {
	case "string":
		val := fmt.Sprintf("%v", req.Value)
		err = rdb.Set(ctx, req.Key, val, expiration).Err()
	case "hash":
		// value 应为 map[string]interface{}
		if m, ok := req.Value.(map[string]interface{}); ok {
			pipe := rdb.Pipeline()
			pipe.Del(ctx, req.Key)
			if len(m) > 0 {
				fields := make([]interface{}, 0, len(m)*2)
				for k, v := range m {
					fields = append(fields, k, fmt.Sprintf("%v", v))
				}
				pipe.HSet(ctx, req.Key, fields...)
			}
			if expiration > 0 {
				pipe.Expire(ctx, req.Key, expiration)
			}
			_, err = pipe.Exec(ctx)
		} else {
			response.ErrorCode(c, http.StatusBadRequest, "hash 类型的 value 应为对象")
			return
		}
	case "list":
		if arr, ok := req.Value.([]interface{}); ok {
			pipe := rdb.Pipeline()
			pipe.Del(ctx, req.Key)
			if len(arr) > 0 {
				pipe.RPush(ctx, req.Key, arr...)
			}
			if expiration > 0 {
				pipe.Expire(ctx, req.Key, expiration)
			}
			_, err = pipe.Exec(ctx)
		} else {
			response.ErrorCode(c, http.StatusBadRequest, "list 类型的 value 应为数组")
			return
		}
	case "set":
		if arr, ok := req.Value.([]interface{}); ok {
			pipe := rdb.Pipeline()
			pipe.Del(ctx, req.Key)
			if len(arr) > 0 {
				pipe.SAdd(ctx, req.Key, arr...)
			}
			if expiration > 0 {
				pipe.Expire(ctx, req.Key, expiration)
			}
			_, err = pipe.Exec(ctx)
		} else {
			response.ErrorCode(c, http.StatusBadRequest, "set 类型的 value 应为数组")
			return
		}
	case "zset":
		if arr, ok := req.Value.([]interface{}); ok {
			pipe := rdb.Pipeline()
			pipe.Del(ctx, req.Key)
			var members []redis.Z
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					score := 0.0
					if s, ok := m["score"].(float64); ok {
						score = s
					}
					members = append(members, redis.Z{
						Score:  score,
						Member: fmt.Sprintf("%v", m["member"]),
					})
				}
			}
			if len(members) > 0 {
				pipe.ZAdd(ctx, req.Key, members...)
			}
			if expiration > 0 {
				pipe.Expire(ctx, req.Key, expiration)
			}
			_, err = pipe.Exec(ctx)
		} else {
			response.ErrorCode(c, http.StatusBadRequest, "zset 类型的 value 应为数组")
			return
		}
	default:
		response.ErrorCode(c, http.StatusBadRequest, "不支持的类型: "+req.Type)
		return
	}

	if err != nil {
		s.writeAuditLog(c, mw, "", cmdDesc, "insert", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("设置失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", cmdDesc, "insert", "success", nil, 0, 1)
	response.SuccessWithMessage(c, "设置成功", nil)
}

// PLACEHOLDER_REDIS_HANDLERS_PART3

// RedisKeyAction 对键执行部分操作（HSET/HDEL/SADD/SREM/LPUSH/RPUSH/LPOP/RPOP/ZADD/ZREM）
func (s *MiddlewareService) RedisKeyAction(c *gin.Context) {
	mw, rdb, ok := s.getMiddlewareRedis(c)
	if !ok {
		return
	}
	defer rdb.Close()

	var req struct {
		Key    string      `json:"key" binding:"required"`
		Action string      `json:"action" binding:"required"`
		Field  string      `json:"field"`
		Value  interface{} `json:"value"`
		Score  float64     `json:"score"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	var result interface{}
	var err error

	cmdDesc := fmt.Sprintf("RedisKeyAction %s %s", req.Action, req.Key)

	switch strings.ToUpper(req.Action) {
	case "HSET":
		err = rdb.HSet(ctx, req.Key, req.Field, fmt.Sprintf("%v", req.Value)).Err()
		result = "OK"
	case "HDEL":
		var n int64
		n, err = rdb.HDel(ctx, req.Key, req.Field).Result()
		result = n
	case "SADD":
		var n int64
		n, err = rdb.SAdd(ctx, req.Key, fmt.Sprintf("%v", req.Value)).Result()
		result = n
	case "SREM":
		var n int64
		n, err = rdb.SRem(ctx, req.Key, fmt.Sprintf("%v", req.Value)).Result()
		result = n
	case "LPUSH":
		var n int64
		n, err = rdb.LPush(ctx, req.Key, fmt.Sprintf("%v", req.Value)).Result()
		result = n
	case "RPUSH":
		var n int64
		n, err = rdb.RPush(ctx, req.Key, fmt.Sprintf("%v", req.Value)).Result()
		result = n
	case "LPOP":
		result, err = rdb.LPop(ctx, req.Key).Result()
	case "RPOP":
		result, err = rdb.RPop(ctx, req.Key).Result()
	case "LSET":
		idx, _ := strconv.ParseInt(req.Field, 10, 64)
		err = rdb.LSet(ctx, req.Key, idx, fmt.Sprintf("%v", req.Value)).Err()
		result = "OK"
	case "ZADD":
		var n int64
		n, err = rdb.ZAdd(ctx, req.Key, redis.Z{
			Score:  req.Score,
			Member: fmt.Sprintf("%v", req.Value),
		}).Result()
		result = n
	case "ZREM":
		var n int64
		n, err = rdb.ZRem(ctx, req.Key, fmt.Sprintf("%v", req.Value)).Result()
		result = n
	default:
		response.ErrorCode(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}

	if err != nil {
		s.writeAuditLog(c, mw, "", cmdDesc, "other", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("操作失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", cmdDesc, "other", "success", nil, 0, 1)
	response.Success(c, result)
}

// DeleteRedisKeys 删除键
func (s *MiddlewareService) DeleteRedisKeys(c *gin.Context) {
	mw, rdb, ok := s.getMiddlewareRedis(c)
	if !ok {
		return
	}
	defer rdb.Close()

	var req struct {
		Keys []string `json:"keys" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	cmdDesc := fmt.Sprintf("DEL %v", req.Keys)
	n, err := rdb.Del(c.Request.Context(), req.Keys...).Result()
	if err != nil {
		s.writeAuditLog(c, mw, "", cmdDesc, "delete", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("删除失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", cmdDesc, "delete", "success", nil, 0, n)
	response.Success(c, gin.H{"deleted": n})
}

// SetRedisKeyTTL 设置键的 TTL
func (s *MiddlewareService) SetRedisKeyTTL(c *gin.Context) {
	mw, rdb, ok := s.getMiddlewareRedis(c)
	if !ok {
		return
	}
	defer rdb.Close()

	var req struct {
		Key string `json:"key" binding:"required"`
		TTL int64  `json:"ttl" binding:"required"` // -1=移除过期, >0=秒数
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	cmdDesc := fmt.Sprintf("EXPIRE %s %d", req.Key, req.TTL)
	ctx := c.Request.Context()
	var err error
	if req.TTL < 0 {
		err = rdb.Persist(ctx, req.Key).Err()
	} else {
		err = rdb.Expire(ctx, req.Key, time.Duration(req.TTL)*time.Second).Err()
	}

	if err != nil {
		s.writeAuditLog(c, mw, "", cmdDesc, "other", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("设置TTL失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", cmdDesc, "other", "success", nil, 0, 1)
	response.SuccessWithMessage(c, "设置成功", nil)
}

// RenameRedisKey 重命名键
func (s *MiddlewareService) RenameRedisKey(c *gin.Context) {
	mw, rdb, ok := s.getMiddlewareRedis(c)
	if !ok {
		return
	}
	defer rdb.Close()

	var req struct {
		OldKey string `json:"oldKey" binding:"required"`
		NewKey string `json:"newKey" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	cmdDesc := fmt.Sprintf("RENAME %s %s", req.OldKey, req.NewKey)
	if err := rdb.Rename(c.Request.Context(), req.OldKey, req.NewKey).Err(); err != nil {
		s.writeAuditLog(c, mw, "", cmdDesc, "other", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("重命名失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", cmdDesc, "other", "success", nil, 0, 1)
	response.SuccessWithMessage(c, "重命名成功", nil)
}

// ===== ClickHouse 专用 API =====

// getMiddlewareClickHouse 获取中间件并建立ClickHouse连接（内部复用方法）
func (s *MiddlewareService) getMiddlewareClickHouse(c *gin.Context) (*assetbiz.Middleware, *sql.DB, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return nil, nil, false
	}

	mw, err := s.middlewareUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取中间件信息失败: "+err.Error())
		return nil, nil, false
	}

	if mw.Type != assetbiz.MiddlewareTypeClickHouse {
		response.ErrorCode(c, http.StatusBadRequest, "该接口仅支持 ClickHouse 类型中间件")
		return nil, nil, false
	}

	connector := &ClickHouseConnector{}
	db, err := sql.Open("clickhouse", connector.buildDSN(mw))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("连接失败: %v", err))
		return nil, nil, false
	}

	if err := db.PingContext(c.Request.Context()); err != nil {
		db.Close()
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("连接失败: %v", err))
		return nil, nil, false
	}

	return mw, db, true
}

// ListClickHouseDatabases 获取 ClickHouse 数据库列表
func (s *MiddlewareService) ListClickHouseDatabases(c *gin.Context) {
	_, db, ok := s.getMiddlewareClickHouse(c)
	if !ok {
		return
	}
	defer db.Close()

	rows, err := db.QueryContext(c.Request.Context(), "SHOW DATABASES")
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			databases = append(databases, name)
		}
	}

	response.Success(c, databases)
}

// ListClickHouseTables 获取 ClickHouse 指定数据库的表列表
func (s *MiddlewareService) ListClickHouseTables(c *gin.Context) {
	_, db, ok := s.getMiddlewareClickHouse(c)
	if !ok {
		return
	}
	defer db.Close()

	database := c.Query("database")
	if database == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 database 参数")
		return
	}

	query := fmt.Sprintf("SHOW TABLES FROM `%s`", database)
	rows, err := db.QueryContext(c.Request.Context(), query)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}

	response.Success(c, tables)
}

// ListClickHouseColumns 获取 ClickHouse 表的列信息
func (s *MiddlewareService) ListClickHouseColumns(c *gin.Context) {
	_, db, ok := s.getMiddlewareClickHouse(c)
	if !ok {
		return
	}
	defer db.Close()

	database := c.Query("database")
	table := c.Query("table")
	if database == "" || table == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 database 或 table 参数")
		return
	}

	query := fmt.Sprintf("DESCRIBE TABLE `%s`.`%s`", database, table)
	rows, err := db.QueryContext(c.Request.Context(), query)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}
	defer rows.Close()

	type ColumnInfo struct {
		Name              string `json:"name"`
		Type              string `json:"type"`
		DefaultType       string `json:"defaultType"`
		DefaultExpression string `json:"defaultExpression"`
		Comment           string `json:"comment"`
	}

	cols, _ := rows.Columns()
	var columns []ColumnInfo
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		col := ColumnInfo{}
		if len(values) > 0 {
			col.Name = fmt.Sprintf("%v", values[0])
		}
		if len(values) > 1 {
			col.Type = fmt.Sprintf("%v", values[1])
		}
		if len(values) > 2 {
			col.DefaultType = fmt.Sprintf("%v", values[2])
		}
		if len(values) > 3 {
			col.DefaultExpression = fmt.Sprintf("%v", values[3])
		}
		if len(values) > 4 {
			col.Comment = fmt.Sprintf("%v", values[4])
		}
		columns = append(columns, col)
	}

	response.Success(c, columns)
}

// ===== MongoDB 专用 API =====

// getMiddlewareMongoDB 获取中间件并建立MongoDB连接（内部复用方法）
func (s *MiddlewareService) getMiddlewareMongoDB(c *gin.Context) (*assetbiz.Middleware, *mongo.Client, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return nil, nil, false
	}

	mw, err := s.middlewareUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取中间件信息失败: "+err.Error())
		return nil, nil, false
	}

	if mw.Type != assetbiz.MiddlewareTypeMongoDB {
		response.ErrorCode(c, http.StatusBadRequest, "该接口仅支持 MongoDB 类型中间件")
		return nil, nil, false
	}

	connector := &MongoDBConnector{}
	ctx := c.Request.Context()
	client, err := connector.connect(ctx, mw)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("连接失败: %v", err))
		return nil, nil, false
	}

	return mw, client, true
}

// ListMongoDatabases 获取 MongoDB 数据库列表
func (s *MiddlewareService) ListMongoDatabases(c *gin.Context) {
	_, client, ok := s.getMiddlewareMongoDB(c)
	if !ok {
		return
	}
	defer client.Disconnect(c.Request.Context())

	databases, err := client.ListDatabaseNames(c.Request.Context(), bson.M{})
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}

	response.Success(c, databases)
}

// ListMongoCollections 获取 MongoDB 指定数据库的集合列表
func (s *MiddlewareService) ListMongoCollections(c *gin.Context) {
	_, client, ok := s.getMiddlewareMongoDB(c)
	if !ok {
		return
	}
	defer client.Disconnect(c.Request.Context())

	database := c.Query("database")
	if database == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 database 参数")
		return
	}

	collections, err := client.Database(database).ListCollectionNames(c.Request.Context(), bson.M{})
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}

	response.Success(c, collections)
}

// QueryMongoDocuments 查询 MongoDB 文档
func (s *MiddlewareService) QueryMongoDocuments(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMongoDB(c)
	if !ok {
		return
	}
	defer client.Disconnect(c.Request.Context())

	var req struct {
		Database   string                 `json:"database" binding:"required"`
		Collection string                 `json:"collection" binding:"required"`
		Filter     map[string]interface{} `json:"filter"`
		Sort       map[string]interface{} `json:"sort"`
		Limit      int64                  `json:"limit"`
		Skip       int64                  `json:"skip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if req.Limit <= 0 {
		req.Limit = 50
	}

	filter := bson.M(req.Filter)
	if filter == nil {
		filter = bson.M{}
	}

	opts := options.Find().SetLimit(req.Limit).SetSkip(req.Skip)
	if req.Sort != nil {
		opts.SetSort(bson.M(req.Sort))
	}

	ctx := c.Request.Context()
	coll := client.Database(req.Database).Collection(req.Collection)

	// 获取总数
	total, _ := coll.CountDocuments(ctx, filter)

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		cmdDesc := fmt.Sprintf("db.%s.find(%v)", req.Collection, req.Filter)
		s.writeAuditLog(c, mw, req.Database, cmdDesc, "query", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("读取结果失败: %v", err))
		return
	}

	cmdDesc := fmt.Sprintf("db.%s.find(%v)", req.Collection, req.Filter)
	s.writeAuditLog(c, mw, req.Database, cmdDesc, "query", "success", nil, 0, int64(len(results)))

	response.Success(c, gin.H{
		"total": total,
		"list":  results,
	})
}

// MongoInsertDocument 插入 MongoDB 文档
func (s *MiddlewareService) MongoInsertDocument(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMongoDB(c)
	if !ok {
		return
	}
	defer client.Disconnect(c.Request.Context())

	var req struct {
		Database   string                 `json:"database" binding:"required"`
		Collection string                 `json:"collection" binding:"required"`
		Document   map[string]interface{} `json:"document" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	cmdDesc := fmt.Sprintf("db.%s.insertOne(%v)", req.Collection, req.Document)
	result, err := client.Database(req.Database).Collection(req.Collection).InsertOne(c.Request.Context(), bson.M(req.Document))
	if err != nil {
		s.writeAuditLog(c, mw, req.Database, cmdDesc, "insert", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("插入失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, req.Database, cmdDesc, "insert", "success", nil, 0, 1)
	response.Success(c, gin.H{"insertedId": result.InsertedID})
}

// MongoUpdateDocuments 更新 MongoDB 文档
func (s *MiddlewareService) MongoUpdateDocuments(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMongoDB(c)
	if !ok {
		return
	}
	defer client.Disconnect(c.Request.Context())

	var req struct {
		Database   string                 `json:"database" binding:"required"`
		Collection string                 `json:"collection" binding:"required"`
		Filter     map[string]interface{} `json:"filter" binding:"required"`
		Update     map[string]interface{} `json:"update" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	cmdDesc := fmt.Sprintf("db.%s.updateMany(%v, %v)", req.Collection, req.Filter, req.Update)
	result, err := client.Database(req.Database).Collection(req.Collection).UpdateMany(
		c.Request.Context(), bson.M(req.Filter), bson.M(req.Update))
	if err != nil {
		s.writeAuditLog(c, mw, req.Database, cmdDesc, "update", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("更新失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, req.Database, cmdDesc, "update", "success", nil, 0, result.ModifiedCount)
	response.Success(c, gin.H{
		"matchedCount":  result.MatchedCount,
		"modifiedCount": result.ModifiedCount,
	})
}

// MongoDeleteDocuments 删除 MongoDB 文档
func (s *MiddlewareService) MongoDeleteDocuments(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMongoDB(c)
	if !ok {
		return
	}
	defer client.Disconnect(c.Request.Context())

	var req struct {
		Database   string                 `json:"database" binding:"required"`
		Collection string                 `json:"collection" binding:"required"`
		Filter     map[string]interface{} `json:"filter" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	cmdDesc := fmt.Sprintf("db.%s.deleteMany(%v)", req.Collection, req.Filter)
	result, err := client.Database(req.Database).Collection(req.Collection).DeleteMany(
		c.Request.Context(), bson.M(req.Filter))
	if err != nil {
		s.writeAuditLog(c, mw, req.Database, cmdDesc, "delete", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("删除失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, req.Database, cmdDesc, "delete", "success", nil, 0, result.DeletedCount)
	response.Success(c, gin.H{"deletedCount": result.DeletedCount})
}

// GetMongoStats 获取 MongoDB 服务器状态
func (s *MiddlewareService) GetMongoStats(c *gin.Context) {
	_, client, ok := s.getMiddlewareMongoDB(c)
	if !ok {
		return
	}
	defer client.Disconnect(c.Request.Context())

	var result bson.M
	err := client.Database("admin").RunCommand(c.Request.Context(), bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取状态失败: %v", err))
		return
	}

	response.Success(c, result)
}

// CreateDatabase 创建 MySQL 数据库
func (s *MiddlewareService) CreateDatabase(c *gin.Context) {
	_, db, ok := s.getMiddlewareMySQL(c)
	if !ok {
		return
	}
	defer db.Close()

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", req.Name)
	if _, err := db.ExecContext(c.Request.Context(), query); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建数据库失败: %v", err))
		return
	}

	response.SuccessWithMessage(c, "创建成功", nil)
}

// CreateClickHouseDatabase 创建 ClickHouse 数据库
func (s *MiddlewareService) CreateClickHouseDatabase(c *gin.Context) {
	mw, db, ok := s.getMiddlewareClickHouse(c)
	if !ok {
		return
	}
	defer db.Close()

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", req.Name)
	if _, err := db.ExecContext(c.Request.Context(), query); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("CREATE DATABASE %s", req.Name), "ddl", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建数据库失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", fmt.Sprintf("CREATE DATABASE %s", req.Name), "ddl", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// CreateMongoCollection 创建 MongoDB 集合
func (s *MiddlewareService) CreateMongoCollection(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMongoDB(c)
	if !ok {
		return
	}
	defer client.Disconnect(c.Request.Context())

	var req struct {
		Database   string `json:"database" binding:"required"`
		Collection string `json:"collection" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	cmdDesc := fmt.Sprintf("db.createCollection('%s')", req.Collection)
	if err := client.Database(req.Database).CreateCollection(c.Request.Context(), req.Collection); err != nil {
		s.writeAuditLog(c, mw, req.Database, cmdDesc, "ddl", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建集合失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, req.Database, cmdDesc, "ddl", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// GetMongoCollectionStats 获取 MongoDB 集合统计
func (s *MiddlewareService) GetMongoCollectionStats(c *gin.Context) {
	_, client, ok := s.getMiddlewareMongoDB(c)
	if !ok {
		return
	}
	defer client.Disconnect(c.Request.Context())

	database := c.Query("database")
	collection := c.Query("collection")
	if database == "" || collection == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 database 或 collection 参数")
		return
	}

	var result bson.M
	err := client.Database(database).RunCommand(c.Request.Context(), bson.D{{Key: "collStats", Value: collection}}).Decode(&result)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取集合统计失败: %v", err))
		return
	}

	response.Success(c, result)
}

// ===== Kafka 专用 API =====

// getMiddlewareKafka 获取中间件并建立 Kafka 客户端连接
func (s *MiddlewareService) getMiddlewareKafka(c *gin.Context) (*assetbiz.Middleware, sarama.Client, func(), bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return nil, nil, nil, false
	}

	mw, err := s.middlewareUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取中间件信息失败: "+err.Error())
		return nil, nil, nil, false
	}

	if mw.Type != assetbiz.MiddlewareTypeKafka {
		response.ErrorCode(c, http.StatusBadRequest, "该接口仅支持 Kafka 类型中间件")
		return nil, nil, nil, false
	}

	connector := &KafkaConnector{}
	config, cleanup, err := connector.buildConfig(mw)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("配置错误: %v", err))
		return nil, nil, nil, false
	}

	client, err := sarama.NewClient(connector.getBrokers(mw), config)
	if err != nil {
		cleanup()
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("连接失败: %v", err))
		return nil, nil, nil, false
	}

	return mw, client, cleanup, true
}

// GetKafkaBrokers 获取 Kafka 集群 Broker 列表
func (s *MiddlewareService) GetKafkaBrokers(c *gin.Context) {
	_, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	brokers := client.Brokers()
	controller, _ := client.Controller()

	type BrokerInfo struct {
		ID           int32  `json:"id"`
		Addr         string `json:"addr"`
		Rack         string `json:"rack,omitempty"`
		IsController bool   `json:"isController"`
	}

	var result []BrokerInfo
	for _, b := range brokers {
		info := BrokerInfo{
			ID:   b.ID(),
			Addr: b.Addr(),
		}
		if controller != nil && b.ID() == controller.ID() {
			info.IsController = true
		}
		result = append(result, info)
	}

	response.Success(c, result)
}

// PLACEHOLDER_KAFKA_TOPIC_HANDLERS

// ListKafkaTopics 列出所有 Topic
func (s *MiddlewareService) ListKafkaTopics(c *gin.Context) {
	_, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	topics, err := client.Topics()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取 Topic 列表失败: %v", err))
		return
	}

	type TopicInfo struct {
		Name       string `json:"name"`
		Partitions int    `json:"partitions"`
		Replicas   int    `json:"replicas"`
		IsInternal bool   `json:"isInternal"`
	}

	sort.Strings(topics)
	var result []TopicInfo
	for _, t := range topics {
		partitions, _ := client.Partitions(t)
		replicas := 0
		if len(partitions) > 0 {
			reps, _ := client.Replicas(t, partitions[0])
			replicas = len(reps)
		}
		isInternal := strings.HasPrefix(t, "__")
		result = append(result, TopicInfo{
			Name:       t,
			Partitions: len(partitions),
			Replicas:   replicas,
			IsInternal: isInternal,
		})
	}

	response.Success(c, result)
}

// CreateKafkaTopic 创建 Topic
func (s *MiddlewareService) CreateKafkaTopic(c *gin.Context) {
	mw, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	var req struct {
		Name              string            `json:"name" binding:"required"`
		NumPartitions     int32             `json:"numPartitions"`
		ReplicationFactor int16             `json:"replicationFactor"`
		Config            map[string]string `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.NumPartitions <= 0 {
		req.NumPartitions = 1
	}
	if req.ReplicationFactor <= 0 {
		req.ReplicationFactor = 1
	}

	// PLACEHOLDER_CREATE_TOPIC_BODY

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建管理客户端失败: %v", err))
		return
	}

	configEntries := make(map[string]*string)
	for k, v := range req.Config {
		val := v
		configEntries[k] = &val
	}

	detail := &sarama.TopicDetail{
		NumPartitions:     req.NumPartitions,
		ReplicationFactor: req.ReplicationFactor,
		ConfigEntries:     configEntries,
	}

	if err := admin.CreateTopic(req.Name, detail, false); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("CREATE TOPIC %s", req.Name), "ddl", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建 Topic 失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", fmt.Sprintf("CREATE TOPIC %s", req.Name), "ddl", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// DeleteKafkaTopic 删除 Topic
func (s *MiddlewareService) DeleteKafkaTopic(c *gin.Context) {
	mw, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	topicName := c.Query("topic")
	if topicName == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 topic 参数")
		return
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建管理客户端失败: %v", err))
		return
	}

	if err := admin.DeleteTopic(topicName); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("DELETE TOPIC %s", topicName), "ddl", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("删除 Topic 失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", fmt.Sprintf("DELETE TOPIC %s", topicName), "ddl", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "删除成功", nil)
}

// PLACEHOLDER_KAFKA_TOPIC_DETAIL

// GetKafkaTopicDetail 获取 Topic 详情
func (s *MiddlewareService) GetKafkaTopicDetail(c *gin.Context) {
	_, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	topicName := c.Query("topic")
	if topicName == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 topic 参数")
		return
	}

	partitions, err := client.Partitions(topicName)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取分区信息失败: %v", err))
		return
	}

	type PartitionDetail struct {
		ID       int32   `json:"id"`
		Leader   int32   `json:"leader"`
		Replicas []int32 `json:"replicas"`
		ISR      []int32 `json:"isr"`
	}

	var partitionDetails []PartitionDetail
	for _, p := range partitions {
		leader, _ := client.Leader(topicName, p)
		replicas, _ := client.Replicas(topicName, p)
		isr, _ := client.InSyncReplicas(topicName, p)
		leaderID := int32(-1)
		if leader != nil {
			leaderID = leader.ID()
		}
		partitionDetails = append(partitionDetails, PartitionDetail{
			ID:       p,
			Leader:   leaderID,
			Replicas: replicas,
			ISR:      isr,
		})
	}

	// 获取关联消费组
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		response.Success(c, gin.H{"partitions": partitionDetails, "consumerGroups": []interface{}{}})
		return
	}

	groups, _ := admin.ListConsumerGroups()
	type GroupLag struct {
		GroupID  string `json:"groupId"`
		TotalLag int64  `json:"totalLag"`
	}
	var relatedGroups []GroupLag
	for groupID := range groups {
		offsets, err := admin.ListConsumerGroupOffsets(groupID, map[string][]int32{topicName: partitions})
		if err != nil {
			continue
		}
		totalLag := int64(0)
		hasOffset := false
		for _, p := range partitions {
			block := offsets.GetBlock(topicName, p)
			if block != nil && block.Offset > 0 {
				hasOffset = true
				newest, err := client.GetOffset(topicName, p, sarama.OffsetNewest)
				if err == nil {
					lag := newest - block.Offset
					if lag > 0 {
						totalLag += lag
					}
				}
			}
		}
		if hasOffset {
			relatedGroups = append(relatedGroups, GroupLag{GroupID: groupID, TotalLag: totalLag})
		}
	}

	response.Success(c, gin.H{
		"partitions":     partitionDetails,
		"consumerGroups": relatedGroups,
	})
}

// PLACEHOLDER_KAFKA_TOPIC_CONFIG

// GetKafkaTopicConfig 获取 Topic 配置
func (s *MiddlewareService) GetKafkaTopicConfig(c *gin.Context) {
	_, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	topicName := c.Query("topic")
	if topicName == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 topic 参数")
		return
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建管理客户端失败: %v", err))
		return
	}

	entries, err := admin.DescribeConfig(sarama.ConfigResource{
		Type: sarama.TopicResource,
		Name: topicName,
	})
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取配置失败: %v", err))
		return
	}

	type ConfigEntry struct {
		Name     string `json:"name"`
		Value    string `json:"value"`
		ReadOnly bool   `json:"readOnly"`
		Default  bool   `json:"default"`
		Source   string `json:"source"`
	}

	var result []ConfigEntry
	for _, e := range entries {
		result = append(result, ConfigEntry{
			Name:     e.Name,
			Value:    e.Value,
			ReadOnly: e.ReadOnly,
			Default:  e.Default,
			Source:   e.Source.String(),
		})
	}

	response.Success(c, result)
}

// UpdateKafkaTopicConfig 修改 Topic 配置
func (s *MiddlewareService) UpdateKafkaTopicConfig(c *gin.Context) {
	mw, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	var req struct {
		Topic   string            `json:"topic" binding:"required"`
		Configs map[string]string `json:"configs" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建管理客户端失败: %v", err))
		return
	}

	configEntries := make(map[string]*string)
	for k, v := range req.Configs {
		val := v
		configEntries[k] = &val
	}

	if err := admin.AlterConfig(sarama.TopicResource, req.Topic, configEntries, false); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("ALTER TOPIC CONFIG %s", req.Topic), "other", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("修改配置失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", fmt.Sprintf("ALTER TOPIC CONFIG %s", req.Topic), "other", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "修改成功", nil)
}

// PLACEHOLDER_KAFKA_CONSUMER_GROUPS

// ListKafkaConsumerGroups 列出所有消费组
func (s *MiddlewareService) ListKafkaConsumerGroups(c *gin.Context) {
	_, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建管理客户端失败: %v", err))
		return
	}

	groups, err := admin.ListConsumerGroups()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取消费组列表失败: %v", err))
		return
	}

	type GroupInfo struct {
		GroupID      string `json:"groupId"`
		ProtocolType string `json:"protocolType"`
	}

	var result []GroupInfo
	for id, protocolType := range groups {
		result = append(result, GroupInfo{GroupID: id, ProtocolType: protocolType})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].GroupID < result[j].GroupID })
	response.Success(c, result)
}

// GetKafkaConsumerGroupDetail 获取消费组详情
func (s *MiddlewareService) GetKafkaConsumerGroupDetail(c *gin.Context) {
	_, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	groupID := c.Query("groupId")
	if groupID == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 groupId 参数")
		return
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建管理客户端失败: %v", err))
		return
	}

	desc, err := admin.DescribeConsumerGroups([]string{groupID})
	if err != nil || len(desc) == 0 {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取消费组详情失败: %v", err))
		return
	}

	group := desc[0]

	// PLACEHOLDER_CONSUMER_GROUP_DETAIL_BODY

	type MemberInfo struct {
		ClientID    string   `json:"clientId"`
		ClientHost  string   `json:"clientHost"`
		Assignments []string `json:"assignments"`
	}

	var members []MemberInfo
	// Collect all topic-partitions for lag calculation
	topicPartitions := make(map[string][]int32)
	for _, member := range group.Members {
		assignment, err := member.GetMemberAssignment()
		var assignments []string
		if err == nil && assignment != nil {
			for topic, parts := range assignment.Topics {
				for _, p := range parts {
					assignments = append(assignments, fmt.Sprintf("%s-%d", topic, p))
					topicPartitions[topic] = appendUnique(topicPartitions[topic], p)
				}
			}
		}
		members = append(members, MemberInfo{
			ClientID:    member.ClientId,
			ClientHost:  member.ClientHost,
			Assignments: assignments,
		})
	}

	// Calculate lag
	type LagInfo struct {
		Topic          string `json:"topic"`
		Partition      int32  `json:"partition"`
		CurrentOffset  int64  `json:"currentOffset"`
		LogEndOffset   int64  `json:"logEndOffset"`
		Lag            int64  `json:"lag"`
	}

	offsets, _ := admin.ListConsumerGroupOffsets(groupID, topicPartitions)
	var lagInfos []LagInfo
	if offsets != nil {
		for topic, parts := range topicPartitions {
			for _, p := range parts {
				block := offsets.GetBlock(topic, p)
				currentOffset := int64(-1)
				if block != nil {
					currentOffset = block.Offset
				}
				logEnd, _ := client.GetOffset(topic, p, sarama.OffsetNewest)
				lag := int64(0)
				if currentOffset >= 0 && logEnd > currentOffset {
					lag = logEnd - currentOffset
				}
				lagInfos = append(lagInfos, LagInfo{
					Topic:         topic,
					Partition:     p,
					CurrentOffset: currentOffset,
					LogEndOffset:  logEnd,
					Lag:           lag,
				})
			}
		}
	}

	response.Success(c, gin.H{
		"groupId":  group.GroupId,
		"state":    group.State,
		"protocol": group.Protocol,
		"members":  members,
		"lag":      lagInfos,
	})
}

// appendUnique appends val to slice if not already present
func appendUnique(slice []int32, val int32) []int32 {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

// DeleteKafkaConsumerGroup 删除消费组
func (s *MiddlewareService) DeleteKafkaConsumerGroup(c *gin.Context) {
	mw, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	groupID := c.Query("groupId")
	if groupID == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 groupId 参数")
		return
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建管理客户端失败: %v", err))
		return
	}

	if err := admin.DeleteConsumerGroup(groupID); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("DELETE CONSUMER GROUP %s", groupID), "delete", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("删除消费组失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", fmt.Sprintf("DELETE CONSUMER GROUP %s", groupID), "delete", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "删除成功", nil)
}

// PLACEHOLDER_KAFKA_MESSAGE_HANDLERS

// ProduceKafkaMessage 发送消息到 Kafka
func (s *MiddlewareService) ProduceKafkaMessage(c *gin.Context) {
	mw, client, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	defer cleanup()
	defer client.Close()

	var req struct {
		Topic     string            `json:"topic" binding:"required"`
		Key       string            `json:"key"`
		Value     string            `json:"value" binding:"required"`
		Headers   map[string]string `json:"headers"`
		Partition *int32            `json:"partition"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	connector := &KafkaConnector{}
	config, prodCleanup, err := connector.buildConfig(mw)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("配置错误: %v", err))
		return
	}
	defer prodCleanup()

	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(connector.getBrokers(mw), config)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建生产者失败: %v", err))
		return
	}
	defer producer.Close()

	msg := &sarama.ProducerMessage{
		Topic: req.Topic,
		Value: sarama.StringEncoder(req.Value),
	}
	if req.Key != "" {
		msg.Key = sarama.StringEncoder(req.Key)
	}
	for k, v := range req.Headers {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{
			Key:   []byte(k),
			Value: []byte(v),
		})
	}
	if req.Partition != nil {
		msg.Partition = *req.Partition
	}

	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("PRODUCE %s", req.Topic), "insert", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("发送失败: %v", err))
		return
	}

	s.writeAuditLog(c, mw, "", fmt.Sprintf("PRODUCE %s", req.Topic), "insert", "success", nil, 0, 1)
	response.Success(c, gin.H{
		"partition": partition,
		"offset":    offset,
	})
}

// StartKafkaConsumerSession 启动消费监听会话
func (s *MiddlewareService) StartKafkaConsumerSession(c *gin.Context) {
	mw, _, cleanup, ok := s.getMiddlewareKafka(c)
	if !ok {
		return
	}
	cleanup()

	var req struct {
		Topic       string `json:"topic" binding:"required"`
		StartOffset string `json:"startOffset"` // "latest" or "earliest"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	connector := &KafkaConnector{}
	config, consCleanup, err := connector.buildConfig(mw)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("配置错误: %v", err))
		return
	}

	consumer, err := sarama.NewConsumer(connector.getBrokers(mw), config)
	if err != nil {
		consCleanup()
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建消费者失败: %v", err))
		return
	}
	// Note: consCleanup will be called when session stops (temp files cleaned)
	// For simplicity, clean up now since config is already applied
	consCleanup()

	startOffset := sarama.OffsetNewest
	if req.StartOffset == "earliest" {
		startOffset = sarama.OffsetOldest
	}

	sessionID, err := kafkaSessionManager.StartSession(consumer, req.Topic, startOffset)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("启动消费会话失败: %v", err))
		return
	}

	response.Success(c, gin.H{"sessionId": sessionID})
}

// PollKafkaConsumerSession 拉取消费消息
func (s *MiddlewareService) PollKafkaConsumerSession(c *gin.Context) {
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 sessionId 参数")
		return
	}

	keyword := c.Query("keyword")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))

	messages, ok := kafkaSessionManager.PollMessages(sessionID, keyword, limit)
	if !ok {
		response.ErrorCode(c, http.StatusNotFound, "会话不存在或已过期")
		return
	}

	response.Success(c, messages)
}

// StopKafkaConsumerSession 停止消费会话
func (s *MiddlewareService) StopKafkaConsumerSession(c *gin.Context) {
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 sessionId 参数")
		return
	}

	kafkaSessionManager.StopSession(sessionID)
	response.SuccessWithMessage(c, "已停止", nil)
}

// ===== Milvus 专用 API =====

// getMiddlewareMilvus 获取 Milvus 中间件和客户端连接
func (s *MiddlewareService) getMiddlewareMilvus(c *gin.Context) (*assetbiz.Middleware, milvusclient.Client, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的中间件ID")
		return nil, nil, false
	}

	mw, err := s.middlewareUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取中间件信息失败: "+err.Error())
		return nil, nil, false
	}

	if mw.Type != assetbiz.MiddlewareTypeMilvus {
		response.ErrorCode(c, http.StatusBadRequest, "该接口仅支持 Milvus 类型中间件")
		return nil, nil, false
	}

	connector := &MilvusConnector{}
	client, err := connector.connect(c.Request.Context(), mw)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("连接失败: %v", err))
		return nil, nil, false
	}

	return mw, client, true
}

// ListMilvusDatabases 列出 Milvus 数据库
func (s *MiddlewareService) ListMilvusDatabases(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	dbs, err := client.ListDatabases(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取数据库列表失败: %v", err))
		return
	}

	names := make([]string, len(dbs))
	for i, db := range dbs {
		names[i] = db.Name
	}
	response.Success(c, names)
}

// MILVUS_HANDLERS_PLACEHOLDER

// CreateMilvusDatabase 创建 Milvus 数据库
func (s *MiddlewareService) CreateMilvusDatabase(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := client.CreateDatabase(c.Request.Context(), req.Name); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("CREATE DATABASE %s", req.Name), "ddl", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建数据库失败: %v", err))
		return
	}
	s.writeAuditLog(c, mw, "", fmt.Sprintf("CREATE DATABASE %s", req.Name), "ddl", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// DropMilvusDatabase 删除 Milvus 数据库
func (s *MiddlewareService) DropMilvusDatabase(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	name := c.Query("name")
	if name == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 name 参数")
		return
	}

	if err := client.DropDatabase(c.Request.Context(), name); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("DROP DATABASE %s", name), "ddl", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("删除数据库失败: %v", err))
		return
	}
	s.writeAuditLog(c, mw, "", fmt.Sprintf("DROP DATABASE %s", name), "ddl", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "删除成功", nil)
}

// MILVUS_COLLECTION_PLACEHOLDER

// ListMilvusCollections 列出 Collection
func (s *MiddlewareService) ListMilvusCollections(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	dbName := c.Query("database")
	if dbName != "" {
		if err := client.UsingDatabase(c.Request.Context(), dbName); err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("切换数据库失败: %v", err))
			return
		}
	}

	collections, err := client.ListCollections(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取 Collection 列表失败: %v", err))
		return
	}

	type collInfo struct {
		Name   string `json:"name"`
		ID     int64  `json:"id"`
		Loaded bool   `json:"loaded"`
	}
	result := make([]collInfo, len(collections))
	for i, coll := range collections {
		result[i] = collInfo{Name: coll.Name, ID: coll.ID, Loaded: coll.Loaded}
	}
	response.Success(c, result)
}

// DescribeMilvusCollection 获取 Collection 详情
func (s *MiddlewareService) DescribeMilvusCollection(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	dbName := c.Query("database")
	if dbName != "" {
		_ = client.UsingDatabase(c.Request.Context(), dbName)
	}

	collName := c.Query("collection")
	if collName == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 collection 参数")
		return
	}

	coll, err := client.DescribeCollection(c.Request.Context(), collName)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取 Collection 详情失败: %v", err))
		return
	}

	type fieldInfo struct {
		Name           string            `json:"name"`
		DataType       string            `json:"dataType"`
		PrimaryKey     bool              `json:"primaryKey"`
		AutoID         bool              `json:"autoID"`
		Description    string            `json:"description"`
		TypeParams     map[string]string `json:"typeParams"`
		IsDynamic      bool              `json:"isDynamic"`
		IsPartitionKey bool              `json:"isPartitionKey"`
	}
	fields := make([]fieldInfo, len(coll.Schema.Fields))
	for i, f := range coll.Schema.Fields {
		fields[i] = fieldInfo{
			Name: f.Name, DataType: f.DataType.Name(), PrimaryKey: f.PrimaryKey,
			AutoID: f.AutoID, Description: f.Description, TypeParams: f.TypeParams,
			IsDynamic: f.IsDynamic, IsPartitionKey: f.IsPartitionKey,
		}
	}

	type indexInfo struct {
		FieldName string            `json:"fieldName"`
		IndexName string            `json:"indexName"`
		IndexType string            `json:"indexType"`
		Params    map[string]string `json:"params"`
	}
	var indexes []indexInfo
	for _, f := range coll.Schema.Fields {
		idxs, err := client.DescribeIndex(c.Request.Context(), collName, f.Name)
		if err != nil {
			continue
		}
		for _, idx := range idxs {
			indexes = append(indexes, indexInfo{
				FieldName: f.Name, IndexName: idx.Name(),
				IndexType: string(idx.IndexType()), Params: idx.Params(),
			})
		}
	}

	stats, _ := client.GetCollectionStatistics(c.Request.Context(), collName)
	loadState, _ := client.GetLoadState(c.Request.Context(), collName, nil)
	loadStateStr := "NotLoad"
	switch loadState {
	case entity.LoadStateLoaded:
		loadStateStr = "Loaded"
	case entity.LoadStateLoading:
		loadStateStr = "Loading"
	}

	response.Success(c, gin.H{
		"name": coll.Name, "id": coll.ID, "fields": fields, "indexes": indexes,
		"enableDynamic": coll.Schema.EnableDynamicField, "description": coll.Schema.Description,
		"consistencyLevel": coll.ConsistencyLevel.CommonConsistencyLevel().String(),
		"loadState": loadStateStr, "statistics": stats, "properties": coll.Properties,
	})
}

// MILVUS_NEXT_HANDLERS

// CreateMilvusCollection 创建 Collection
func (s *MiddlewareService) CreateMilvusCollection(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Database    string `json:"database"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		AutoID      bool   `json:"autoID"`
		EnableDynamic bool `json:"enableDynamic"`
		Fields      []struct {
			Name        string `json:"name"`
			DataType    string `json:"dataType"`
			PrimaryKey  bool   `json:"primaryKey"`
			AutoID      bool   `json:"autoID"`
			Description string `json:"description"`
			Dim         int    `json:"dim"`
			MaxLength   int    `json:"maxLength"`
		} `json:"fields" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if req.Database != "" {
		_ = client.UsingDatabase(c.Request.Context(), req.Database)
	}

	schema := entity.NewSchema().WithName(req.Name).WithDescription(req.Description).
		WithAutoID(req.AutoID).WithDynamicFieldEnabled(req.EnableDynamic)

	for _, f := range req.Fields {
		field := entity.NewField().WithName(f.Name).WithDescription(f.Description).
			WithIsPrimaryKey(f.PrimaryKey).WithIsAutoID(f.AutoID).
			WithDataType(milvusFieldType(f.DataType))
		if f.Dim > 0 {
			field.WithDim(int64(f.Dim))
		}
		if f.MaxLength > 0 {
			field.WithMaxLength(int64(f.MaxLength))
		}
		schema.WithField(field)
	}

	if err := client.CreateCollection(c.Request.Context(), schema, entity.DefaultShardNumber); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("CREATE COLLECTION %s", req.Name), "ddl", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建 Collection 失败: %v", err))
		return
	}
	s.writeAuditLog(c, mw, "", fmt.Sprintf("CREATE COLLECTION %s", req.Name), "ddl", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// MILVUS_DROP_COLL_PLACEHOLDER

// milvusFieldType 将字符串转换为 Milvus FieldType
func milvusFieldType(s string) entity.FieldType {
	switch strings.ToLower(s) {
	case "bool":
		return entity.FieldTypeBool
	case "int8":
		return entity.FieldTypeInt8
	case "int16":
		return entity.FieldTypeInt16
	case "int32":
		return entity.FieldTypeInt32
	case "int64":
		return entity.FieldTypeInt64
	case "float":
		return entity.FieldTypeFloat
	case "double":
		return entity.FieldTypeDouble
	case "varchar", "string":
		return entity.FieldTypeVarChar
	case "json":
		return entity.FieldTypeJSON
	case "array":
		return entity.FieldTypeArray
	case "floatvector":
		return entity.FieldTypeFloatVector
	case "binaryvector":
		return entity.FieldTypeBinaryVector
	default:
		return entity.FieldTypeVarChar
	}
}

// DropMilvusCollection 删除 Collection
func (s *MiddlewareService) DropMilvusCollection(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	dbName := c.Query("database")
	if dbName != "" {
		_ = client.UsingDatabase(c.Request.Context(), dbName)
	}

	name := c.Query("collection")
	if name == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 collection 参数")
		return
	}

	if err := client.DropCollection(c.Request.Context(), name); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("DROP COLLECTION %s", name), "ddl", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("删除 Collection 失败: %v", err))
		return
	}
	s.writeAuditLog(c, mw, "", fmt.Sprintf("DROP COLLECTION %s", name), "ddl", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "删除成功", nil)
}

// LoadMilvusCollection 加载 Collection 到内存
func (s *MiddlewareService) LoadMilvusCollection(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Database   string `json:"database"`
		Collection string `json:"collection" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.Database != "" {
		_ = client.UsingDatabase(c.Request.Context(), req.Database)
	}

	if err := client.LoadCollection(c.Request.Context(), req.Collection, true); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("加载失败: %v", err))
		return
	}
	response.SuccessWithMessage(c, "加载请求已提交", nil)
}

// ReleaseMilvusCollection 释放 Collection
func (s *MiddlewareService) ReleaseMilvusCollection(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Database   string `json:"database"`
		Collection string `json:"collection" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.Database != "" {
		_ = client.UsingDatabase(c.Request.Context(), req.Database)
	}

	if err := client.ReleaseCollection(c.Request.Context(), req.Collection); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("释放失败: %v", err))
		return
	}
	response.SuccessWithMessage(c, "释放成功", nil)
}

// MILVUS_INDEX_PLACEHOLDER

// CreateMilvusIndex 创建索引
func (s *MiddlewareService) CreateMilvusIndex(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Database   string            `json:"database"`
		Collection string            `json:"collection" binding:"required"`
		FieldName  string            `json:"fieldName" binding:"required"`
		IndexName  string            `json:"indexName"`
		IndexType  string            `json:"indexType" binding:"required"`
		MetricType string            `json:"metricType"`
		Params     map[string]string `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.Database != "" {
		_ = client.UsingDatabase(c.Request.Context(), req.Database)
	}

	params := req.Params
	if params == nil {
		params = make(map[string]string)
	}
	if req.MetricType != "" {
		params["metric_type"] = req.MetricType
	}
	idx := entity.NewGenericIndex(req.IndexName, entity.IndexType(req.IndexType), params)

	if err := client.CreateIndex(c.Request.Context(), req.Collection, req.FieldName, idx, false); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建索引失败: %v", err))
		return
	}
	response.SuccessWithMessage(c, "创建成功", nil)
}

// DropMilvusIndex 删除索引
func (s *MiddlewareService) DropMilvusIndex(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

// MILVUS_DROP_INDEX_CONT

	dbName := c.Query("database")
	if dbName != "" {
		_ = client.UsingDatabase(c.Request.Context(), dbName)
	}
	collName := c.Query("collection")
	fieldName := c.Query("fieldName")
	if collName == "" || fieldName == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 collection 或 fieldName 参数")
		return
	}

	if err := client.DropIndex(c.Request.Context(), collName, fieldName); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("删除索引失败: %v", err))
		return
	}
	response.SuccessWithMessage(c, "删除成功", nil)
}

// QueryMilvusData 表达式查询
func (s *MiddlewareService) QueryMilvusData(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Database     string   `json:"database"`
		Collection   string   `json:"collection" binding:"required"`
		Filter       string   `json:"filter"`
		OutputFields []string `json:"outputFields"`
		Limit        int      `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.Database != "" {
		_ = client.UsingDatabase(c.Request.Context(), req.Database)
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}

	var opts []milvusclient.SearchQueryOptionFunc
	opts = append(opts, milvusclient.WithLimit(int64(req.Limit)))

	expr := req.Filter
	if expr == "" {
		expr = ""
	}

// MILVUS_QUERY_CONT

	resultSet, err := client.Query(c.Request.Context(), req.Collection, nil, expr, req.OutputFields, opts...)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}

	rows := milvusColumnsToRows(resultSet)
	response.Success(c, gin.H{"rows": rows, "total": len(rows)})
}

// milvusColumnsToRows 将列式数据转换为行式数据
func milvusColumnsToRows(columns []entity.Column) []map[string]interface{} {
	if len(columns) == 0 {
		return []map[string]interface{}{}
	}
	rowCount := columns[0].Len()
	rows := make([]map[string]interface{}, rowCount)
	for i := 0; i < rowCount; i++ {
		row := make(map[string]interface{})
		for _, col := range columns {
			val, err := col.Get(i)
			if err == nil {
				row[col.Name()] = val
			}
		}
		rows[i] = row
	}
	return rows
}

// milvusBuildColumns 从 JSON 行数据构建 Milvus 列数据，处理 JSON 反序列化的类型转换
func milvusBuildColumns(schema *entity.Schema, rows []map[string]interface{}) ([]entity.Column, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("空数据")
	}

	var columns []entity.Column
	for _, field := range schema.Fields {
		if field.PrimaryKey && field.AutoID {
			continue
		}
		col, err := milvusBuildColumn(field, rows)
		if err != nil {
			return nil, fmt.Errorf("字段 %s: %v", field.Name, err)
		}
		columns = append(columns, col)
	}
	return columns, nil
}

func milvusBuildColumn(field *entity.Field, rows []map[string]interface{}) (entity.Column, error) {
	switch field.DataType {
	case entity.FieldTypeBool:
		vals := make([]bool, len(rows))
		for i, row := range rows {
			v, _ := row[field.Name]
			if b, ok := v.(bool); ok {
				vals[i] = b
			}
		}
		return entity.NewColumnBool(field.Name, vals), nil

	case entity.FieldTypeInt8:
		vals := make([]int8, len(rows))
		for i, row := range rows {
			vals[i] = int8(toFloat64(row[field.Name]))
		}
		return entity.NewColumnInt8(field.Name, vals), nil

	case entity.FieldTypeInt16:
		vals := make([]int16, len(rows))
		for i, row := range rows {
			vals[i] = int16(toFloat64(row[field.Name]))
		}
		return entity.NewColumnInt16(field.Name, vals), nil

	case entity.FieldTypeInt32:
		vals := make([]int32, len(rows))
		for i, row := range rows {
			vals[i] = int32(toFloat64(row[field.Name]))
		}
		return entity.NewColumnInt32(field.Name, vals), nil

// MILVUS_BUILD_COL_CONT

	case entity.FieldTypeInt64:
		vals := make([]int64, len(rows))
		for i, row := range rows {
			vals[i] = int64(toFloat64(row[field.Name]))
		}
		return entity.NewColumnInt64(field.Name, vals), nil

	case entity.FieldTypeFloat:
		vals := make([]float32, len(rows))
		for i, row := range rows {
			vals[i] = float32(toFloat64(row[field.Name]))
		}
		return entity.NewColumnFloat(field.Name, vals), nil

	case entity.FieldTypeDouble:
		vals := make([]float64, len(rows))
		for i, row := range rows {
			vals[i] = toFloat64(row[field.Name])
		}
		return entity.NewColumnDouble(field.Name, vals), nil

	case entity.FieldTypeString, entity.FieldTypeVarChar:
		vals := make([]string, len(rows))
		for i, row := range rows {
			if s, ok := row[field.Name].(string); ok {
				vals[i] = s
			} else if row[field.Name] != nil {
				vals[i] = fmt.Sprintf("%v", row[field.Name])
			}
		}
		return entity.NewColumnVarChar(field.Name, vals), nil

	case entity.FieldTypeJSON:
		vals := make([][]byte, len(rows))
		for i, row := range rows {
			v := row[field.Name]
			if v != nil {
				bs, _ := json.Marshal(v)
				vals[i] = bs
			} else {
				vals[i] = []byte("{}")
			}
		}
		return entity.NewColumnJSONBytes(field.Name, vals), nil

	case entity.FieldTypeFloatVector:
		dimStr := field.TypeParams["dim"]
		dim, _ := strconv.Atoi(dimStr)
		if dim == 0 {
			dim = 128
		}
		vals := make([][]float32, len(rows))
		for i, row := range rows {
			vals[i] = toFloat32Slice(row[field.Name], dim)
		}
		return entity.NewColumnFloatVector(field.Name, dim, vals), nil

	case entity.FieldTypeBinaryVector:
		dimStr := field.TypeParams["dim"]
		dim, _ := strconv.Atoi(dimStr)
		if dim == 0 {
			dim = 128
		}
		vals := make([][]byte, len(rows))
		for i, row := range rows {
			vals[i] = toBytesSlice(row[field.Name], dim/8)
		}
		return entity.NewColumnBinaryVector(field.Name, dim, vals), nil

	default:
		return nil, fmt.Errorf("不支持的字段类型: %s", field.DataType.Name())
	}
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func toFloat32Slice(v interface{}, dim int) []float32 {
	result := make([]float32, dim)
	switch arr := v.(type) {
	case []interface{}:
		for i := 0; i < len(arr) && i < dim; i++ {
			result[i] = float32(toFloat64(arr[i]))
		}
	case []float32:
		copy(result, arr)
	}
	return result
}

func toBytesSlice(v interface{}, size int) []byte {
	result := make([]byte, size)
	if arr, ok := v.([]interface{}); ok {
		for i := 0; i < len(arr) && i < size; i++ {
			result[i] = byte(toFloat64(arr[i]))
		}
	}
	return result
}

// InsertMilvusData 插入数据
func (s *MiddlewareService) InsertMilvusData(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Database   string                   `json:"database"`
		Collection string                   `json:"collection" binding:"required"`
		Partition  string                   `json:"partition"`
		Rows       []map[string]interface{} `json:"rows" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.Database != "" {
		_ = client.UsingDatabase(c.Request.Context(), req.Database)
	}

	// 获取 collection schema 用于类型转换
	coll, err := client.DescribeCollection(c.Request.Context(), req.Collection)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取 Collection Schema 失败: %v", err))
		return
	}

	columns, err := milvusBuildColumns(coll.Schema, req.Rows)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("构建列数据失败: %v", err))
		return
	}

	idCol, err := client.Insert(c.Request.Context(), req.Collection, req.Partition, columns...)
	if err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("INSERT INTO %s", req.Collection), "insert", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("插入失败: %v", err))
		return
	}
	s.writeAuditLog(c, mw, "", fmt.Sprintf("INSERT INTO %s", req.Collection), "insert", "success", nil, 0, int64(idCol.Len()))
	response.Success(c, gin.H{"insertCount": idCol.Len()})
}

// DeleteMilvusData 按表达式删除
func (s *MiddlewareService) DeleteMilvusData(c *gin.Context) {
	mw, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Database   string `json:"database"`
		Collection string `json:"collection" binding:"required"`
		Partition  string `json:"partition"`
		Filter     string `json:"filter" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.Database != "" {
		_ = client.UsingDatabase(c.Request.Context(), req.Database)
	}

	if err := client.Delete(c.Request.Context(), req.Collection, req.Partition, req.Filter); err != nil {
		s.writeAuditLog(c, mw, "", fmt.Sprintf("DELETE FROM %s WHERE %s", req.Collection, req.Filter), "delete", "failed", err, 0, 0)
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("删除失败: %v", err))
		return
	}
	s.writeAuditLog(c, mw, "", fmt.Sprintf("DELETE FROM %s WHERE %s", req.Collection, req.Filter), "delete", "success", nil, 0, 0)
	response.SuccessWithMessage(c, "删除成功", nil)
}

// MILVUS_SEARCH_PLACEHOLDER

// SearchMilvusVectors 向量搜索
func (s *MiddlewareService) SearchMilvusVectors(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Database     string      `json:"database"`
		Collection   string      `json:"collection" binding:"required"`
		VectorField  string      `json:"vectorField" binding:"required"`
		Vectors      [][]float32 `json:"vectors" binding:"required"`
		TopK         int         `json:"topK"`
		MetricType   string      `json:"metricType"`
		Filter       string      `json:"filter"`
		OutputFields []string    `json:"outputFields"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.Database != "" {
		_ = client.UsingDatabase(c.Request.Context(), req.Database)
	}
	if req.TopK <= 0 {
		req.TopK = 10
	}
	if req.MetricType == "" {
		req.MetricType = "L2"
	}

	vectors := make([]entity.Vector, len(req.Vectors))
	for i, v := range req.Vectors {
		vectors[i] = entity.FloatVector(v)
	}

	sp, _ := entity.NewIndexFlatSearchParam()
	results, err := client.Search(c.Request.Context(), req.Collection, nil,
		req.Filter, req.OutputFields, vectors, req.VectorField,
		entity.MetricType(req.MetricType), req.TopK, sp)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("搜索失败: %v", err))
		return
	}

// MILVUS_SEARCH_RESULT_CONT

	type searchHit struct {
		Score  float32                `json:"score"`
		Fields map[string]interface{} `json:"fields"`
	}
	var allHits [][]searchHit
	for _, sr := range results {
		hits := make([]searchHit, sr.ResultCount)
		rows := milvusColumnsToRows(sr.Fields)
		for i := 0; i < sr.ResultCount; i++ {
			hit := searchHit{Score: sr.Scores[i]}
			if i < len(rows) {
				hit.Fields = rows[i]
			} else {
				hit.Fields = map[string]interface{}{}
			}
			// 添加 ID
			if sr.IDs != nil {
				val, err := sr.IDs.Get(i)
				if err == nil {
					hit.Fields["_id"] = val
				}
			}
			hits[i] = hit
		}
		allHits = append(allHits, hits)
	}
	response.Success(c, gin.H{"results": allHits})
}

// ListMilvusPartitions 列出分区
func (s *MiddlewareService) ListMilvusPartitions(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	dbName := c.Query("database")
	if dbName != "" {
		_ = client.UsingDatabase(c.Request.Context(), dbName)
	}
	collName := c.Query("collection")
	if collName == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 collection 参数")
		return
	}

	partitions, err := client.ShowPartitions(c.Request.Context(), collName)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("获取分区列表失败: %v", err))
		return
	}

	type partInfo struct {
		Name   string `json:"name"`
		ID     int64  `json:"id"`
		Loaded bool   `json:"loaded"`
	}
	result := make([]partInfo, len(partitions))
	for i, p := range partitions {
		result[i] = partInfo{Name: p.Name, ID: p.ID, Loaded: p.Loaded}
	}
	response.Success(c, result)
}

// CreateMilvusPartition 创建分区
func (s *MiddlewareService) CreateMilvusPartition(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	var req struct {
		Database   string `json:"database"`
		Collection string `json:"collection" binding:"required"`
		Partition  string `json:"partition" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.Database != "" {
		_ = client.UsingDatabase(c.Request.Context(), req.Database)
	}

	if err := client.CreatePartition(c.Request.Context(), req.Collection, req.Partition); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建分区失败: %v", err))
		return
	}
	response.SuccessWithMessage(c, "创建成功", nil)
}

// MILVUS_FINAL_HANDLERS

// DropMilvusPartition 删除分区
func (s *MiddlewareService) DropMilvusPartition(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	dbName := c.Query("database")
	if dbName != "" {
		_ = client.UsingDatabase(c.Request.Context(), dbName)
	}
	collName := c.Query("collection")
	partition := c.Query("partition")
	if collName == "" || partition == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 collection 或 partition 参数")
		return
	}

	if err := client.DropPartition(c.Request.Context(), collName, partition); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("删除分区失败: %v", err))
		return
	}
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetMilvusMetrics 获取系统指标
func (s *MiddlewareService) GetMilvusMetrics(c *gin.Context) {
	_, client, ok := s.getMiddlewareMilvus(c)
	if !ok {
		return
	}
	defer client.Close()

	version, _ := client.GetVersion(c.Request.Context())

	// 获取数据库列表
	dbs, _ := client.ListDatabases(c.Request.Context())
	dbNames := make([]string, len(dbs))
	for i, db := range dbs {
		dbNames[i] = db.Name
	}

	// 获取 collection 列表
	collections, _ := client.ListCollections(c.Request.Context())
	collCount := len(collections)

	response.Success(c, gin.H{
		"version":         version,
		"databases":       dbNames,
		"collectionCount": collCount,
	})
}
