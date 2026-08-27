package services

import (
	"testing"
	"time"

	"qingye/server/models"
	"qingye/server/repositories"
)

func TestWeatherConfig_DefaultWhenNotSet(t *testing.T) {
	setupTestDB(t)
	def, _ := LoadWeatherConfig()
	if def.ColdTemp != 5 {
		t.Errorf("default ColdTemp = %v, want 5", def.ColdTemp)
	}
	if def.Enabled {
		t.Error("default Enabled should be false")
	}
}

func TestWeatherConfig_LoadSave(t *testing.T) {
	setupTestDB(t)
	cfg := models.WeatherConfig{
		City:       "北京",
		ColdTemp:   3,
		HotTemp:    35,
		WaterAdj:   20,
		FertAdj:    25,
		RainDelayH: 12,
		Enabled:    true,
	}
	if err := SaveWeatherConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got, _ := LoadWeatherConfig()
	if got.City != "北京" || got.ColdTemp != 3 || !got.Enabled {
		t.Errorf("loaded config = %+v", got)
	}
}

func TestWeatherService_LoadConfig(t *testing.T) {
	setupTestDB(t)
	svc := NewWeatherService()
	cfg := svc.LoadConfig()
	if cfg.ColdTemp != 5 {
		t.Errorf("LoadConfig default ColdTemp = %v", cfg.ColdTemp)
	}
}

func TestKindLabel(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{models.WeatherKindCold, "低温"},
		{models.WeatherKindHot, "高温"},
		{models.WeatherKindRain, "降雨"},
		{"normal", "正常"},
		{"", "正常"},
	}
	for _, tt := range tests {
		if got := kindLabel(tt.in); got != tt.want {
			t.Errorf("kindLabel(%q) = %q", tt.in, got)
		}
	}
}

func TestIsRaining(t *testing.T) {
	tests := []struct {
		cond, icon string
		want       bool
	}{
		{"小雨", "", true},
		{"Rain", "", true},
		{"晴", "999", false},
		{"多云", "999", false},
		{"晴", "", false},
		{"雷阵雨", "302", true},
		{"阴", "2xx", true},
	}
	for _, tt := range tests {
		if got := isRaining(tt.cond, tt.icon); got != tt.want {
			t.Errorf("isRaining(%q,%q) = %v, want %v", tt.cond, tt.icon, got, tt.want)
		}
	}
}

func TestRoomOutdoor(t *testing.T) {
	setupTestDB(t)
	if err := repositories.NewRoomRepo().Create(&models.Room{Name: "阳台", IsOutdoor: true}); err != nil {
		t.Fatal(err)
	}
	if err := repositories.NewRoomRepo().Create(&models.Room{Name: "客厅", IsOutdoor: false}); err != nil {
		t.Fatal(err)
	}
	repo := repositories.NewRoomRepo()
	rooms, _ := repo.List()
	var outdoorID, indoorID uint
	for _, r := range rooms {
		if r.IsOutdoor {
			outdoorID = r.ID
		} else {
			indoorID = r.ID
		}
	}
	if !roomOutdoor(repo, outdoorID) {
		t.Error("outdoor room should be true")
	}
	if roomOutdoor(repo, indoorID) {
		t.Error("indoor room should be false")
	}
}

func TestWeatherService_ApplyStrategy_Cold(t *testing.T) {
	db := setupTestDB(t)
	svc := NewWeatherService()
	p := &models.Plant{Name: "X"}
	db.Create(p)
	db.Create(&models.Task{PlantID: p.ID, Type: "water", Title: "浇水", IntervalDays: 7, BaseIntervalDays: 7, NextDue: time.Now(), Active: true})

	cfg := models.WeatherConfig{ColdTemp: 10, WaterAdj: 30, Enabled: true}
	svc.applyStrategy(cfg, &WeatherNow{Temp: 0, Condition: "晴", Icon: "100"})

	var t2 models.Task
	db.Where("plant_id = ?", p.ID).First(&t2)
	// 低温应拉长浇水间隔：7 * 1.3 = 9.1 → round 9
	if t2.IntervalDays != 9 {
		t.Errorf("interval after cold = %d, want 9", t2.IntervalDays)
	}
}

func TestWeatherService_ApplyStrategy_RainDefersOutdoor(t *testing.T) {
	db := setupTestDB(t)
	svc := NewWeatherService()
	if err := repositories.NewRoomRepo().Create(&models.Room{Name: "阳台", IsOutdoor: true}); err != nil {
		t.Fatal(err)
	}
	room, _ := repositories.NewRoomRepo().List()
	var roomID uint
	for _, r := range room {
		if r.IsOutdoor {
			roomID = r.ID
		}
	}
	p := &models.Plant{Name: "X", RoomID: roomID}
	db.Create(p)
	now := time.Now()
	db.Create(&models.Task{PlantID: p.ID, Type: "water", Title: "浇水", IntervalDays: 7, BaseIntervalDays: 7, NextDue: now.Add(time.Hour), Active: true})

	cfg := models.WeatherConfig{RainDelayH: 24, Enabled: true}
	svc.applyStrategy(cfg, &WeatherNow{Temp: 20, Condition: "小雨", Icon: "305"})

	var t2 models.Task
	db.Where("plant_id = ?", p.ID).First(&t2)
	// 应推迟：next_due 至少推到 now + 24h 之后
	if t2.NextDue.Before(now.Add(23 * time.Hour)) {
		t.Errorf("next_due not deferred: %v", t2.NextDue)
	}
}

func TestWeatherService_Current_Logs(t *testing.T) {
	setupTestDB(t)
	svc := NewWeatherService()
	if svc.Current() != nil {
		t.Error("Current should be nil initially")
	}
	logs, err := svc.Logs(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 0 {
		t.Errorf("logs = %d, want 0", len(logs))
	}
}

func TestWeatherService_Location(t *testing.T) {
	svc := NewWeatherService()
	// 经纬度优先：location=经度,纬度
	if loc := svc.Location(models.WeatherConfig{Lat: 39.9, Lon: 116.4}); loc != "116.40,39.90" {
		t.Errorf("loc lat/lon = %q", loc)
	}
	// 否则城市
	if loc := svc.Location(models.WeatherConfig{City: "北京"}); loc != "北京" {
		t.Errorf("loc city = %q", loc)
	}
}

func TestWeatherService_Poll_NoKey(t *testing.T) {
	setupTestDB(t)
	svc := NewWeatherService()
	// 未配置 key 时 Poll 应安全返回（不 panic）
	svc.Poll()
}
