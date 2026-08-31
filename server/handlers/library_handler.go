package handlers

import (
	"encoding/json"
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
	// 兼容前端 ?q= 与历史 ?keyword= 两种参数名
	keyword := c.Query("q")
	if keyword == "" {
		keyword = c.Query("keyword")
	}
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
// ?limit=N 每轮最多检索的新条目数（默认 30）。以 SSE 实时推送每条进度，
// 结束时推送 done 事件（SyncReport）。无 Flusher 环境（如单测）降级为 JSON。
func (h *LibraryHandler) SyncPopular(c *gin.Context) {
	if !h.svc.OnlineEnabled() {
		BadRequest(c, "未配置 Plantbook 凭据（PLANTBOOK_CLIENT_ID / PLANTBOOK_CLIENT_SECRET），无法在线同步")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	if limit <= 0 || limit > 200 {
		limit = 30
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// 非流式环境（如单测）：直接返回汇总 JSON
		rep := h.svc.SyncPopularStream(c.Request.Context(), limit, nil)
		OK(c, rep)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	rep := h.svc.SyncPopularStream(c.Request.Context(), limit, func(p services.SyncProgress) {
		data, err := json.Marshal(p)
		if err != nil {
			return
		}
		fmt.Fprintf(c.Writer, "event: progress\ndata: %s\n\n", data)
		flusher.Flush()
	})

	data, err := json.Marshal(rep)
	if err == nil {
		fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", data)
		flusher.Flush()
	}
}

// RefreshGuide 本地刷新：将所有英文养护指南翻译为中文（不调用外部API）
func (h *LibraryHandler) RefreshGuide(c *gin.Context) {
	count, err := h.svc.RefreshLocalGuides()
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"refreshed": count})
}

// ClearLibrary 清空资料库所有条目
func (h *LibraryHandler) ClearLibrary(c *gin.Context) {
	if err := h.svc.ClearLibrary(); err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"message": "资料库已清空"})
}

// ResyncAndTranslate 重新拉取所有植物的英文Guide并翻译为中文（消耗API配额）
func (h *LibraryHandler) ResyncAndTranslate(c *gin.Context) {
	if !h.svc.OnlineEnabled() {
		BadRequest(c, "未配置 Plantbook 凭据，无法在线重新拉取")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		rep := h.svc.ResyncAndTranslate(c.Request.Context(), limit, nil)
		OK(c, rep)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	rep := h.svc.ResyncAndTranslate(c.Request.Context(), limit, func(p services.ResyncAndTranslateProgress) {
		data, err := json.Marshal(p)
		if err != nil {
			return
		}
		fmt.Fprintf(c.Writer, "event: progress\ndata: %s\n\n", data)
		flusher.Flush()
	})

	data, err := json.Marshal(rep)
	if err == nil {
		fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", data)
		flusher.Flush()
	}
}
