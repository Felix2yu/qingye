package handlers

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"

	"qingye/server/models"
	"qingye/server/services"

	"github.com/gin-gonic/gin"
)

// DiaryHandler 照片日记
type DiaryHandler struct {
	svc       *services.DiaryService
	uploadDir string
}

func NewDiaryHandler(uploadDir string) *DiaryHandler {
	return &DiaryHandler{svc: services.NewDiaryService(), uploadDir: uploadDir}
}

// List GET /api/diaries?plantId=&page=&pageSize=
func (h *DiaryHandler) List(c *gin.Context) {
	plantID, _ := parseUintQuery(c, "plantId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.svc.Page(plantID, page, pageSize)
	if err != nil {
		ServerError(c, "获取日记失败")
		return
	}
	OK(c, gin.H{"list": list, "total": total, "page": page, "pageSize": pageSize})
}

// Create POST /api/diaries  multipart/form-data: plantId, image, caption, takenAt
func (h *DiaryHandler) Create(c *gin.Context) {
	plantID, err := strconv.ParseUint(c.PostForm("plantId"), 10, 64)
	if err != nil || plantID == 0 {
		BadRequest(c, "缺少植物")
		return
	}
	file, err := c.FormFile("image")
	if err != nil {
		BadRequest(c, "缺少图片")
		return
	}
	// 生成唯一文件名并保存
	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomString(8), filepath.Ext(file.Filename))
	dst := filepath.Join(h.uploadDir, name)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		ServerError(c, "图片保存失败")
		return
	}
	d, err := h.svc.Create(&models.PhotoDiary{
		PlantID: uint(plantID),
		Image:   "/uploads/" + name,
		Caption: c.PostForm("caption"),
		TakenAt: parseFlexTime(c.PostForm("takenAt")),
	})
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, d)
}

// Delete DELETE /api/diaries/:id
func (h *DiaryHandler) Delete(c *gin.Context) {
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

// parseFlexTime 解析可选的 RFC3339 / 日期时间字符串；失败返回零值
func parseFlexTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return time.Time{}
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
