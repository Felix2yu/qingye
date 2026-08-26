package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"qingye/server/config"
	"qingye/server/models"
	"qingye/server/repositories"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTest(t *testing.T) *gin.Engine {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(
		&models.Plant{},
		&models.Room{},
		&models.Task{},
		&models.TaskLog{},
		&models.CareLog{},
		&models.PhotoDiary{},
		&models.PlantLibrary{},
		&models.UserSetting{},
		&models.WeatherLog{},
	)
	repositories.SetDB(db)

	r := gin.New()
	api := r.Group("/api")

	plantH := NewPlantHandler()
	api.GET("/rooms", plantH.ListRooms)
	api.POST("/rooms", plantH.CreateRoom)
	api.PUT("/rooms/:id", plantH.UpdateRoom)
	api.DELETE("/rooms/:id", plantH.DeleteRoom)
	api.GET("/plants", plantH.ListPlants)
	api.POST("/plants", plantH.CreatePlant)
	api.GET("/plants/:id", plantH.GetPlant)
	api.PUT("/plants/:id", plantH.UpdatePlant)
	api.DELETE("/plants/:id", plantH.DeletePlant)

	taskH := NewTaskHandler()
	api.GET("/tasks", taskH.List)
	api.POST("/tasks", taskH.Create)
	api.POST("/tasks/:id/done", taskH.Done)
	api.POST("/tasks/:id/postpone", taskH.Postpone)
	api.DELETE("/tasks/:id", taskH.Delete)

	careH := NewCareLogHandler()
	api.GET("/care-logs", careH.List)
	api.POST("/care-logs", careH.Create)

	diaryH := NewDiaryHandler(t.TempDir())
	api.GET("/diaries", diaryH.List)
	api.POST("/diaries", diaryH.Create)
	api.DELETE("/diaries/:id", diaryH.Delete)

	settingH := NewSettingHandler()
	api.GET("/settings", settingH.Get)
	api.PUT("/settings", settingH.Update)

	libH := NewLibraryHandler(&config.Config{})
	api.GET("/library", libH.Search)

	weatherH := NewWeatherHandler()
	api.GET("/weather/config", weatherH.GetConfig)
	api.PUT("/weather/config", weatherH.SaveConfig)

	importH := NewImportHandler()
	api.POST("/import/preview", importH.Preview)
	api.POST("/import/confirm", importH.Confirm)
	api.POST("/import/template-preview", importH.TemplatePreview)

	return r
}

func perform(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("json decode: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

// ---- Room CRUD ----

func TestIntegration_RoomCRUD(t *testing.T) {
	r := setupTest(t)

	// 创建
	w := perform(r, "POST", "/api/rooms", map[string]string{"name": "客厅"})
	if w.Code != 200 {
		t.Fatalf("create room: %d %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	data := resp["data"].(map[string]any)
	roomID := data["id"].(float64)
	if data["name"] != "客厅" {
		t.Errorf("name = %v", data["name"])
	}

	// 列表
	w = perform(r, "GET", "/api/rooms", nil)
	if w.Code != 200 {
		t.Fatalf("list rooms: %d", w.Code)
	}
	resp = decodeJSON(t, w)
	list := resp["data"].([]any)
	if len(list) != 1 {
		t.Fatalf("rooms count = %d, want 1", len(list))
	}

	// 更新
	w = perform(r, "PUT", fmt.Sprintf("/api/rooms/%d", int(roomID)), map[string]string{"name": "新客厅"})
	if w.Code != 200 {
		t.Fatalf("update room: %d", w.Code)
	}

	// 删除
	w = perform(r, "DELETE", fmt.Sprintf("/api/rooms/%d", int(roomID)), nil)
	if w.Code != 200 {
		t.Fatalf("delete room: %d", w.Code)
	}
}

// ---- Plant CRUD ----

func TestIntegration_PlantCRUD(t *testing.T) {
	r := setupTest(t)

	// 先创建房间
	w := perform(r, "POST", "/api/rooms", map[string]string{"name": "客厅"})
	resp := decodeJSON(t, w)
	roomID := int(resp["data"].(map[string]any)["id"].(float64))

	// 创建植物
	w = perform(r, "POST", "/api/plants", map[string]any{
		"name":   "绿萝",
		"roomId": roomID,
	})
	if w.Code != 200 {
		t.Fatalf("create plant: %d %s", w.Code, w.Body.String())
	}
	resp = decodeJSON(t, w)
	plantID := int(resp["data"].(map[string]any)["id"].(float64))

	// 获取
	w = perform(r, "GET", fmt.Sprintf("/api/plants/%d", plantID), nil)
	if w.Code != 200 {
		t.Fatalf("get plant: %d", w.Code)
	}
	resp = decodeJSON(t, w)
	p := resp["data"].(map[string]any)
	if p["name"] != "绿萝" {
		t.Errorf("name = %v", p["name"])
	}

	// 列表
	w = perform(r, "GET", "/api/plants", nil)
	if w.Code != 200 {
		t.Fatalf("list plants: %d", w.Code)
	}

	// 更新
	w = perform(r, "PUT", fmt.Sprintf("/api/plants/%d", plantID), map[string]any{
		"name":    "黄金葛",
		"species": "Epipremnum",
	})
	if w.Code != 200 {
		t.Fatalf("update plant: %d", w.Code)
	}

	// 删除
	w = perform(r, "DELETE", fmt.Sprintf("/api/plants/%d", plantID), nil)
	if w.Code != 200 {
		t.Fatalf("delete plant: %d", w.Code)
	}
}

// ---- Task CRUD ----

func TestIntegration_TaskCRUD(t *testing.T) {
	r := setupTest(t)

	// 先创建植物
	w := perform(r, "POST", "/api/plants", map[string]any{"name": "Test"})
	resp := decodeJSON(t, w)
	plantID := int(resp["data"].(map[string]any)["id"].(float64))

	// 创建任务
	w = perform(r, "POST", "/api/tasks", map[string]any{
		"plantId":      plantID,
		"type":         "water",
		"title":        "浇水",
		"intervalDays": 7,
		"nextDue":      time.Now().AddDate(0, 0, 7).Format(time.RFC3339),
	})
	if w.Code != 200 {
		t.Fatalf("create task: %d %s", w.Code, w.Body.String())
	}
	resp = decodeJSON(t, w)
	taskID := int(resp["data"].(map[string]any)["id"].(float64))

	// 列表
	w = perform(r, "GET", "/api/tasks", nil)
	if w.Code != 200 {
		t.Fatalf("list tasks: %d", w.Code)
	}

	// 完成
	w = perform(r, "POST", fmt.Sprintf("/api/tasks/%d/done", taskID), nil)
	if w.Code != 200 {
		t.Fatalf("done task: %d %s", w.Code, w.Body.String())
	}

	// 删除
	w = perform(r, "DELETE", fmt.Sprintf("/api/tasks/%d", taskID), nil)
	if w.Code != 200 {
		t.Fatalf("delete task: %d", w.Code)
	}
}

// ---- CareLog ----

func TestIntegration_CareLogCRUD(t *testing.T) {
	r := setupTest(t)

	// 创建植物
	w := perform(r, "POST", "/api/plants", map[string]any{"name": "X"})
	resp := decodeJSON(t, w)
	plantID := int(resp["data"].(map[string]any)["id"].(float64))

	// 创建养护记录
	w = perform(r, "POST", "/api/care-logs", map[string]any{
		"plantId": plantID,
		"type":    "water",
		"title":   "浇水",
	})
	if w.Code != 200 {
		t.Fatalf("create care log: %d %s", w.Code, w.Body.String())
	}

	// 列表
	w = perform(r, "GET", "/api/care-logs", nil)
	if w.Code != 200 {
		t.Fatalf("list care logs: %d", w.Code)
	}
}

// ---- PhotoDiary ----

func TestIntegration_DiaryCRUD(t *testing.T) {
	r := setupTest(t)

	// 创建植物
	w := perform(r, "POST", "/api/plants", map[string]any{"name": "X"})
	resp := decodeJSON(t, w)
	plantID := int(resp["data"].(map[string]any)["id"].(float64))

	// 创建日记（multipart form）
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("plantId", fmt.Sprintf("%d", plantID))
	writer.WriteField("caption", "今天长新叶了")
	writer.WriteField("takenAt", time.Now().Format(time.RFC3339))
	// 创建一个假图片文件
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write([]byte("fake-jpg-data"))
	writer.Close()

	w = httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/diaries", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("create diary: %d %s", w.Code, w.Body.String())
	}
	resp = decodeJSON(t, w)
	diaryID := int(resp["data"].(map[string]any)["id"].(float64))

	// 列表
	w = perform(r, "GET", "/api/diaries", nil)
	if w.Code != 200 {
		t.Fatalf("list diaries: %d", w.Code)
	}

	// 删除
	w = perform(r, "DELETE", fmt.Sprintf("/api/diaries/%d", diaryID), nil)
	if w.Code != 200 {
		t.Fatalf("delete diary: %d", w.Code)
	}
}

// ---- Settings ----

func TestIntegration_SettingsCRUD(t *testing.T) {
	r := setupTest(t)

	// 获取默认
	w := perform(r, "GET", "/api/settings", nil)
	if w.Code != 200 {
		t.Fatalf("get settings: %d", w.Code)
	}

	// 更新
	w = perform(r, "PUT", "/api/settings", map[string]any{
		"workdays": []int{1, 2, 3, 4, 5},
	})
	if w.Code != 200 {
		t.Fatalf("update settings: %d %s", w.Code, w.Body.String())
	}
}

// ---- Library Search ----

func TestIntegration_LibrarySearch(t *testing.T) {
	r := setupTest(t)

	// 插入资料库条目
	db := repositories.DB
	db.Create(&models.PlantLibrary{PID: "monstera", Name: "龟背竹", Guide: "浇水：适中"})
	db.Create(&models.PlantLibrary{Name: "绿萝", Guide: "浇水：频繁"})

	// 搜索
	w := perform(r, "GET", "/api/library?keyword=龟背", nil)
	if w.Code != 200 {
		t.Fatalf("search library: %d", w.Code)
	}
	resp := decodeJSON(t, w)
	list := resp["data"].([]any)
	if len(list) != 1 {
		t.Fatalf("search results = %d, want 1", len(list))
	}

	// 空搜索
	w = perform(r, "GET", "/api/library", nil)
	if w.Code != 200 {
		t.Fatalf("empty search: %d", w.Code)
	}
}

// ---- Error Cases ----

func TestIntegration_InvalidID(t *testing.T) {
	r := setupTest(t)

	w := perform(r, "GET", "/api/plants/abc", nil)
	if w.Code != 400 {
		t.Errorf("invalid id: status = %d, want 400", w.Code)
	}

	w = perform(r, "GET", "/api/plants/0", nil)
	if w.Code != 400 {
		t.Errorf("zero id: status = %d, want 400", w.Code)
	}
}

func TestIntegration_NotFound(t *testing.T) {
	r := setupTest(t)

	w := perform(r, "GET", "/api/plants/99999", nil)
	if w.Code != 404 {
		t.Errorf("not found: status = %d, want 404", w.Code)
	}
}

func TestIntegration_RoomDeleteWithoutPlants(t *testing.T) {
	r := setupTest(t)

	// 创建房间
	w := perform(r, "POST", "/api/rooms", map[string]any{"name": "R"})
	resp := decodeJSON(t, w)
	roomID := int(resp["data"].(map[string]any)["id"].(float64))

	// 在房间里创建植物
	perform(r, "POST", "/api/plants", map[string]any{"name": "P", "roomId": roomID})

	// 删除房间（有植物引用时不应删除）
	w = perform(r, "DELETE", fmt.Sprintf("/api/rooms/%d", roomID), nil)
	if w.Code == 200 {
		t.Error("room with plants should not be deleted")
	}
}

// ---- Import Preview ----

func TestIntegration_ImportPreviewPlants(t *testing.T) {
	r := setupTest(t)

	// multipart form with CSV file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("kind", "plants")
	part, _ := writer.CreateFormFile("file", "plants.csv")
	part.Write([]byte("name,species,room,note\n绿萝,Epipremnum,客厅,测试\n龟背竹,Monstera,,"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import/preview", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("preview plants: %d %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	data := resp["data"].(map[string]any)
	if data["valid"].(float64) != 2 {
		t.Errorf("valid = %v, want 2", data["valid"])
	}
}

func TestIntegration_ImportPreviewPlants_错误行(t *testing.T) {
	r := setupTest(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("kind", "plants")
	part, _ := writer.CreateFormFile("file", "bad.csv")
	part.Write([]byte("name,species\n,Epipremnum"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import/preview", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("preview: %d", w.Code)
	}
	resp := decodeJSON(t, w)
	data := resp["data"].(map[string]any)
	if data["invalid"].(float64) != 1 {
		t.Errorf("invalid = %v, want 1", data["invalid"])
	}
}

func TestIntegration_ImportConfirmPlants(t *testing.T) {
	r := setupTest(t)

	csv := "name,species,room\n绿萝,Epipremnum,\n龟背竹,Monstera,"
	w := perform(r, "POST", "/api/import/confirm", map[string]any{
		"kind":    "plants",
		"content": csv,
	})
	if w.Code != 200 {
		t.Fatalf("confirm plants: %d %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	data := resp["data"].(map[string]any)
	if data["created"].(float64) != 2 {
		t.Errorf("created = %v, want 2", data["created"])
	}
}

// ---- Import Template Preview ----

func TestIntegration_ImportTemplatePreview(t *testing.T) {
	r := setupTest(t)

	// 先创建植物
	w := perform(r, "POST", "/api/plants", map[string]any{"name": "Source"})
	resp := decodeJSON(t, w)
	sourceID := int(resp["data"].(map[string]any)["id"].(float64))

	w = perform(r, "POST", "/api/plants", map[string]any{"name": "Target1"})
	resp = decodeJSON(t, w)
	target1 := int(resp["data"].(map[string]any)["id"].(float64))

	w = perform(r, "POST", "/api/plants", map[string]any{"name": "Target2"})
	resp = decodeJSON(t, w)
	target2 := int(resp["data"].(map[string]any)["id"].(float64))

	// 模板预览
	w = perform(r, "POST", "/api/import/template-preview", map[string]any{
		"sourceId":  sourceID,
		"targetIds": []int{target1, target2},
	})
	// 来源无任务，应返回 warning
	if w.Code != 200 {
		t.Fatalf("template preview: %d %s", w.Code, w.Body.String())
	}
}

// ---- Weather Config ----

func TestIntegration_WeatherConfigCRUD(t *testing.T) {
	r := setupTest(t)

	// 获取默认配置
	w := perform(r, "GET", "/api/weather/config", nil)
	if w.Code != 200 {
		t.Fatalf("get config: %d", w.Code)
	}

	// 保存配置
	w = perform(r, "PUT", "/api/weather/config", map[string]any{
		"city":      "北京",
		"coldTemp":  5,
		"hotTemp":   35,
		"enabled":   true,
	})
	if w.Code != 200 {
		t.Fatalf("save config: %d %s", w.Code, w.Body.String())
	}

	// 验证保存
	w = perform(r, "GET", "/api/weather/config", nil)
	if w.Code != 200 {
		t.Fatalf("get config again: %d", w.Code)
	}
}

// ---- Task Done & Postpone ----

func TestIntegration_TaskDoneAndPostpone(t *testing.T) {
	r := setupTest(t)

	// 创建植物和任务
	w := perform(r, "POST", "/api/plants", map[string]any{"name": "X"})
	resp := decodeJSON(t, w)
	plantID := int(resp["data"].(map[string]any)["id"].(float64))

	w = perform(r, "POST", "/api/tasks", map[string]any{
		"plantId":      plantID,
		"type":         "water",
		"intervalDays": 7,
		"nextDue":      time.Now().Format(time.RFC3339),
	})
	resp = decodeJSON(t, w)
	taskID := int(resp["data"].(map[string]any)["id"].(float64))

	// 完成
	w = perform(r, "POST", fmt.Sprintf("/api/tasks/%d/done", taskID), nil)
	if w.Code != 200 {
		t.Fatalf("done: %d %s", w.Code, w.Body.String())
	}

	// 推迟
	w = perform(r, "POST", fmt.Sprintf("/api/tasks/%d/postpone", taskID), nil)
	if w.Code != 200 {
		t.Fatalf("postpone: %d %s", w.Code, w.Body.String())
	}
}

// ---- Room Update ----

func TestIntegration_RoomUpdate(t *testing.T) {
	r := setupTest(t)

	w := perform(r, "POST", "/api/rooms", map[string]any{"name": "客厅"})
	resp := decodeJSON(t, w)
	roomID := int(resp["data"].(map[string]any)["id"].(float64))

	w = perform(r, "PUT", fmt.Sprintf("/api/rooms/%d", roomID), map[string]any{"name": "新客厅"})
	if w.Code != 200 {
		t.Fatalf("update: %d", w.Code)
	}
}
