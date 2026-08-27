package services

import (
	"testing"
	"time"
)

func TestDigestHour(t *testing.T) {
	setupTestDB(t)
	// 默认 DB 中 digest_hour 为 0（合法区间，直接返回）
	if h := digestHour(); h != 0 {
		t.Errorf("digestHour = %d, want 0", h)
	}
}

func TestNextOccurrence(t *testing.T) {
	now := time.Now()
	// 未来小时 → 今天该时刻（应晚于现在）
	later := (now.Hour() + 2) % 24
	occ := nextOccurrence(later)
	if !occ.After(now) {
		t.Errorf("nextOccurrence(%d)=%v should be after now", later, occ)
	}
	// 过去小时 → 明天该时刻
	earlier := (now.Hour() + 22) % 24
	occ2 := nextOccurrence(earlier)
	if !occ2.After(now) {
		t.Errorf("nextOccurrence(%d) past should be tomorrow, got %v", earlier, occ2)
	}
	// 恰好当前小时但分钟已过 → 明天
	sameHour := now.Hour()
	occ3 := nextOccurrence(sameHour)
	if !occ3.After(now) {
		t.Errorf("nextOccurrence(same hour, past) should be tomorrow, got %v", occ3)
	}
}

func TestBuildDailyDigest_NoTasks(t *testing.T) {
	setupTestDB(t)
	msg, ok := buildDailyDigest()
	if ok {
		t.Errorf("expected ok=false with no tasks, got msg=%q", msg)
	}
}

func TestRescheduleNotifier(t *testing.T) {
	// 不应 panic，仅向 channel 发送（满则丢弃）
	RescheduleNotifier()
	RescheduleNotifier()
}
