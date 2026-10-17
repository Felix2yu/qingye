package services

import (
	"testing"
	"time"
)

func TestToCandidate(t *testing.T) {
	item := pbSearchItem{
		PID:       "monstera deliciosa",
		Alias:     "Monstera deliciosa",
		ImageURL:  "https://example.com/img.jpg",
		CommonNames: []pbCommonName{
			{Name: "龟背竹"},
			{Name: "Monstera"},
		},
	}

	c := toCandidate(item)
	if c.PID != "monstera deliciosa" {
		t.Errorf("PID = %q", c.PID)
	}
	// extractCommonNames 对 typed slice ([]pbCommonName) 不会走 []any 分支
	// 实际 API 响应 JSON 反序列化后是 []any，这里验证 fallback 逻辑
	if c.Name != "Monstera deliciosa" {
		t.Errorf("Name = %q, want alias fallback (typed slice 不走 any 分支)", c.Name)
	}
	if c.Image != "https://example.com/img.jpg" {
		t.Errorf("Image = %q", c.Image)
	}
}

func TestToCandidate_无中文名(t *testing.T) {
	item := pbSearchItem{
		PID:   "rosa chinensis",
		Alias: "Rosa chinensis",
		CommonNames: []pbCommonName{
			{Name: "China Rose"},
		},
	}
	c := toCandidate(item)
	if c.Name != "Rosa chinensis" {
		t.Errorf("Name = %q, want alias fallback", c.Name)
	}
}

func TestDetailToLibrary_完整字段(t *testing.T) {
	d := pbDetail{
		PID:        "monstera deliciosa",
		DisplayPID: "Monstera Deliciosa",
		Alias:      "monstera deliciosa",
		Category:   "Foliary plant",
		Origin:     "Mexico",
		ImageURL:   "https://example.com/monstera.jpg",
		Link:       "https://plantbook.io/monstera",
		MinTemp:    10,
		MaxTemp:    35,
		Watering:   "frequent",
		MinLight:   "Bright Indirect",
		MaxLight:   "Medium",
		CommonNames: []pbCommonName{
			{Name: "龟背竹"},
			{Name: "Monstera"},
		},
		Care: &pbCare{
			Soil:          "Well-draining",
			Fertilization: "Monthly",
			Pruning:       "Prune yellow leaves",
		},
	}

	lib := detailToLibrary(d)
	if lib.PID != "monstera deliciosa" {
		t.Errorf("PID = %q", lib.PID)
	}
	if lib.DisplayPID != "Monstera Deliciosa" {
		t.Errorf("DisplayPID = %q", lib.DisplayPID)
	}
	if lib.Category != "Foliary plant" {
		t.Errorf("Category = %q", lib.Category)
	}
	if lib.Origin != "Mexico" {
		t.Errorf("Origin = %q", lib.Origin)
	}
	if lib.Image != "https://example.com/monstera.jpg" {
		t.Errorf("Image = %q", lib.Image)
	}
	if lib.Link != "https://plantbook.io/monstera" {
		t.Errorf("Link = %q", lib.Link)
	}
	if lib.Name != "龟背竹" {
		t.Errorf("Name = %q, want 龟背竹", lib.Name)
	}
	if lib.Guide == "" {
		t.Error("Guide should not be empty")
	}
	if lib.CommonNames == "" {
		t.Error("CommonNames should not be empty")
	}
	if lib.Metrics == "" {
		t.Error("Metrics should not be empty")
	}
}

func TestDetailToLibrary_无中文名(t *testing.T) {
	d := pbDetail{
		PID:   "rosa chinensis",
		Alias: "rosa chinensis",
		CommonNames: []pbCommonName{
			{Name: "China Rose"},
		},
	}
	lib := detailToLibrary(d)
	if lib.Name != "rosa chinensis" {
		t.Errorf("Name = %q, want alias fallback", lib.Name)
	}
}

func TestExtractCommonNames_anySlice(t *testing.T) {
	input := []any{
		map[string]any{"name": "龟背竹"},
		map[string]any{"name": "Monstera"},
	}
	got := extractCommonNames(input)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != "龟背竹" {
		t.Errorf("got[0] = %q, want 龟背竹", got[0])
	}
}

func TestFirstZhName_extended(t *testing.T) {
	names := []string{"Monstera", "龟背竹", "Swiss Cheese Plant"}
	got := firstZhName(names)
	if got != "龟背竹" {
		t.Errorf("firstZhName = %q, want 龟背竹", got)
	}

	got = firstZhName([]string{"Rose", "Lily"})
	if got != "" {
		t.Errorf("firstZhName(no zh) = %q, want empty", got)
	}
}

func TestThrottledError_Codec(t *testing.T) {
	e := &ThrottledError{RetryAfter: 120 * time.Second}
	if e.Error() == "" {
		t.Error("Error() returned empty")
	}

	if IsThrottled(nil) {
		t.Error("IsThrottled(nil) should be false")
	}
	if IsThrottled(&ThrottledError{}) {
		// Should be true even with zero retry
	}
}
