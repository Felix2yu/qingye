package handlers

import (
	"strconv"
	"time"

	"qingye/server/services"

	"github.com/gin-gonic/gin"
)

// CareLogHandler 统一养护时间线
type CareLogHandler struct {
	listSvc  *services.CareLogService
	taskSvc  *services.TaskService
}

func NewCareLogHandler() *CareLogHandler {
	return &CareLogHandler{
		listSvc: services.NewCareLogService(),
		taskSvc: services.NewTaskService(),
	}
}

// List GET /api/care-logs?plantId=&limit=
func (h *CareLogHandler) List(c *gin.Context) {
	plantID, _ := parseUintQuery(c, "plantId")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	var (
		logs any
		err  error
	)
	if plantID > 0 {
		logs, err = h.listSvc.ListByPlant(plantID)
	} else {
		logs, err = h.listSvc.List(limit)
	}
	if err != nil {
		ServerError(c, "获取养护记录失败")
		return
	}
	OK(c, logs)
}

// Create POST /api/care-logs  记录一次人工养护（无任务来源）
func (h *CareLogHandler) Create(c *gin.Context) {
	var body struct {
		PlantID uint   `json:"plantId"`
		Type    string `json:"type"`
		Title   string `json:"title"`
		Note    string `json:"note"`
		At      string `json:"at"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	var at time.Time
	if body.At != "" {
		if t, perr := time.Parse(time.RFC3339, body.At); perr == nil {
			at = t
		}
	}
	log, err := h.taskSvc.RecordManual(body.PlantID, body.Type, body.Title, body.Note, at)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, log)
}
