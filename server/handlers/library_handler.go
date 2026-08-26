package handlers

import (
	"fmt"
	"net/http"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list})
}

// SearchOnline 在线搜索候选（Plantbook）；未配置 token 时返回 enabled=false
func (h *LibraryHandler) SearchOnline(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	cands, err := h.svc.SearchOnline(keyword)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 pid"})
		return
	}
	lib, err := h.svc.ImportOnline(body.PID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if lib == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该植物"})
		return
	}
	c.JSON(http.StatusOK, lib)
}

// SyncPopular 批量同步内置热门植物到本地资料库（离线可用）
func (h *LibraryHandler) SyncPopular(c *gin.Context) {
	if !h.svc.OnlineEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置 Plantbook token，无法在线同步"})
		return
	}
	added, failed, err := h.svc.SyncPopular()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"added":  added,
		"failed": failed,
		"total":  added + failed,
		"message": fmt.Sprintf("已同步 %d 种，失败 %d 种", added, failed),
	})
}
