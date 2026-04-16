package alert

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	alertbiz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
	alertsvc "github.com/ydcloud-dy/opshub/internal/service/alert"
	agentserver "github.com/ydcloud-dy/opshub/internal/server/agent"
)

// HTTPServer 告警管理 HTTP 服务
type HTTPServer struct {
	dsRepo                *alertdata.DataSourceRepo
	dsAgentRelationRepo   alertbiz.DataSourceAgentRelationRepo
	ruleGroupRepo         *alertdata.RuleGroupRepo
	ruleRepo              *alertdata.RuleRepo
	eventRepo             *alertdata.EventRepo
	channelRepo           *alertdata.ChannelRepo
	subRepo               *alertdata.SubscriptionRepo
	subRuleRepo           *alertdata.SubscriptionRuleRepo
	subChannelRepo        *alertdata.SubscriptionChannelRepo
	subUserRepo           *alertdata.SubscriptionUserRepo
	silenceRuleRepo       *alertdata.SilenceRuleRepo
	notifySvc             *alertsvc.NotifyService
	evalEngine            *alertsvc.EvalEngine
	agentHub              *agentserver.AgentHub
}

// NewAlertServices 工厂函数，组装所有依赖
func NewAlertServices(db *gorm.DB, rdb *redis.Client) *HTTPServer {
	dsRepo := alertdata.NewDataSourceRepo(db)
	dsAgentRelationRepo := alertdata.NewDataSourceAgentRelationRepo(db)
	ruleGroupRepo := alertdata.NewRuleGroupRepo(db)
	ruleRepo := alertdata.NewRuleRepo(db)
	eventRepo := alertdata.NewEventRepo(db)
	channelRepo := alertdata.NewChannelRepo(db)
	subRepo := alertdata.NewSubscriptionRepo(db)
	subRuleRepo := alertdata.NewSubscriptionRuleRepo(db)
	subChannelRepo := alertdata.NewSubscriptionChannelRepo(db)
	subUserRepo := alertdata.NewSubscriptionUserRepo(db)
	silenceRuleRepo := alertdata.NewSilenceRuleRepo(db)
	notifySvc := alertsvc.NewNotifyService(channelRepo)
	evalEngine := alertsvc.NewEvalEngine(db, rdb)

	return &HTTPServer{
		dsRepo:              dsRepo,
		dsAgentRelationRepo: dsAgentRelationRepo,
		ruleGroupRepo:       ruleGroupRepo,
		ruleRepo:            ruleRepo,
		eventRepo:           eventRepo,
		channelRepo:         channelRepo,
		subRepo:             subRepo,
		subRuleRepo:         subRuleRepo,
		subChannelRepo:      subChannelRepo,
		subUserRepo:         subUserRepo,
		silenceRuleRepo:     silenceRuleRepo,
		notifySvc:           notifySvc,
		evalEngine:          evalEngine,
	}
}

// SetAgentHub 注入AgentHub
func (s *HTTPServer) SetAgentHub(agentHub *agentserver.AgentHub) {
	s.agentHub = agentHub
}

// GetEvalEngine 返回评估引擎（供 server.go 启动）
func (s *HTTPServer) GetEvalEngine() *alertsvc.EvalEngine {
	return s.evalEngine
}

// RegisterRoutes 注册路由
func (s *HTTPServer) RegisterRoutes(rg *gin.RouterGroup) {
	alert := rg.Group("/alert")

	// 数据源
	ds := alert.Group("/datasources")
	{
		ds.GET("", s.listDataSources)
		ds.POST("", s.createDataSource)
		ds.GET("/:id", s.getDataSource)
		ds.PUT("/:id", s.updateDataSource)
		ds.DELETE("/:id", s.deleteDataSource)
		ds.POST("/:id/test", s.testDataSource)
		// 数据源Agent关联
		agentRels := ds.Group("/:id/agent-relations")
		{
			agentRels.GET("", s.listAgentRelations)
			agentRels.POST("", s.createAgentRelation)
		}
		ds.DELETE("/agent-relations/:id", s.deleteAgentRelation)
	}

	// 规则分类
	rg2 := alert.Group("/rule-groups")
	{
		rg2.GET("", s.listRuleGroups)
		rg2.POST("", s.createRuleGroup)
		rg2.GET("/:id", s.getRuleGroup)
		rg2.PUT("/:id", s.updateRuleGroup)
		rg2.DELETE("/:id", s.deleteRuleGroup)
	}

	// 告警规则
	rules := alert.Group("/rules")
	{
		rules.GET("", s.listRules)
		rules.POST("", s.createRule)
		rules.GET("/:id", s.getRule)
		rules.PUT("/:id", s.updateRule)
		rules.DELETE("/:id", s.deleteRule)
		rules.PUT("/:id/toggle", s.toggleRule)
		rules.POST("/:id/test", s.testRule)
		rules.POST("/:id/clone", s.cloneRule)
		rules.POST("/import", s.importRules)
		rules.GET("/export", s.exportRules)
		rules.POST("/adhoc-test", s.adhocTestRule)
	}

	// 告警事件
	events := alert.Group("/events")
	{
		events.GET("/active", s.listActiveEvents)
		events.GET("/history", s.listHistoryEvents)
		events.POST("/:id/silence", s.silenceEvent)
		events.POST("/:id/handle", s.handleEvent)
		events.GET("/stats", s.getEventStats)
		events.GET("/trend", s.getEventTrend)
		events.POST("/batch-silence", s.batchSilenceEvents)
		events.POST("/batch-unsilence", s.batchUnsilenceEvents)
		events.GET("/silenced", s.listSilencedEvents)
	}

	// 通知通道
	channels := alert.Group("/channels")
	{
		channels.GET("", s.listChannels)
		channels.POST("", s.createChannel)
		channels.GET("/:id", s.getChannel)
		channels.PUT("/:id", s.updateChannel)
		channels.DELETE("/:id", s.deleteChannel)
		channels.POST("/:id/test", s.testChannel)
	}

	// 订阅任务
	subs := alert.Group("/subscriptions")
	{
		subs.GET("", s.listSubscriptions)
		subs.POST("", s.createSubscription)
		subs.GET("/:id", s.getSubscription)
		subs.PUT("/:id", s.updateSubscription)
		subs.DELETE("/:id", s.deleteSubscription)
	}

	// 屏蔽规则
	silenceRules := alert.Group("/silence-rules")
	{
		silenceRules.GET("", s.listSilenceRules)
		silenceRules.POST("", s.createSilenceRule)
		silenceRules.GET("/:id", s.getSilenceRule)
		silenceRules.PUT("/:id", s.updateSilenceRule)
		silenceRules.DELETE("/:id", s.deleteSilenceRule)
		silenceRules.PUT("/:id/toggle", s.toggleSilenceRule)
	}
}

// RegisterPublicRoutes 注册公开路由（无需认证）
func (s *HTTPServer) RegisterPublicRoutes(router *gin.Engine) {
	// 数据源代理路由（无需认证，通过 ProxyToken 验证）
	proxy := router.Group("/api/v1/alert/proxy/datasource")
	{
		proxy.Any("/:token/*path", s.proxyDataSourceRequest)
	}
}
