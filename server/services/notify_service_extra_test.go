package services

import (
	"testing"
)

func TestNotifyService_Send_InvalidURL(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()
	if _, err := svc.SaveNotifyURL("bogusscheme://x"); err != nil {
		t.Fatalf("save notify url: %v", err)
	}
	n := NewNotifyService()
	// 未知 scheme 应使 shoutrrr.Send 失败，覆盖 Send 的 error 分支
	if err := n.Send("hi"); err == nil {
		t.Error("expected error from invalid notify url")
	}
}

func TestNotifyService_Send_DisabledNoPanic(t *testing.T) {
	// 无 setting 行时 url() 可能返回错误，Send 应原样返回错误而非 panic
	n := NewNotifyService()
	_ = n.Send("hi")
}

func TestNotifyService_WeatherAlert_RealSend(t *testing.T) {
	setupTestDB(t)
	// 未配置地址时 Send 静默跳过，WeatherAlert 不应 panic
	n := NewNotifyService()
	n.WeatherAlert("今天小雨，室外浇水已推迟")
}

func TestTaskTypeEmoji_All(t *testing.T) {
	cases := map[string]string{
		TaskTypeWater:      "💧",
		TaskTypeFertilize:  "🌱",
		TaskTypeMist:       "🌫️",
		TaskTypeRepot:      "🪴",
		TaskTypePrune:      "✂️",
		TaskTypeClean:      "🧹",
		TaskTypePesticide:  "🐛",
		TaskTypeOther:      "✨",
	}
	for typ, want := range cases {
		if got := taskTypeEmoji(typ); got != want {
			t.Errorf("taskTypeEmoji(%q) = %q, want %q", typ, got, want)
		}
	}
	if got := taskTypeEmoji("unknown"); got != "✨" {
		t.Errorf("taskTypeEmoji(unknown) = %q, want ✨", got)
	}
}
