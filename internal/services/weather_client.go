package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// qweatherAPIKey 和风天气 API Key（环境变量 QWEATHER_KEY），未配置时天气模块关闭
func qweatherAPIKey() string { return os.Getenv("QWEATHER_KEY") }

// QWeatherKeyAvailable 是否已配置天气 API Key
func QWeatherKeyAvailable() bool { return os.Getenv("QWEATHER_KEY") != "" }

// WeatherNow 实时天气结果
type WeatherNow struct {
	Temp      float64
	Condition string // 天气现象文本（晴 / 多云 / 小雨…）
	Icon      string // 图标代码
	ObsTime   time.Time
}

// QWeatherNow 调和风天气 v7 实时天气接口
// GET https://devapi.qweather.com/v7/weather/now?location=经度,纬度&key=KEY
// location 支持经纬度（"经度,纬度"）或城市 LocationID。
func QWeatherNow(loc string) (*WeatherNow, error) {
	key := qweatherAPIKey()
	if key == "" {
		return nil, fmt.Errorf("未配置 QWEATHER_KEY 环境变量，天气模块不可用")
	}
	base := strings.TrimRight(os.Getenv("QWEATHER_API"), "/")
	if base == "" {
		base = "https://devapi.qweather.com/v7/weather/now"
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	url := fmt.Sprintf("%s%slocation=%s&key=%s", base, sep, loc, key)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求天气失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("天气接口返回 %d", resp.StatusCode)
	}

	var payload struct {
		Code string `json:"code"`
		Now  struct {
			Temp string `json:"temp"`
			Text string `json:"text"`
			Icon string `json:"icon"`
		} `json:"now"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析天气响应失败: %w", err)
	}
	if payload.Code != "200" {
		return nil, fmt.Errorf("天气接口业务错误: code=%s", payload.Code)
	}
	temp, _ := strconv.ParseFloat(payload.Now.Temp, 64)
	return &WeatherNow{
		Temp:      temp,
		Condition: payload.Now.Text,
		Icon:      payload.Now.Icon,
	}, nil
}
