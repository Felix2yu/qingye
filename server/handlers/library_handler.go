package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"qingye/server/config"
	"qingye/server/services"

	"github.com/gin-gonic/gin"
)

// LibraryHandler 资料库接口
type LibraryHandler struct {
	svc *services.LibraryService
}

func NewLibraryHandler(cfg *config.Config) *LibraryHandler {
	return &LibraryHandler{svc: services.NewLibraryService(cfg)}
}

// Search 本地资料库搜索（添加植物时带入指南）
func (h *LibraryHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	list, err := h.svc.Search(keyword)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, list)
}

// SearchOnline 在线搜索候选（Plantbook）；未配置凭据时返回 enabled=false
func (h *LibraryHandler) SearchOnline(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	cands, err := h.svc.SearchOnline(keyword)
	if err != nil {
		Fail(c, http.StatusBadGateway, http.StatusBadGateway, err.Error())
		return
	}
	OK(c, gin.H{
		"enabled": h.svc.OnlineEnabled(),
		"list":    cands,
	})
}

// ImportOnline 按 pid 拉取详情并写回本地资料库
func (h *LibraryHandler) ImportOnline(c *gin.Context) {
	var body struct {
		PID string `json:"pid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.PID == "" {
		BadRequest(c, "缺少 pid")
		return
	}
	lib, err := h.svc.ImportOnline(body.PID)
	if err != nil {
		Fail(c, http.StatusBadGateway, http.StatusBadGateway, err.Error())
		return
	}
	if lib == nil {
		NotFound(c, "未找到该植物")
		return
	}
	OK(c, lib)
}

// SyncPopular 批量同步内置热门植物到本地资料库（离线可用）
// ?limit=N 每轮最多检索的新条目数（默认 30）
func (h *LibraryHandler) SyncPopular(c *gin.Context) {
	if !h.svc.OnlineEnabled() {
		BadRequest(c, "未配置 Plantbook 凭据（PLANTBOOK_CLIENT_ID / PLANTBOOK_CLIENT_SECRET），无法在线同步")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	if limit <= 0 || limit > 200 {
		limit = 30
	}
	added, failed, skipped, remaining, firstErr, throttled := h.svc.SyncPopular(limit)

	msg := fmt.Sprintf("本轮新增 %d 种，失败 %d 种", added, failed)
	if skipped > 0 {
		msg += fmt.Sprintf("，跳过 %d 种（本地已存在）", skipped)
	}
	switch {
	case throttled != nil:
		msg += fmt.Sprintf("；%s。稍后再次同步可从断点继续", throttled.Error())
	case firstErr != "":
		msg += fmt.Sprintf("；已停止：%s", firstErr)
	case remaining > 0:
		msg += fmt.Sprintf("。还有 %d 种待同步，稍后再次点击继续", remaining)
	default:
		msg += "。全部条目已处理完毕 🌿"
	}
	OK(c, gin.H{
		"added":     added,
		"failed":    failed,
		"skipped":   skipped,
		"remaining": remaining,
		"total":     added + failed + skipped + remaining,
		"throttled": throttled != nil,
		"message":   msg,
	})
}
