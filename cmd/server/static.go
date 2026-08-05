package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"outlook-manager/web"
)

// mountStatic 挂载内嵌前端（SPA：非 API 的 GET 一律回退 index.html）。
func mountStatic(r *gin.Engine, log *slog.Logger) {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		log.Warn("前端产物不可用，仅提供 API", "err", err)
		return
	}
	fileServer := http.FileServer(http.FS(dist))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// API 路径不回退
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在"})
			return
		}
		// 静态资源存在则直接服务
		if path != "/" {
			if f, err := dist.Open(strings.TrimPrefix(path, "/")); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		// SPA 回退
		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			c.String(http.StatusServiceUnavailable, "前端未构建：请先在 web/ 执行 npm run build")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})
}
