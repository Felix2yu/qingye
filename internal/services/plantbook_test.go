package services

import (
	"testing"
	"time"
)

// ---- firstNonEmpty ----

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name string
		vals []string
		want string
	}{
		{"第一个非空", []string{"a", "b", "c"}, "a"},
		{"跳过空串", []string{"", "b", ""}, "b"},
		{"跳过空白", []string{"  ", "\t", "c"}, "c"},
		{"全空", []string{"", "", ""}, ""},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstNonEmpty(tt.vals...); got != tt.want {
				t.Errorf("firstNonEmpty(%v) = %q, want %q", tt.vals, got, tt.want)
			}
		})
	}
}

// ---- isChinese ----

func TestIsChinese(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"绿萝", true},
		{"a", false},
		{"123", false},
		{"abc绿", true},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isChinese(tt.s); got != tt.want {
				t.Errorf("isChinese(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// ---- firstZhName ----

func TestFirstZhName(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		want  string
	}{
		{"有中文名", []string{"Swiss Cheese Plant", "龟背竹", "Monstera"}, "龟背竹"},
		{"无中文名", []string{"Rose", "Lily"}, ""},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstZhName(tt.names); got != tt.want {
				t.Errorf("firstZhName(%v) = %q, want %q", tt.names, got, tt.want)
			}
		})
	}
}

// ---- extractCommonNames ----

func TestExtractCommonNames(t *testing.T) {
	input := []any{
		map[string]any{"name": "龟背竹"},
		map[string]any{"name": "Monstera"},
		map[string]any{"name": "Swiss Cheese Plant"},
	}
	got := extractCommonNames(input)
	if len(got) != 3 {
		t.Fatalf("extractCommonNames len = %d, want 3", len(got))
	}
	if got[0] != "龟背竹" {
		t.Errorf("got[0] = %q, want 龟背竹", got[0])
	}
}

func TestExtractCommonNames_stringSlice(t *testing.T) {
	input := []any{"绿萝", "Golden Pothos"}
	got := extractCommonNames(input)
	if len(got) != 2 {
		t.Fatalf("extractCommonNames len = %d, want 2", len(got))
	}
	if got[0] != "绿萝" {
		t.Errorf("got[0] = %q, want 绿萝", got[0])
	}
}

// ---- wateringEnumText ----

func TestWateringEnumText(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"frequent", "频繁（保持土壤湿润）"},
		{"average", "适中（表土干透再浇）"},
		{"minimum", "极少（耐旱，少浇）"},
		{"none", "无需浇水"},
		{"Frequent", "频繁（保持土壤湿润）"}, // 大小写不敏感
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if got := wateringEnumText(tt.code); got != tt.want {
				t.Errorf("wateringEnumText(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

// ---- formatDuration ----

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30 秒"},
		{90 * time.Second, "1 分钟"},
		{3600 * time.Second, "1 小时"},
		{5400 * time.Second, "1 小时 30 分钟"},
		{7200 * time.Second, "2 小时"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatDuration(tt.d); got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

// ---- parseThrottle ----

func TestParseThrottle_fromHeader(t *testing.T) {
	e := parseThrottle("120", "")
	if e.RetryAfter != 120*time.Second {
		t.Errorf("RetryAfter = %v, want 120s", e.RetryAfter)
	}
}

func TestParseThrottle_fromBody(t *testing.T) {
	body := `{"error": "rate limit exceeded", "message": "Expected available in 300 seconds"}`
	e := parseThrottle("", body)
	if e.RetryAfter != 300*time.Second {
		t.Errorf("RetryAfter = %v, want 300s", e.RetryAfter)
	}
}

func TestParseThrottle_header优先(t *testing.T) {
	body := `Expected available in 999 seconds`
	e := parseThrottle("60", body)
	if e.RetryAfter != 60*time.Second {
		t.Errorf("RetryAfter = %v, want 60s (header should take priority)", e.RetryAfter)
	}
}

func TestParseThrottle_无数据(t *testing.T) {
	e := parseThrottle("", "")
	if e.RetryAfter != 0 {
		t.Errorf("RetryAfter = %v, want 0", e.RetryAfter)
	}
}

// ---- buildGuide ----

func TestBuildGuide_完整字段(t *testing.T) {
	d := pbDetail{
		Watering: "frequent",
		MinTemp:  10,
		MaxTemp:  35,
		MinLight: "Bright Indirect",
		MaxLight: "Medium",
		Care: &pbCare{
			Soil:          "Well-draining potting mix",
			Fertilization: "Monthly during growing season",
			Pruning:       "Remove yellow leaves",
		},
	}
	guide := buildGuide(d)
	if guide == "" {
		t.Fatal("buildGuide 返回空")
	}
	// 应包含所有有数据的行
	for _, want := range []string{"浇水：", "光照：", "温度：", "土壤：", "施肥：", "修剪："} {
		if !contains(guide, want) {
			t.Errorf("guide 缺少 %q\n完整内容:\n%s", want, guide)
		}
	}
}

func TestBuildGuide_care优先(t *testing.T) {
	d := pbDetail{
		Watering: "frequent", // 枚举回退
		Care: &pbCare{
			Watering: "Keep soil consistently moist",
		},
	}
	guide := buildGuide(d)
	if !contains(guide, "保持土壤持续湿润") {
		t.Errorf("care 文本应翻译为中文\nguide: %s", guide)
	}
}

func TestBuildGuide_空值不输出行(t *testing.T) {
	d := pbDetail{
		MinTemp: 10,
		MaxTemp: 30,
	}
	guide := buildGuide(d)
	// 只有温度行，不应有浇水/光照/土壤/施肥/修剪行
	if contains(guide, "浇水：") {
		t.Error("空浇水不应输出")
	}
	if contains(guide, "施肥：") {
		t.Error("空施肥不应输出")
	}
	if !contains(guide, "温度：10℃ ~ 30℃") {
		t.Errorf("应有温度行\nguide: %s", guide)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---- ThrottledError ----

func TestIsThrottled(t *testing.T) {
	if !IsThrottled(&ThrottledError{RetryAfter: 60 * time.Second}) {
		t.Error("IsThrottled 应返回 true")
	}
	if IsThrottled(nil) {
		t.Error("IsThrottled(nil) 应返回 false")
	}
}

func TestThrottledError_Error(t *testing.T) {
	e := &ThrottledError{RetryAfter: 120 * time.Second}
	msg := e.Error()
	if msg == "" {
		t.Error("Error() 返回空")
	}
}
