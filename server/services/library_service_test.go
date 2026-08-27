package services

import (
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
