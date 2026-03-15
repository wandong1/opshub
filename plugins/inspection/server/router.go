package server

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	inspection := r.Group("/inspection")
	{
		// 统计数据
		inspection.GET("/stats", handler.GetStats)

		// 巡检组
		groups := inspection.Group("/groups")
		{
			groups.POST("", handler.CreateGroup)
			groups.PUT("/:id", handler.UpdateGroup)
			groups.DELETE("/:id", handler.DeleteGroup)
			groups.GET("/:id", handler.GetGroup)
			groups.GET("", handler.ListGroups)
			groups.GET("/all", handler.GetAllGroups)
			groups.POST("/:id/items", handler.BatchSaveItems)
		}

		// 巡检项
		items := inspection.Group("/items")
		{
			items.POST("", handler.CreateItem)
			items.PUT("/:id", handler.UpdateItem)
			items.DELETE("/:id", handler.DeleteItem)
			items.GET("/:id", handler.GetItem)
			items.GET("", handler.ListItems)
			items.POST("/test-run", handler.TestRunItems)
		}

		// 定时任务
		tasks := inspection.Group("/tasks")
		{
			tasks.POST("", handler.CreateTask)
			tasks.PUT("/:id", handler.UpdateTask)
			tasks.DELETE("/:id", handler.DeleteTask)
			tasks.GET("/:id", handler.GetTask)
			tasks.GET("", handler.ListTasks)
		}

		// 执行记录
		records := inspection.Group("/records")
		{
			records.GET("/:id", handler.GetRecord)
			records.GET("", handler.ListRecords)
		}
	}
}
