package handlers

import (
	"qingye/server/services"

	"github.com/gin-gonic/gin"
)

// NotifyHandler 通知（shoutrrr）配置与测试
type NotifyHandler struct{ svc *services.SettingService }

func NewNotifyHandler() *NotifyHandler {
	return &NotifyHandler{svc: services.NewSettingService()}
}

// SaveNotify PUT /api/settings/notify
// body: { "url": "discord://xxx" }  空字符串表示关闭通知
func (h *NotifyHandler) SaveNotify(c *gin.Context) {
	var body struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	st, err := h.svc.SaveNotifyURL(body.URL)
	if err != nil {
		ServerError(c, "保存通知配置失败")
		return
	}
	OK(c, st)
}

// Test POST /api/notify/test  向已配置的地址发送一条测试通知
func (h *NotifyHandler) Test(c *gin.Context) {
	msg, err := services.NewNotifyService().Test()
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, gin.H{"message": msg})
}

// SaveDigestHour PUT /api/settings/digest-hour
// body: { "hour": 8 }  每日养护摘要的推送小时（0-23）
func (h *NotifyHandler) SaveDigestHour(c *gin.Context) {
	var body struct {
		Hour int `json:"hour"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	st, err := h.svc.SaveDigestHour(body.Hour)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	// 立即按新时间重排每日摘要调度
	services.RescheduleNotifier()
	OK(c, st)
}
