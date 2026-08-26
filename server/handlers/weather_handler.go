package handlers

import (
	"strconv"

	"qingye/server/models"
	"qingye/server/services"

	"github.com/gin-gonic/gin"
)

// WeatherHandler 天气与智能养护策略
type WeatherHandler struct {
	svc *services.WeatherService
}

func NewWeatherHandler() *WeatherHandler {
	return &WeatherHandler{svc: services.NewWeatherService()}
}

// Current GET /api/weather/current  当前天气 + 策略状态
func (h *WeatherHandler) Current(c *gin.Context) {
	cfg := h.svc.LoadConfig()
	now := h.svc.Current()
	OK(c, gin.H{
		"config":    cfg,
		"current":   now,
		"available": services.QWeatherKeyAvailable(),
	})
}

// Config GET /api/weather/config  读取策略配置
func (h *WeatherHandler) GetConfig(c *gin.Context) {
	OK(c, h.svc.LoadConfig())
}

// SaveConfig PUT /api/weather/config  保存策略配置
func (h *WeatherHandler) SaveConfig(c *gin.Context) {
	var cfg models.WeatherConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	if err := services.SaveWeatherConfig(cfg); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, h.svc.LoadConfig())
}

// Logs GET /api/weather/logs?limit=20  调整日志
func (h *WeatherHandler) Logs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	logs, err := h.svc.Logs(limit)
	if err != nil {
		ServerError(c, "获取天气日志失败")
		return
	}
	OK(c, logs)
}

// Refresh POST /api/weather/refresh  手动触发一次策略调整
func (h *WeatherHandler) Refresh(c *gin.Context) {
	cfg := h.svc.LoadConfig()
	if services.QWeatherKeyAvailable() == false {
		BadRequest(c, "未配置 QWEATHER_KEY，天气模块不可用")
		return
	}
	if !cfg.Enabled {
		BadRequest(c, "天气策略未启用")
		return
	}
	h.svc.Poll()
	OK(c, h.svc.Current())
}
