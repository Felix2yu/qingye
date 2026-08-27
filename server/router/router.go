package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"qingye/server/config"
	"qingye/server/handlers"

	"github.com/gin-gonic/gin"
)

// Setup 注册路由与中间件（CORS、日志、Recovery 由 gin.Default 提供）
func Setup(r *gin.Engine, cfg *config.Config) {
	r.Use(corsMiddleware(cfg.CORSOrigins))
	// 照片静态目录
	r.Static("/uploads", cfg.UploadDir)
	// 健康检查
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")

	// 植物与房间
	plantH := handlers.NewPlantHandler()
	api.GET("/rooms", plantH.ListRooms)
	api.POST("/rooms", plantH.CreateRoom)
	api.PUT("/rooms/:id", plantH.UpdateRoom)
	api.DELETE("/rooms/:id", plantH.DeleteRoom)
	api.GET("/plants", plantH.ListPlants)
	api.POST("/plants", plantH.CreatePlant)
	api.GET("/plants/:id", plantH.GetPlant)
	api.PUT("/plants/:id", plantH.UpdatePlant)
	api.DELETE("/plants/:id", plantH.DeletePlant)

	// 任务
	taskH := handlers.NewTaskHandler()
	api.GET("/tasks", taskH.List)
	api.GET("/tasks/today", taskH.Today)
	api.GET("/tasks/upcoming", taskH.Upcoming)
	api.POST("/tasks", taskH.Create)
	api.POST("/tasks/:id/done", taskH.Done)
	api.POST("/tasks/:id/postpone", taskH.Postpone)
	api.GET("/tasks/:id/logs", taskH.Logs)
	api.DELETE("/tasks/:id", taskH.Delete)

	// 统一养护时间线（聚合 task_logs 与人工养护）
	careH := handlers.NewCareLogHandler()
	api.GET("/care-logs", careH.List)
	api.POST("/care-logs", careH.Create)

	// 日记
	diaryH := handlers.NewDiaryHandler(cfg.UploadDir)
	api.GET("/diaries", diaryH.List)
	api.POST("/diaries", diaryH.Create)
	api.DELETE("/diaries/:id", diaryH.Delete)

	// 资料库
	libH := handlers.NewLibraryHandler(cfg)
	api.GET("/library", libH.Search)
	api.GET("/library/online", libH.SearchOnline)
	api.POST("/library/import", libH.ImportOnline)
	api.POST("/library/sync-popular", libH.SyncPopular)

	// 设置
	settingH := handlers.NewSettingHandler()
	api.GET("/settings", settingH.Get)
	api.PUT("/settings", settingH.Update)

	// 通知（shoutrrr）
	notifyH := handlers.NewNotifyHandler()
	api.PUT("/settings/notify", notifyH.SaveNotify)
	api.PUT("/settings/digest-hour", notifyH.SaveDigestHour)
	api.POST("/notify/test", notifyH.Test)

	// 天气与智能养护
	weatherH := handlers.NewWeatherHandler()
	api.GET("/weather/current", weatherH.Current)
	api.GET("/weather/config", weatherH.GetConfig)
	api.PUT("/weather/config", weatherH.SaveConfig)
	api.GET("/weather/logs", weatherH.Logs)
	api.POST("/weather/refresh", weatherH.Refresh)

	// 批量导入（先预览后确认）
	importH := handlers.NewImportHandler()
	api.POST("/import/preview", importH.Preview)
	api.POST("/import/confirm", importH.Confirm)
	api.POST("/import/template-preview", importH.TemplatePreview)

	// 前端静态站点（SPA）：用 WEB_DIR 指向构建产物目录；未配置则不托管页面
	serveWeb(r, cfg.WebDir)
}

// serveWeb 将前端静态构建目录挂到根路径，并对非 /api、非 /uploads 的未知路由
// 回退到 index.html（SPA 前端路由）。WebDir 为空时跳过。
func serveWeb(r *gin.Engine, webDir string) {
	if strings.TrimSpace(webDir) == "" {
		return
	}
	index := filepath.Join(webDir, "index.html")
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api") || strings.HasPrefix(p, "/uploads") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// 尝试直接返回静态文件（如 /_app/xxx.js、/favicon.svg）
		file := filepath.Join(webDir, filepath.Clean(p))
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			c.File(file)
			return
		}
		// 其余回退到 SPA 入口
		c.File(index)
	})
}

// corsMiddleware 限定允许的开发期前端源；origin 不在白名单时仍放行但不带 CORS 头
func corsMiddleware(origins []string) gin.HandlerFunc {
	allow := make(map[string]bool, len(origins))
	allowAll := false
	for _, o := range origins {
		if o == "*" {
			allowAll = true
		}
		allow[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if allowAll {
				c.Header("Access-Control-Allow-Origin", "*")
			} else if allow[origin] {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
