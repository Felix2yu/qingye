package services

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"qingye/server/models"
	"qingye/server/repositories"
)

// WeatherService 智能养护策略：根据实时天气调整任务频率、推迟室外浇水、记录调整日志
type WeatherService struct {
	mu      sync.RWMutex
	cfg     models.WeatherConfig
	current *WeatherNow // 最近一次天气快照（内存缓存，供 API 展示）

	logs *repositories.WeatherLogRepo
	tasks *repositories.TaskRepo
	rooms *repositories.RoomRepo
}

func NewWeatherService() *WeatherService {
	return &WeatherService{
		logs:  repositories.NewWeatherLogRepo(),
		tasks: repositories.NewTaskRepo(),
		rooms: repositories.NewRoomRepo(),
	}
}

// ---------- 配置读写（存于 UserSetting.Prefs） ----------

// LoadConfig 从设置读取天气策略配置；无则返回默认
func (s *WeatherService) LoadConfig() models.WeatherConfig {
	cfg, err := LoadWeatherConfig()
	if err != nil {
		return models.DefaultWeatherConfig()
	}
	return cfg
}

func LoadWeatherConfig() (models.WeatherConfig, error) {
	st, err := NewSettingService().Get()
	if err != nil {
		return models.DefaultWeatherConfig(), err
	}
	var cfg models.WeatherConfig
	if st.Prefs == "" || st.Prefs == "{}" {
		return models.DefaultWeatherConfig(), nil
	}
	var prefs map[string]any
	if err := json.Unmarshal([]byte(st.Prefs), &prefs); err != nil {
		return models.DefaultWeatherConfig(), nil
	}
	if prefs == nil {
		return models.DefaultWeatherConfig(), nil
	}
	// prefs.weather 为可选嵌套对象
	raw, ok := prefs["weather"]
	if !ok || raw == nil {
		return models.DefaultWeatherConfig(), nil
	}
	b, _ := json.Marshal(raw)
	cfg = models.DefaultWeatherConfig()
	if err := json.Unmarshal(b, &cfg); err != nil {
		return models.DefaultWeatherConfig(), nil
	}
	return cfg, nil
}

// SaveConfig 保存天气策略配置到 UserSetting.Prefs
func SaveWeatherConfig(cfg models.WeatherConfig) error {
	st, err := NewSettingService().Get()
	if err != nil {
		return err
	}
	var prefs map[string]any
	if st.Prefs != "" {
		_ = json.Unmarshal([]byte(st.Prefs), &prefs)
	}
	if prefs == nil {
		prefs = map[string]any{}
	}
	prefs["weather"] = cfg
	b, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	// 仅更新 prefs，不触碰工作日
	return repositories.NewSettingRepo().Save(&models.UserSetting{ID: st.ID, Workdays: st.Workdays, Prefs: string(b)})
}

// Location 用于查询天气的位置字符串：经纬度优先，其次城市
func (s *WeatherService) Location(cfg models.WeatherConfig) string {
	if cfg.Lat != 0 && cfg.Lon != 0 {
		return fmt.Sprintf("%.2f,%.2f", cfg.Lon, cfg.Lat) // 和风 location=经度,纬度
	}
	return strings.TrimSpace(cfg.City)
}

// Current 返回最近一次天气快照
func (s *WeatherService) Current() *WeatherNow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Logs 返回天气调整日志（按时间倒序）
func (s *WeatherService) Logs(limit int) ([]models.WeatherLog, error) {
	return s.logs.List(limit)
}

// ---------- 策略执行 ----------

// Poll 执行一次天气轮询：拉天气 → 温度/降雨策略调整 → 记日志
func (s *WeatherService) Poll() {
	cfg := s.LoadConfig()
	if qweatherAPIKey() == "" {
		return // 未配置 key，模块关闭
	}
	if !cfg.Enabled {
		return // 用户未启用
	}
	loc := s.Location(cfg)
	if loc == "" {
		return // 未配置位置
	}
	now, err := QWeatherNow(loc)
	if err != nil {
		// 记录一次失败快照，避免吞错
		s.mu.Lock()
		s.current = nil
		s.mu.Unlock()
		return
	}
	s.mu.Lock()
	s.current = now
	s.mu.Unlock()

	s.applyStrategy(cfg, now)
}

// applyStrategy 依据天气类别调整任务并记录日志
func (s *WeatherService) applyStrategy(cfg models.WeatherConfig, now *WeatherNow) {
	tasks, err := s.tasks.ListActiveByType("")
	if err != nil {
		return
	}
	isRain := isRaining(now.Condition, now.Icon)

	// 收集本次调整要点，用于推送通知（仅在有调整时发送）
	var notes []string

	// 温度类别
	kind := "normal"
	if now.Temp < cfg.ColdTemp {
		kind = models.WeatherKindCold
	} else if now.Temp > cfg.HotTemp {
		kind = models.WeatherKindHot
	}

	// 先解析 base interval：任务调整前需有基准
	for i := range tasks {
		t := &tasks[i]
		if t.Type != "water" && t.Type != "fertilize" {
			continue
		}
		base := t.BaseIntervalDays
		if base <= 0 {
			base = t.IntervalDays
		}

		var factor float64 = 1
		switch {
		case kind == models.WeatherKindCold:
			// 低温：降低浇水与施肥频率（间隔变长）
			if t.Type == "water" {
				factor = 1 + float64(cfg.WaterAdj)/100
			} else {
				factor = 1 + float64(cfg.FertAdj)/100
			}
		case kind == models.WeatherKindHot:
			// 高温：降低施肥；浇水保持或适度增加（间隔略短，但不低于 50%）
			if t.Type == "water" {
				factor = math.Max(0.5, 1-float64(cfg.WaterAdj)/100)
			} else {
				factor = 1 + float64(cfg.FertAdj)/100
			}
		default:
			factor = 1 // 正常恢复基准
		}

		newInterval := int(math.Round(float64(base) * factor))
		if newInterval < 1 {
			newInterval = 1
		}
		if newInterval == t.IntervalDays {
			continue
		}
		if err := s.tasks.SetInterval(t.ID, base, newInterval); err != nil {
			continue
		}
		detail := fmt.Sprintf("温度 %.1f°C（%s）：%s 周期 %d→%d 天",
			now.Temp, kindLabel(kind), TaskTypeName(t.Type), t.IntervalDays, newInterval)
		_ = s.logs.Create(&models.WeatherLog{
			At: time.Now(), Temp: now.Temp, Condition: now.Condition,
			Kind: kind, TaskID: &t.ID, PlantID: &t.PlantID, Detail: detail,
		})
		notes = append(notes, detail)
	}

	// 降雨：推迟室外植物浇水任务
	if isRain {
		s.deferOutdoorWatering(cfg, now, &notes)
	}

	// 有调整时推送天气通知（节流由 NotifyService 内部处理）
	if len(notes) > 0 {
		msg := fmt.Sprintf("🌤️ 青野集天气养护调整（%s %.1f°C）\n%s",
			now.Condition, now.Temp, strings.Join(notes, "\n"))
		NewNotifyService().WeatherAlert(msg)
	}
}

// deferOutdoorWatering 降雨时推迟室外植物（Room.IsOutdoor）的浇水任务
func (s *WeatherService) deferOutdoorWatering(cfg models.WeatherConfig, now *WeatherNow, notes *[]string) {
	plants, err := repositories.NewPlantRepo().List(0)
	if err != nil {
		return
	}
	nowTime := time.Now()
	for _, p := range plants {
		if !roomOutdoor(s.rooms, p.RoomID) {
			continue
		}
		tasks, err := s.tasks.ListActiveByPlantType(p.ID, "water")
		if err != nil {
			continue
		}
		for i := range tasks {
			t := &tasks[i]
			// 若 next_due 即将到来（未来 rainDelay 小时内），推迟
			if t.NextDue.Before(nowTime) || t.NextDue.Sub(nowTime).Hours() <= float64(cfg.RainDelayH) {
				newDue := nowTime.Add(time.Duration(cfg.RainDelayH) * time.Hour)
				if err := s.tasks.SetNextDue(t.ID, newDue); err != nil {
					continue
				}
				_ = s.logs.Create(&models.WeatherLog{
					At: nowTime, Temp: now.Temp, Condition: now.Condition,
					Kind: models.WeatherKindRain, TaskID: &t.ID, PlantID: &p.ID,
					Detail: fmt.Sprintf("降雨(%s)：室外植物 %s 的浇水任务推迟 %d 小时", now.Condition, p.Name, cfg.RainDelayH),
				})
				if notes != nil {
					*notes = append(*notes, fmt.Sprintf("🌧️ %s 浇水推迟 %d 小时", p.Name, cfg.RainDelayH))
				}
			}
		}
	}
}

func roomOutdoor(rooms *repositories.RoomRepo, roomID uint) bool {
	roomsSet := make(map[uint]bool)
	if list, err := rooms.List(); err == nil {
		for _, r := range list {
			if r.IsOutdoor {
				roomsSet[r.ID] = true
			}
		}
	}
	return roomsSet[roomID]
}

func isRaining(condition, icon string) bool {
	c := strings.ToLower(condition)
	if strings.Contains(c, "雨") || strings.Contains(c, "rain") {
		return true
	}
	// 和风 icon 代码：3xx 雷阵雨、1xx-2xx 降雨
	if len(icon) >= 1 {
		first := icon[0]
		if first == '3' {
			return true
		}
		if first == '1' || first == '2' {
			return true
		}
	}
	return false
}

func kindLabel(k string) string {
	switch k {
	case models.WeatherKindCold:
		return "低温"
	case models.WeatherKindHot:
		return "高温"
	case models.WeatherKindRain:
		return "降雨"
	}
	return "正常"
}
