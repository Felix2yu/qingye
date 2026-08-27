package services

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"qingye/server/models"
	"qingye/server/repositories"
)

func TestQWeatherKeyAvailable(t *testing.T) {
	os.Setenv("QWEATHER_KEY", "k")
	defer os.Unsetenv("QWEATHER_KEY")
	if !QWeatherKeyAvailable() {
		t.Error("should be available when QWEATHER_KEY set")
	}
	os.Unsetenv("QWEATHER_KEY")
	if QWeatherKeyAvailable() {
		t.Error("should be unavailable when QWEATHER_KEY empty")
	}
}

func TestQWeatherNow_Success(t *testing.T) {
	os.Setenv("QWEATHER_KEY", "k")
	defer os.Unsetenv("QWEATHER_KEY")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"200","now":{"temp":"21.5","text":"晴","icon":"100"}}`))
	}))
	defer srv.Close()
	os.Setenv("QWEATHER_API", srv.URL)
	defer os.Unsetenv("QWEATHER_API")

	now, err := QWeatherNow("116.40,39.90")
	if err != nil {
		t.Fatalf("QWeatherNow err: %v", err)
	}
	if now.Temp != 21.5 || now.Condition != "晴" || now.Icon != "100" {
		t.Errorf("unexpected weather: %+v", now)
	}
}

func TestQWeatherNow_NoKey(t *testing.T) {
	os.Unsetenv("QWEATHER_KEY")
	if _, err := QWeatherNow("116.40,39.90"); err == nil {
		t.Error("expected error when QWEATHER_KEY unset")
	}
}

func TestQWeatherNow_StatusNotOK(t *testing.T) {
	os.Setenv("QWEATHER_KEY", "k")
	defer os.Unsetenv("QWEATHER_KEY")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	os.Setenv("QWEATHER_API", srv.URL)
	defer os.Unsetenv("QWEATHER_API")

	if _, err := QWeatherNow("116.40,39.90"); err == nil {
		t.Error("expected error on non-200 status")
	}
}

func TestQWeatherNow_BusinessError(t *testing.T) {
	os.Setenv("QWEATHER_KEY", "k")
	defer os.Unsetenv("QWEATHER_KEY")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"404","now":{}}`))
	}))
	defer srv.Close()
	os.Setenv("QWEATHER_API", srv.URL)
	defer os.Unsetenv("QWEATHER_API")

	if _, err := QWeatherNow("116.40,39.90"); err == nil {
		t.Error("expected business error when code != 200")
	}
}

func TestWeatherService_Poll_Success(t *testing.T) {
	setupTestDB(t)
	os.Setenv("QWEATHER_KEY", "k")
	defer os.Unsetenv("QWEATHER_KEY")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"200","now":{"temp":"18.0","text":"小雨","icon":"305"}}`))
	}))
	defer srv.Close()
	os.Setenv("QWEATHER_API", srv.URL)
	defer os.Unsetenv("QWEATHER_API")

	cfg := models.DefaultWeatherConfig()
	cfg.Enabled = true
	cfg.Lat = 39.9
	cfg.Lon = 116.4
	cfg.RainDelayH = 24
	if err := SaveWeatherConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	svc := NewWeatherService()
	svc.Poll()
	if svc.Current() == nil {
		t.Error("current snapshot should be set after successful poll")
	}
}

func TestWeatherService_Poll_Disabled(t *testing.T) {
	setupTestDB(t)
	os.Unsetenv("QWEATHER_KEY")
	svc := NewWeatherService()
	svc.Poll() // 无 key，应直接返回，不 panic
	if svc.Current() != nil {
		t.Error("current should remain nil when disabled")
	}
}

func TestWeatherConfig_RoundTrip(t *testing.T) {
	setupTestDB(t)
	cfg := models.DefaultWeatherConfig()
	cfg.Enabled = true
	cfg.ColdTemp = 5
	cfg.HotTemp = 33
	cfg.WaterAdj = -2
	cfg.FertAdj = 3
	cfg.RainDelayH = 18
	cfg.Lat = 31.2
	cfg.Lon = 121.5
	if err := SaveWeatherConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	got, err := LoadWeatherConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !got.Enabled || got.ColdTemp != 5 || got.HotTemp != 33 ||
		got.WaterAdj != -2 || got.FertAdj != 3 || got.RainDelayH != 18 ||
		got.Lat != 31.2 || got.Lon != 121.5 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestLoadWeatherConfig_InvalidPrefsFallback(t *testing.T) {
	setupTestDB(t)
	st, err := NewSettingService().Get()
	if err != nil {
		t.Fatal(err)
	}
	st.Prefs = "not-json"
	if err := repositories.NewSettingRepo().Save(st); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadWeatherConfig()
	if err != nil {
		t.Fatalf("should fallback, got err: %v", err)
	}
	if cfg.Enabled {
		t.Error("fallback config should be default (disabled)")
	}
}
