package handlers

import (
	"qingye/server/services"

	"github.com/gin-gonic/gin"
)

// SettingHandler 设置中心
type SettingHandler struct{ svc *services.SettingService }

func NewSettingHandler() *SettingHandler {
	return &SettingHandler{svc: services.NewSettingService()}
}

// Get GET /api/settings
func (h *SettingHandler) Get(c *gin.Context) {
	st, err := h.svc.Get()
	if err != nil {
		ServerError(c, "获取设置失败")
		return
	}
	OK(c, st)
}

// Update PUT /api/settings
func (h *SettingHandler) Update(c *gin.Context) {
	var body struct {
		Workdays []int          `json:"workdays"`
		Prefs    map[string]any `json:"prefs"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	if body.Workdays == nil {
		BadRequest(c, "缺少工作日设置")
		return
	}
	st, err := h.svc.Update(body.Workdays, body.Prefs)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, st)
}
