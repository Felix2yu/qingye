package handlers

import (
	"testing"
)

// 触发各类 Create/Update 处理器的请求体错误分支（无效 JSON / 缺字段），
// 这些分支在未传错误请求时长期 0% 覆盖。
func TestHandler_ImportOnline_EmptyPID(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "POST", "/api/library/import", map[string]any{})
	if w.Code != 400 {
		t.Errorf("ImportOnline empty pid code=%d, want 400", w.Code)
	}
}

func TestHandler_TaskCreate_InvalidJSON(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "POST", "/api/tasks", "not-json")
	if w.Code != 400 {
		t.Errorf("Task Create invalid json code=%d, want 400", w.Code)
	}
}

func TestHandler_TaskCreate_MissingPlant(t *testing.T) {
	r := setupTest(t)
	// 合法 JSON 但植物不存在：svc.Create 返回错误
	w := perform(r, "POST", "/api/tasks", map[string]any{
		"plantId":      99999,
		"type":         "water",
		"title":        "浇水",
		"intervalDays": 7,
	})
	if w.Code != 400 {
		t.Errorf("Task Create missing plant code=%d, want 400", w.Code)
	}
}

func TestHandler_CareLogCreate_InvalidJSON(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "POST", "/api/care-logs", "not-json")
	if w.Code != 400 {
		t.Errorf("CareLog Create invalid json code=%d, want 400", w.Code)
	}
}

func TestHandler_CareLogCreate_MissingPlant(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "POST", "/api/care-logs", map[string]any{"type": "water", "title": "浇水"})
	if w.Code != 400 {
		t.Errorf("CareLog Create missing plant code=%d, want 400", w.Code)
	}
}

func TestHandler_SettingUpdate_InvalidJSON(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "PUT", "/api/settings", "not-json")
	if w.Code != 400 {
		t.Errorf("Setting Update invalid json code=%d, want 400", w.Code)
	}
}

func TestHandler_SettingUpdate_NilWorkdays(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "PUT", "/api/settings", map[string]any{"prefs": map[string]any{}})
	if w.Code != 400 {
		t.Errorf("Setting Update nil workdays code=%d, want 400", w.Code)
	}
}

func TestHandler_RoomCreate_InvalidJSON(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "POST", "/api/rooms", "not-json")
	if w.Code != 400 {
		t.Errorf("Room Create invalid json code=%d, want 400", w.Code)
	}
}

func TestHandler_PlantCreate_InvalidJSON(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "POST", "/api/plants", "not-json")
	if w.Code != 400 {
		t.Errorf("Plant Create invalid json code=%d, want 400", w.Code)
	}
}

func TestHandler_NotifySave_InvalidJSON(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "PUT", "/api/settings/notify", "not-json")
	if w.Code != 400 {
		t.Errorf("Notify Save invalid json code=%d, want 400", w.Code)
	}
}

func TestHandler_WeatherSaveConfig_InvalidJSON(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "PUT", "/api/weather/config", "not-json")
	if w.Code != 400 {
		t.Errorf("Weather SaveConfig invalid json code=%d, want 400", w.Code)
	}
}

// 校验 diaries 列表接口在无数据时返回空数组（覆盖 diary List 成功分支）
func TestHandler_DiaryList_Empty(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "GET", "/api/diaries", nil)
	if w.Code != 200 {
		t.Fatalf("GET /api/diaries code=%d", w.Code)
	}
}

// 校验 care-logs 列表接口在无数据时返回 200（覆盖 care-log List 成功分支）
func TestHandler_CareLogList_Empty(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "GET", "/api/care-logs", nil)
	if w.Code != 200 {
		t.Fatalf("GET /api/care-logs code=%d", w.Code)
	}
}
