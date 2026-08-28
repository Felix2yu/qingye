package services

import (
	"os"
	"testing"

	"qingye/server/config"
	"qingye/server/models"
	"qingye/server/repositories"
)

func TestNewLibraryService(t *testing.T) {
	svc := NewLibraryService(&config.Config{})
	if svc == nil {
		t.Fatal("nil")
	}
}

func TestLibraryService_OnlineEnabled(t *testing.T) {
	// 未配置 token → 不可用
	svc := NewLibraryService(&config.Config{})
	if svc.OnlineEnabled() {
		t.Error("should be disabled without token")
	}
	// 配置了 token → 可用
	svc2 := NewLibraryService(&config.Config{PlantbookToken: "tok"})
	if !svc2.OnlineEnabled() {
		t.Error("should be enabled with token")
	}
}

func TestLibraryService_Search(t *testing.T) {
	setupTestDB(t)
	svc := NewLibraryService(&config.Config{})
	repositories.DB.Create(&models.PlantLibrary{PID: "local:x", Name: "测试植物", Guide: "guide"})
	list, err := svc.Search("测试")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("search = %d, want 1", len(list))
	}
}

// TestAliasToPIDNormalizesSpacedPid 守护「已同步排除」不命中的回归：
// Plantbook 返回的 pid 可能是带空格的原始学名(Monstera deliciosa)，而 buildPending
// 用下划线小写(monstera_deliciosa)比对。两侧必须经 aliasToPID 归一化到同一 key，
// 否则已同步条目永远不命中、每轮都重请求。
func TestAliasToPIDNormalizesSpacedPid(t *testing.T) {
	spaced := aliasToPID("Monstera deliciosa")   // 形如 Plantbook 返回的空格学名
	underscore := aliasToPID("monstera_deliciosa") // 形如比对侧 guess
	if spaced != "monstera_deliciosa" {
		t.Fatalf("aliasToPID(spaced) = %q, want monstera_deliciosa", spaced)
	}
	if spaced != underscore {
		t.Fatalf("归一化不一致：%q != %q，已同步项将无法命中排除", spaced, underscore)
	}
}

// TestLoadNotFoundMergesKnown 守护「已知未收录」跨环境生效：
// loadNotFound 必须合并硬编码的 knownNotFound，不依赖运行时文件是否存在。
func TestLoadNotFoundMergesKnown(t *testing.T) {
	// 准备一个临时文件，仅含一个条目，验证与 knownNotFound 合并
	f, err := os.CreateTemp(t.TempDir(), "sync_state_*.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"not_found":["foo_bar"]}`)
	f.Close()

	svc := NewLibraryService(&config.Config{})
	svc.syncStatePath = f.Name()
	set := svc.loadNotFound()
	for _, want := range append([]string{"foo_bar"}, knownNotFound...) {
		if !set[want] {
			t.Errorf("loadNotFound 缺少 %q（knownNotFound 或文件条目未合并）", want)
		}
	}
}
