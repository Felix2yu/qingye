package services

import (
	"testing"

	"qingye/server/repositories"
)

func TestSettingService_NotifyURL_SaveNotifyURL(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()

	if _, err := svc.SaveNotifyURL("discord://token@id"); err != nil {
		t.Fatal(err)
	}
	url, err := svc.NotifyURL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "discord://token@id" {
		t.Errorf("notify url = %q", url)
	}
}

func TestSettingService_DigestHour_SaveDigestHour(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()

	// 默认 DB 中 digest_hour 为 0（合法区间，直接返回，不回退）
	if h := svc.DigestHour(); h != 0 {
		t.Errorf("default digest hour = %d, want 0", h)
	}

	if _, err := svc.SaveDigestHour(20); err != nil {
		t.Fatal(err)
	}
	if h := svc.DigestHour(); h != 20 {
		t.Errorf("digest hour = %d, want 20", h)
	}

	// 非法范围
	if _, err := svc.SaveDigestHour(24); err == nil {
		t.Error("hour 24 should error")
	}
	if _, err := svc.SaveDigestHour(-1); err == nil {
		t.Error("hour -1 should error")
	}
}

func TestSettingService_Get_PrefsDefault(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()
	s, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if s.Prefs != "{}" {
		t.Errorf("default prefs = %q, want {}", s.Prefs)
	}
}

func TestSettingService_Update_Success(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()
	st, err := svc.Update([]int{1, 3, 5}, map[string]any{"theme": "dark"})
	if err != nil {
		t.Fatalf("update err: %v", err)
	}
	if st.Workdays != "1,3,5" {
		t.Errorf("workdays = %q, want 1,3,5", st.Workdays)
	}
	if st.Prefs != `{"theme":"dark"}` {
		t.Errorf("prefs = %q", st.Prefs)
	}
	// 二次读取应持久化
	again, _ := svc.Get()
	if again.Workdays != "1,3,5" {
		t.Errorf("persisted workdays = %q", again.Workdays)
	}
}

func TestSettingService_Update_Validation(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()
	if _, err := svc.Update(nil, nil); err == nil {
		t.Error("empty workdays should error")
	}
	if _, err := svc.Update([]int{0, 8}, nil); err == nil {
		t.Error("out-of-range weekday should error")
	}
}

func TestSettingService_DigestHour_InvalidFallback(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()
	// 写入非法值，DigestHour 应回退到 8
	st, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	st.DigestHour = 99
	if err := repositories.NewSettingRepo().Save(st); err != nil {
		t.Fatal(err)
	}
	if h := svc.DigestHour(); h != 8 {
		t.Errorf("invalid digest hour = %d, want fallback 8", h)
	}
}

