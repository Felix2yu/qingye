package services

import "testing"

func TestTaskTypeEmoji(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{TaskTypeWater, "💧"},
		{TaskTypeFertilize, "🌱"},
		{TaskTypeMist, "🌫️"},
		{TaskTypeRepot, "🪴"},
		{TaskTypePrune, "✂️"},
		{TaskTypeClean, "🧹"},
		{TaskTypePesticide, "🐛"},
		{TaskTypeOther, "✨"},
		{"unknown", "✨"},
	}
	for _, tt := range tests {
		if got := taskTypeEmoji(tt.in); got != tt.want {
			t.Errorf("taskTypeEmoji(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNewNotifyService(t *testing.T) {
	if NewNotifyService() == nil {
		t.Error("NewNotifyService should not be nil")
	}
}

func TestNotifyService_Test_NoURL(t *testing.T) {
	setupTestDB(t)
	n := NewNotifyService()
	msg, err := n.Test()
	if err != nil {
		t.Fatal(err)
	}
	if msg != "未配置通知地址" {
		t.Errorf("msg = %q", msg)
	}
}

func TestNotifyService_Send_NoURL(t *testing.T) {
	setupTestDB(t)
	n := NewNotifyService()
	if err := n.Send("hello"); err != nil {
		t.Errorf("Send with no url should return nil, got %v", err)
	}
}

func TestNotifyService_WeatherAlert_Empty(t *testing.T) {
	n := NewNotifyService()
	n.WeatherAlert("") // 不应 panic，提前返回
}

func TestNotifyService_WeatherAlert_Send(t *testing.T) {
	setupTestDB(t)
	n := NewNotifyService()
	// 已配置空 url 时内部静默跳过，不 panic
	n.WeatherAlert("降雨提醒")
}
