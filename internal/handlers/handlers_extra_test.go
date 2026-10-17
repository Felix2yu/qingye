package handlers

import "testing"

func TestTaskHandler_List(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "GET", "/api/tasks", nil)
	if w.Code != 200 {
		t.Fatalf("GET /api/tasks code=%d", w.Code)
	}
}

func TestTaskHandler_Today(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "GET", "/api/tasks/today", nil)
	if w.Code != 200 {
		t.Fatalf("GET /api/tasks/today code=%d", w.Code)
	}
}

func TestTaskHandler_Upcoming(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "GET", "/api/tasks/upcoming", nil)
	if w.Code != 200 {
		t.Fatalf("GET /api/tasks/upcoming code=%d", w.Code)
	}
}

func TestTaskHandler_Logs(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "GET", "/api/tasks/1/logs", nil)
	// 任务不存在 → 400；存在 → 200。均覆盖 handler
	if w.Code != 200 && w.Code != 400 {
		t.Fatalf("GET /api/tasks/1/logs code=%d", w.Code)
	}
}

func TestNotifyHandler_SaveNotify(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "PUT", "/api/settings/notify", map[string]any{"url": "discord://x@y"})
	if w.Code != 200 {
		t.Fatalf("PUT /api/settings/notify code=%d", w.Code)
	}
}

func TestNotifyHandler_SaveDigestHour(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "PUT", "/api/settings/digest-hour", map[string]any{"hour": 9})
	if w.Code != 200 {
		t.Fatalf("PUT /api/settings/digest-hour code=%d", w.Code)
	}
}

func TestNotifyHandler_Test(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "POST", "/api/notify/test", nil)
	if w.Code != 200 {
		t.Fatalf("POST /api/notify/test code=%d", w.Code)
	}
}

func TestWeatherHandler_Current(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "GET", "/api/weather/current", nil)
	if w.Code != 200 {
		t.Fatalf("GET /api/weather/current code=%d", w.Code)
	}
}

func TestWeatherHandler_Logs(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "GET", "/api/weather/logs", nil)
	if w.Code != 200 {
		t.Fatalf("GET /api/weather/logs code=%d", w.Code)
	}
}

func TestWeatherHandler_Refresh(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "POST", "/api/weather/refresh", nil)
	// 未配置 key → 400；配置了 → 200。两条路径均覆盖 handler
	if w.Code != 200 && w.Code != 400 {
		t.Fatalf("POST /api/weather/refresh code=%d", w.Code)
	}
}

func TestLibraryHandler_SearchOnline_NoToken(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "GET", "/api/library/online", nil)
	if w.Code != 200 {
		t.Fatalf("GET /api/library/online code=%d", w.Code)
	}
}

func TestLibraryHandler_SyncPopular(t *testing.T) {
	r := setupTest(t)
	w := perform(r, "POST", "/api/library/sync-popular", map[string]any{"limit": 1})
	// 未配置凭据 → 400；配置了 → 200。均覆盖 handler
	if w.Code != 200 && w.Code != 400 {
		t.Fatalf("POST /api/library/sync-popular code=%d", w.Code)
	}
}
