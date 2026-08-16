// Package router 注册全部 HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"

	"outlook-manager/internal/handler"
	"outlook-manager/internal/middleware"
	"outlook-manager/internal/service"
)

// New 构建 Gin 引擎与 API 路由。
// staticFS 非空时挂载前端静态资源（生产单二进制模式）。
func New(h *handler.Handler, svc *service.Service, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS())

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", h.Login)

		// 自动化对接（API Key 认证）
		ingest := api.Group("/ingest", middleware.APIKeyAuth(svc.APIKeys))
		ingest.POST("/accounts", h.IngestAccounts)

		// 管理端（JWT 认证）
		authed := api.Group("", middleware.JWTAuth(jwtSecret))
		{
			authed.GET("/auth/profile", h.Profile)
			authed.POST("/auth/change-password", h.ChangePassword)

			authed.GET("/accounts", h.ListAccounts)
			authed.POST("/accounts", h.CreateAccount)
			authed.GET("/accounts/groups", h.ListGroups)
			authed.GET("/accounts/export", h.ExportAccounts)
			authed.POST("/accounts/import", h.ImportAccounts)
			authed.POST("/accounts/batch-delete", h.BatchDeleteAccounts)
			authed.POST("/accounts/delete-by-status", h.DeleteAccountsByStatus)
			authed.POST("/accounts/refresh-all", h.RefreshAllAccounts)
			authed.POST("/accounts/check-all", h.CheckAllAccounts)
			authed.POST("/accounts/batch-check", h.BatchCheckAccounts)
			authed.POST("/accounts/batch-refresh", h.BatchRefreshAccounts)
			authed.GET("/accounts/:id", h.GetAccount)
			authed.PUT("/accounts/:id", h.UpdateAccount)
			authed.DELETE("/accounts/:id", h.DeleteAccount)
			authed.POST("/accounts/:id/refresh", h.RefreshAccount)
			authed.POST("/accounts/:id/check", h.CheckAccount)

			authed.GET("/accounts/:id/mails", h.ListMails)
			authed.POST("/accounts/:id/mails/fetch", h.FetchMails)
			authed.GET("/mails/:id", h.GetMail)

			authed.GET("/tasks/logs", h.ListTaskLogs)
			authed.POST("/tasks/logs/cleanup", h.CleanupTaskLogs)
			authed.GET("/tasks/schedule", h.GetSchedule)
			authed.PUT("/tasks/schedule", h.SetSchedule)

			authed.GET("/stats/dashboard", h.Dashboard)

			authed.GET("/apikeys", h.ListAPIKeys)
			authed.POST("/apikeys", h.CreateAPIKey)
			authed.DELETE("/apikeys/:id", h.DeleteAPIKey)

			authed.POST("/settings/import-from-register", h.ImportFromRegister)
		}
	}

	api.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	return r
}
