package models

import "time"

// WeatherConfig 天气智能养护策略配置（存储在 UserSetting.Prefs 的 JSON 中）
type WeatherConfig struct {
	City        string  `json:"city"`        // 城市名（展示用）
	Lat         float64 `json:"lat"`         // 纬度（未配置经纬度时可为 0）
	Lon         float64 `json:"lon"`         // 经度
	ColdTemp    float64 `json:"coldTemp"`    // 低温阈值（°C）：低于则降低浇水/施肥频率
	HotTemp     float64 `json:"hotTemp"`     // 高温阈值（°C）：高于则降低施肥、保持/增加浇水
	WaterAdj    int     `json:"waterAdj"`    // 浇水频率调整幅度（百分比，如 30 表示 ×1.3）
	FertAdj     int     `json:"fertAdj"`     // 施肥频率调整幅度（百分比，如 30 表示 ×1.3）
	RainDelayH  int     `json:"rainDelayH"`  // 降雨时推迟室外植物浇水的小时数
	Enabled     bool    `json:"enabled"`     // 是否启用天气策略
	PollMinutes int     `json:"pollMinutes"` // 轮询间隔（分钟）
}

// DefaultWeatherConfig 返回默认策略配置
func DefaultWeatherConfig() WeatherConfig {
	return WeatherConfig{
		ColdTemp:    5,
		HotTemp:     32,
		WaterAdj:    30,
		FertAdj:     30,
		RainDelayH:  24,
		Enabled:     false,
		PollMinutes: 60,
	}
}

// WeatherLog 天气触发的策略调整日志
type WeatherLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	At        time.Time `gorm:"index" json:"at"`
	Temp      float64   `json:"temp"`    // 触发时温度
	Condition string    `gorm:"size:30" json:"condition"` // 天气现象（晴/雨…）
	Kind      string    `gorm:"size:30" json:"kind"`      // cold / hot / rain / refresh
	TaskID    *uint     `gorm:"index" json:"taskId,omitempty"`
	PlantID   *uint     `gorm:"index" json:"plantId,omitempty"`
	Detail    string    `gorm:"size:300" json:"detail"` // 调整说明
}

// WeatherLog 触发类型
const (
	WeatherKindCold    = "cold"
	WeatherKindHot     = "hot"
	WeatherKindRain    = "rain"
	WeatherKindRefresh = "refresh" // 轮询无调整
)
