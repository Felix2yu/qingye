package services

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/containrrr/shoutrrr"
)

// NotifyService 基于 shoutrrr 的通知发送服务。
// 通知地址（shoutrrr URL，如 discord://token@id、telegram://token@chatid、
// gotify://host/token、slack://token@channel 等）存储在 UserSetting.NotifyURL，
// 未配置时所有发送静默跳过。
//
// 模块使用我自己 fork 的 github.com/Felix2yu/shoutrrr（go.mod 中以 replace 指向）。
type NotifyService struct {
	mu              sync.Mutex
	lastWeatherSent time.Time // 天气通知节流：同一天气状况下 6 小时内只发一次，避免轮询刷屏
}

func NewNotifyService() *NotifyService { return &NotifyService{} }

// url 读取并清理配置的通知地址
func (n *NotifyService) url() (string, error) {
	st, err := NewSettingService().Get()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(st.NotifyURL), nil
}

// Send 发送一条通知；未配置地址时静默跳过（返回 nil）。
func (n *NotifyService) Send(message string) error {
	url, err := n.url()
	if err != nil {
		log.Printf("[notify] 读取配置失败: %v", err)
		return err
	}
	if url == "" {
		return nil
	}
	if err := shoutrrr.Send(url, message); err != nil {
		log.Printf("[notify] 发送失败: %v", err)
		return err
	}
	return nil
}

// Test 发送一条测试通知，返回给前端的提示信息。
func (n *NotifyService) Test() (string, error) {
	url, err := n.url()
	if err != nil {
		return "", err
	}
	if url == "" {
		return "未配置通知地址", nil
	}
	if err := n.Send("✅ 青野集通知测试：配置成功！"); err != nil {
		return "", fmt.Errorf("发送失败：%v", err)
	}
	return "测试通知已发送，请查收", nil
}

// WeatherAlert 发送天气策略调整通知。6 小时内同一来源仅发一次，避免轮询刷屏。
// 文本为空时不发送。
func (n *NotifyService) WeatherAlert(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	n.mu.Lock()
	if time.Since(n.lastWeatherSent) < 6*time.Hour {
		n.mu.Unlock()
		return
	}
	n.lastWeatherSent = time.Now()
	n.mu.Unlock()

	_ = n.Send(text)
}

// taskTypeEmoji 任务类型对应的 emoji，用于通知摘要
func taskTypeEmoji(t string) string {
	switch t {
	case TaskTypeWater:
		return "💧"
	case TaskTypeFertilize:
		return "🌱"
	case TaskTypeMist:
		return "🌫️"
	case TaskTypeRepot:
		return "🪴"
	case TaskTypePrune:
		return "✂️"
	case TaskTypeClean:
		return "🧹"
	case TaskTypePesticide:
		return "🐛"
	default:
		return "✨"
	}
}
