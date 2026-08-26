package handlers

import (
	"qingye/server/models"
	"qingye/server/services"

	"github.com/gin-gonic/gin"
)

// ImportHandler 批量导入
type ImportHandler struct {
	importSvc *services.ImportService
}

// NewImportHandler 构造
func NewImportHandler() *ImportHandler {
	return &ImportHandler{importSvc: services.NewImportService()}
}

// Preview 解析 CSV 并返回预览（支持 plants / tasks）
// POST /api/import/preview  multipart: file + form: kind
func (h *ImportHandler) Preview(c *gin.Context) {
	kind := c.PostForm("kind")
	if kind != "plants" && kind != "tasks" {
		BadRequest(c, "kind 仅支持 plants 或 tasks")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		BadRequest(c, "请上传 CSV 文件")
		return
	}
	f, err := file.Open()
	if err != nil {
		BadRequest(c, "无法读取文件")
		return
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	if _, err := f.Read(buf); err != nil {
		BadRequest(c, "读取文件失败")
		return
	}
	content := string(buf)

	var preview *models.ImportPreview
	if kind == "plants" {
		preview, err = h.importSvc.PreviewPlants(content)
	} else {
		preview, err = h.importSvc.PreviewTasks(content)
	}
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, preview)
}

// Confirm 确认并落库
// POST /api/import/confirm  JSON
func (h *ImportHandler) Confirm(c *gin.Context) {
	var req models.ImportConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}

	var (
		preview *models.ImportPreview
		err     error
		result  *models.ImportResult
	)

	switch req.Kind {
	case "plants":
		preview, err = h.importSvc.PreviewPlants(req.Content)
		if err != nil {
			BadRequest(c, err.Error())
			return
		}
		result, err = h.importSvc.ConfirmPlants(preview, req.Accepted)
	case "tasks":
		preview, err = h.importSvc.PreviewTasks(req.Content)
		if err != nil {
			BadRequest(c, err.Error())
			return
		}
		result, err = h.importSvc.ConfirmTasks(preview, req.Accepted)
	case "template":
		result, err = h.importSvc.ConfirmTemplate(req.SourceID, req.TargetIDs)
	default:
		BadRequest(c, "不支持的 kind")
		return
	}
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, result)
}

// TemplatePreview 预览模板复制
// POST /api/import/template-preview  JSON: {sourceId, targetIds}
func (h *ImportHandler) TemplatePreview(c *gin.Context) {
	var req struct {
		SourceID  uint   `json:"sourceId"`
		TargetIDs []uint `json:"targetIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	preview, err := h.importSvc.PreviewTemplate(req.SourceID, req.TargetIDs)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, preview)
}
