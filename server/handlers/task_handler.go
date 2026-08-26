package handlers

import (
	"strconv"

	"qingye/server/models"
	"qingye/server/services"

	"github.com/gin-gonic/gin"
)

// TaskHandler 任务清单
type TaskHandler struct{ svc *services.TaskService }

func NewTaskHandler() *TaskHandler {
	return &TaskHandler{svc: services.NewTaskService()}
}

// List GET /api/tasks?type=&includeDone=&plantId=
func (h *TaskHandler) List(c *gin.Context) {
	taskType := c.Query("type")
	includeDone := c.DefaultQuery("includeDone", "false") == "true"
	plantID, _ := parseUintQuery(c, "plantId")
	list, err := h.svc.List(taskType, includeDone, plantID)
	if err != nil {
		ServerError(c, "获取任务失败")
		return
	}
	OK(c, list)
}

// Today GET /api/tasks/today
func (h *TaskHandler) Today(c *gin.Context) {
	list, err := h.svc.Today()
	if err != nil {
		ServerError(c, "获取今日任务失败")
		return
	}
	OK(c, list)
}

// Upcoming GET /api/tasks/upcoming?days=3
func (h *TaskHandler) Upcoming(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "3"))
	list, err := h.svc.Upcoming(days)
	if err != nil {
		ServerError(c, "获取临近任务失败")
		return
	}
	OK(c, list)
}

// Create POST /api/tasks
func (h *TaskHandler) Create(c *gin.Context) {
	var body models.Task
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	task, err := h.svc.Create(&body)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, task)
}

// Done POST /api/tasks/:id/done
func (h *TaskHandler) Done(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)
	task, err := h.svc.Done(id, body.Note)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, task)
}

// Postpone POST /api/tasks/:id/postpone
func (h *TaskHandler) Postpone(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		Days int    `json:"days"`
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)
	task, err := h.svc.Postpone(id, body.Days, body.Note)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, task)
}

// Logs GET /api/tasks/:id/logs
func (h *TaskHandler) Logs(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	logs, err := h.svc.History(id)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, logs)
}

// Delete DELETE /api/tasks/:id
func (h *TaskHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, nil)
}

// parseUintQuery 解析可选的无符号整型查询参数
func parseUintQuery(c *gin.Context, key string) (uint, error) {
	v, err := strconv.ParseUint(c.DefaultQuery(key, "0"), 10, 64)
	return uint(v), err
}
