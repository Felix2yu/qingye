package services

import "testing"

func TestLookupLatin命中(t *testing.T) {
	tests := []struct {
		zh   string
		want string
	}{
		{"绿萝", "epipremnum aureum"},
		{"龟背竹", "monstera deliciosa"},
		{"仙人掌", "opuntia"},
		{"向日葵", "helianthus annuus"},
		{"薰衣草", "lavandula angustifolia"},
		{"黄金葛", "epipremnum aureum"}, // 别名
		{"龟背芋", "monstera deliciosa"},
	}
	for _, tt := range tests {
		t.Run(tt.zh, func(t *testing.T) {
			got, ok := lookupLatin(tt.zh)
			if !ok {
				t.Fatalf("lookupLatin(%q) 未命中", tt.zh)
			}
			if got != tt.want {
				t.Errorf("lookupLatin(%q) = %q, want %q", tt.zh, got, tt.want)
			}
		})
	}
}

func TestLookupLatin未命中(t *testing.T) {
	_, ok := lookupLatin("不存在的植物")
	if ok {
		t.Error("lookupLatin(\"不存在的植物\") 应返回 false")
	}
	_, ok = lookupLatin("")
	if ok {
		t.Error("lookupLatin(\"\") 应返回 false")
	}
}

func Test映射表无重复键(t *testing.T) {
	seen := make(map[string]bool)
	for _, a := range plantAliases {
		if seen[a.Zh] {
			t.Errorf("重复中文键: %q (拉丁: %s)", a.Zh, a.Latin)
		}
		seen[a.Zh] = true
	}
}

func Test映射表数据完整性(t *testing.T) {
	for i, a := range plantAliases {
		if a.Zh == "" {
			t.Errorf("条目 %d: Zh 为空", i)
		}
		if a.Latin == "" {
			t.Errorf("条目 %d (%s): Latin 为空", i, a.Zh)
		}
	}
	t.Logf("共 %d 条映射", len(plantAliases))
}
