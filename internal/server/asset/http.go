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
	"github.com/gin-gonic/gin"
	assetService "github.com/ydcloud-dy/opshub/internal/service/asset"
	assetdata "github.com/ydcloud-dy/opshub/internal/data/asset"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	rbacService "github.com/ydcloud-dy/opshub/internal/service/rbac"
	rbacdata "github.com/ydcloud-dy/opshub/internal/data/rbac"
	rbacbiz "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"gorm.io/gorm"
)

type HTTPServer struct {
	assetGroupService            *assetService.AssetGroupService
	hostService                  *assetService.HostService
	middlewareService            *assetService.MiddlewareService
	middlewarePermissionService  *rbacService.MiddlewarePermissionService
	terminalManager              *TerminalManager
	terminalAuditHandler         *TerminalAuditHandler
	authMiddleware               *rbacService.AuthMiddleware
}

func NewHTTPServer(
	assetGroupService *assetService.AssetGroupService,
	hostService *assetService.HostService,
	middlewareService *assetService.MiddlewareService,
	middlewarePermissionService *rbacService.MiddlewarePermissionService,
	terminalManager *TerminalManager,
	db *gorm.DB,
	authMiddleware *rbacService.AuthMiddleware,
) *HTTPServer {
	return &HTTPServer{
		assetGroupService:           assetGroupService,
		hostService:                 hostService,
		middlewareService:           middlewareService,
		middlewarePermissionService: middlewarePermissionService,
		terminalManager:             terminalManager,
		terminalAuditHandler:        NewTerminalAuditHandler(db),
		authMiddleware:              authMiddleware,
	}
}

func (s *HTTPServer) RegisterRoutes(r *gin.RouterGroup) {
	// 资产分组管理
	groups := r.Group("/asset-groups")
	{
		groups.GET("/tree", s.assetGroupService.GetGroupTree)
		groups.GET("/parent-options", s.assetGroupService.GetParentOptions)
		groups.POST("", s.assetGroupService.CreateGroup)
		groups.GET("/:id", s.assetGroupService.GetGroup)
		groups.PUT("/:id", s.assetGroupService.UpdateGroup)
		groups.DELETE("/:id", s.assetGroupService.DeleteGroup)
	}

	// 主机管理
	hosts := r.Group("/hosts")
	{
		hosts.GET("", s.hostService.ListHosts)
		hosts.GET("/template/download", s.hostService.DownloadExcelTemplate)
		hosts.POST("/import", s.hostService.ImportFromExcel)
		hosts.POST("/batch-collect", s.hostService.BatchCollectHostInfo)
		hosts.POST("/batch-delete", s.hostService.BatchDeleteHosts)

		// 查看权限 - 查看主机详情
		hosts.GET("/:id",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionView),
			s.hostService.GetHost)

		// 编辑权限 - 创建、修改主机配置
		hosts.POST("",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionEdit),
			s.hostService.CreateHost)
		hosts.PUT("/:id",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionEdit),
			s.hostService.UpdateHost)

		// 删除权限 - 删除主机
		hosts.DELETE("/:id",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionDelete),
			s.hostService.DeleteHost)

		// 采集权限 - 采集主机信息
		hosts.POST("/:id/collect",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionCollect),
			s.hostService.CollectHostInfo)
		hosts.POST("/:id/test", s.hostService.TestHostConnection)

		// 文件管理权限 - 文件上传、下载、删除
		hosts.GET("/:id/files",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionFile),
			s.hostService.ListHostFiles)
		hosts.POST("/:id/files/upload",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionFile),
			s.hostService.UploadHostFile)
		hosts.GET("/:id/files/download",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionFile),
			s.hostService.DownloadHostFile)
		hosts.DELETE("/:id/files",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionFile),
			s.hostService.DeleteHostFile)
	}

	// 凭证管理
	credentials := r.Group("/credentials")
	{
		credentials.GET("", s.hostService.ListCredentials)
		credentials.GET("/all", s.hostService.GetAllCredentials)
		credentials.GET("/:id", s.hostService.GetCredential)
		credentials.POST("", s.hostService.CreateCredential)
		credentials.PUT("/:id", s.hostService.UpdateCredential)
		credentials.DELETE("/:id", s.hostService.DeleteCredential)
	}

	// 云平台账号管理
	cloudAccounts := r.Group("/cloud-accounts")
	{
		cloudAccounts.GET("", s.hostService.ListCloudAccounts)
		cloudAccounts.GET("/all", s.hostService.GetAllCloudAccounts)
		cloudAccounts.GET("/:id", s.hostService.GetCloudAccount)
		cloudAccounts.GET("/:id/regions", s.hostService.GetCloudRegions)
		cloudAccounts.GET("/:id/instances", s.hostService.GetCloudInstances)
		cloudAccounts.POST("", s.hostService.CreateCloudAccount)
		cloudAccounts.PUT("/:id", s.hostService.UpdateCloudAccount)
		cloudAccounts.DELETE("/:id", s.hostService.DeleteCloudAccount)
		cloudAccounts.POST("/import", s.hostService.ImportFromCloud)
	}

	// SSH终端 - 终端权限
	terminal := r.Group("/asset/terminal")
	{
		terminal.GET("/:id",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionTerminal),
			s.HandleSSHConnection)
		terminal.POST("/:id/resize", s.ResizeTerminal)
	}

	// 终端审计
	terminalSessions := r.Group("/terminal-sessions")
	{
		terminalSessions.GET("", s.terminalAuditHandler.ListTerminalSessions)
		terminalSessions.GET("/:id/play", s.terminalAuditHandler.PlayTerminalSession)
		terminalSessions.DELETE("/:id", s.terminalAuditHandler.DeleteTerminalSession)
	}

	// 中间件管理
	middlewares := r.Group("/middlewares")
	{
		middlewares.GET("", s.middlewareService.ListMiddlewares)
		middlewares.GET("/:id",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermView),
			s.middlewareService.GetMiddleware)
		middlewares.POST("",
			s.middlewareService.CreateMiddleware)
		middlewares.PUT("/:id",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermEdit),
			s.middlewareService.UpdateMiddleware)
		middlewares.DELETE("/:id",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermDelete),
			s.middlewareService.DeleteMiddleware)
		middlewares.POST("/batch-delete", s.middlewareService.BatchDeleteMiddlewares)
		middlewares.POST("/:id/test",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermConnect),
			s.middlewareService.TestMiddlewareConnection)
		middlewares.POST("/:id/execute",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ExecuteMiddleware)
		middlewares.GET("/:id/databases",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListDatabases)
		middlewares.POST("/:id/databases",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.CreateDatabase)
		middlewares.GET("/:id/tables",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListTables)
		middlewares.GET("/:id/columns",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListColumns)

		// Redis 专用路由
		middlewares.GET("/:id/redis/info",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetRedisInfo)
		middlewares.GET("/:id/redis/databases",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetRedisDatabases)
		middlewares.GET("/:id/redis/keys",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ScanRedisKeys)
		middlewares.GET("/:id/redis/key",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetRedisKeyDetail)
		middlewares.POST("/:id/redis/key",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.SetRedisKey)
		middlewares.POST("/:id/redis/key/action",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.RedisKeyAction)
		middlewares.DELETE("/:id/redis/key",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.DeleteRedisKeys)
		middlewares.PUT("/:id/redis/key/ttl",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.SetRedisKeyTTL)
		middlewares.PUT("/:id/redis/key/rename",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.RenameRedisKey)

		// ClickHouse 专用路由
		middlewares.GET("/:id/clickhouse/databases",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListClickHouseDatabases)
		middlewares.POST("/:id/clickhouse/databases",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.CreateClickHouseDatabase)
		middlewares.GET("/:id/clickhouse/tables",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListClickHouseTables)
		middlewares.GET("/:id/clickhouse/columns",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListClickHouseColumns)

		// MongoDB 专用路由
		middlewares.GET("/:id/mongo/databases",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListMongoDatabases)
		middlewares.GET("/:id/mongo/collections",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListMongoCollections)
		middlewares.POST("/:id/mongo/collections",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.CreateMongoCollection)
		middlewares.POST("/:id/mongo/query",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.QueryMongoDocuments)
		middlewares.POST("/:id/mongo/insert",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.MongoInsertDocument)
		middlewares.POST("/:id/mongo/update",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.MongoUpdateDocuments)
		middlewares.POST("/:id/mongo/delete",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.MongoDeleteDocuments)
		middlewares.GET("/:id/mongo/stats",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetMongoStats)
		middlewares.GET("/:id/mongo/collection-stats",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetMongoCollectionStats)

		// Kafka 专用路由
		middlewares.GET("/:id/kafka/brokers",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetKafkaBrokers)
		middlewares.GET("/:id/kafka/topics",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListKafkaTopics)
		middlewares.POST("/:id/kafka/topics",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.CreateKafkaTopic)
		middlewares.DELETE("/:id/kafka/topics",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.DeleteKafkaTopic)
		middlewares.GET("/:id/kafka/topic-detail",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetKafkaTopicDetail)
		middlewares.GET("/:id/kafka/topic-config",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetKafkaTopicConfig)
		middlewares.PUT("/:id/kafka/topic-config",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.UpdateKafkaTopicConfig)
		middlewares.GET("/:id/kafka/consumer-groups",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListKafkaConsumerGroups)
		middlewares.GET("/:id/kafka/consumer-group-detail",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetKafkaConsumerGroupDetail)
		middlewares.DELETE("/:id/kafka/consumer-groups",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.DeleteKafkaConsumerGroup)
		middlewares.POST("/:id/kafka/produce",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ProduceKafkaMessage)
		middlewares.POST("/:id/kafka/consumer-session/start",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.StartKafkaConsumerSession)
		middlewares.GET("/:id/kafka/consumer-session/poll",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.PollKafkaConsumerSession)
		middlewares.DELETE("/:id/kafka/consumer-session/stop",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.StopKafkaConsumerSession)

		// Milvus 专用路由
		middlewares.GET("/:id/milvus/databases",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListMilvusDatabases)
		middlewares.POST("/:id/milvus/databases",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.CreateMilvusDatabase)
		middlewares.DELETE("/:id/milvus/databases",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.DropMilvusDatabase)
		middlewares.GET("/:id/milvus/collections",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListMilvusCollections)
		middlewares.GET("/:id/milvus/collection-detail",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.DescribeMilvusCollection)
		middlewares.POST("/:id/milvus/collections",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.CreateMilvusCollection)
		middlewares.DELETE("/:id/milvus/collections",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.DropMilvusCollection)
		middlewares.POST("/:id/milvus/collection/load",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.LoadMilvusCollection)
		middlewares.POST("/:id/milvus/collection/release",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ReleaseMilvusCollection)
		middlewares.POST("/:id/milvus/index",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.CreateMilvusIndex)
		middlewares.DELETE("/:id/milvus/index",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.DropMilvusIndex)
		middlewares.POST("/:id/milvus/query",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.QueryMilvusData)
		middlewares.POST("/:id/milvus/insert",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.InsertMilvusData)
		middlewares.POST("/:id/milvus/delete",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.DeleteMilvusData)
		middlewares.POST("/:id/milvus/search",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.SearchMilvusVectors)
		middlewares.GET("/:id/milvus/partitions",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.ListMilvusPartitions)
		middlewares.POST("/:id/milvus/partitions",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.CreateMilvusPartition)
		middlewares.DELETE("/:id/milvus/partitions",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.DropMilvusPartition)
		middlewares.GET("/:id/milvus/metrics",
			s.authMiddleware.RequireMiddlewarePermission(rbacbiz.MWPermExecute),
			s.middlewareService.GetMilvusMetrics)
	}

	// 中间件权限管理
	mwPerms := r.Group("/middleware-permissions")
	{
		mwPerms.GET("", s.middlewarePermissionService.ListMiddlewarePermissions)
		mwPerms.POST("", s.middlewarePermissionService.CreateMiddlewarePermission)
		mwPerms.PUT("/:id", s.middlewarePermissionService.UpdateMiddlewarePermission)
		mwPerms.DELETE("/:id", s.middlewarePermissionService.DeleteMiddlewarePermission)
	}
}

// NewAssetServices 创建asset相关的服务
func NewAssetServices(db *gorm.DB) (
	*assetService.AssetGroupService,
	*assetService.HostService,
	*assetService.MiddlewareService,
	*rbacService.MiddlewarePermissionService,
	*TerminalManager,
) {
	// 初始化Repository
	assetGroupRepo := assetdata.NewAssetGroupRepo(db)
	hostRepo := assetdata.NewHostRepo(db)
	credentialRepo := assetdata.NewCredentialRepo(db)
	cloudAccountRepo := assetdata.NewCloudAccountRepo(db)
	assetPermissionRepo := rbacdata.NewAssetPermissionRepo(db)
	middlewareRepo := assetdata.NewMiddlewareRepo(db)
	mwPermissionRepo := rbacdata.NewMiddlewarePermissionRepo(db)

	// 初始化UseCase
	assetGroupUseCase := assetbiz.NewAssetGroupUseCase(assetGroupRepo)
	credentialUseCase := assetbiz.NewCredentialUseCase(credentialRepo, hostRepo)
	cloudAccountUseCase := assetbiz.NewCloudAccountUseCase(cloudAccountRepo)
	hostUseCase := assetbiz.NewHostUseCase(hostRepo, credentialRepo, assetGroupRepo, cloudAccountRepo)
	assetPermissionUseCase := rbacbiz.NewAssetPermissionUseCase(assetPermissionRepo)
	middlewareUseCase := assetbiz.NewMiddlewareUseCase(middlewareRepo, assetGroupRepo, hostRepo)
	mwPermissionUseCase := rbacbiz.NewMiddlewarePermissionUseCase(mwPermissionRepo)

	// 初始化Service
	assetGroupService := assetService.NewAssetGroupService(assetGroupUseCase)
	hostService := assetService.NewHostService(hostUseCase, credentialUseCase, cloudAccountUseCase, assetPermissionUseCase)
	middlewareService := assetService.NewMiddlewareService(middlewareUseCase, mwPermissionUseCase, db)
	mwPermissionService := rbacService.NewMiddlewarePermissionService(mwPermissionUseCase)

	// 初始化TerminalManager
	terminalManager := NewTerminalManager(hostUseCase, db)

	return assetGroupService, hostService, middlewareService, mwPermissionService, terminalManager
}
