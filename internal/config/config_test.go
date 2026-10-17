package config

import (
	"os"
	"testing"
)

func TestLoad_defaults(t *testing.T) {
	// 清除环境变量以测试默认值
	keys := []string{"PORT", "DB_PATH", "UPLOAD_DIR", "CORS_ORIGINS", "MAX_UPLOAD_MB", "WEB_DIR"}
	origins := make(map[string]string)
	for _, k := range keys {
		origins[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for _, k := range keys {
			if v := origins[k]; v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	cfg := Load()
	if cfg.Port != "8081" {
		t.Errorf("default Port = %q, want %q", cfg.Port, "8081")
	}
	if cfg.DBPath != "./data/qingye.db" {
		t.Errorf("default DBPath = %q, want %q", cfg.DBPath, "./data/qingye.db")
	}
	if cfg.UploadDir != "./uploads" {
		t.Errorf("default UploadDir = %q, want %q", cfg.UploadDir, "./uploads")
	}
	if cfg.MaxUploadMB != 10 {
		t.Errorf("default MaxUploadMB = %d, want %d", cfg.MaxUploadMB, 10)
	}
	if cfg.WebDir != "" {
		t.Errorf("default WebDir = %q, want empty", cfg.WebDir)
	}
}

func TestLoad_customEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("CORS_ORIGINS", "http://a.com, http://b.com")
	os.Setenv("MAX_UPLOAD_MB", "20")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("CORS_ORIGINS")
		os.Unsetenv("MAX_UPLOAD_MB")
	}()

	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("custom Port = %q, want %q", cfg.Port, "9090")
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Errorf("custom CORSOrigins len = %d, want 2", len(cfg.CORSOrigins))
	}
	if cfg.CORSOrigins[0] != "http://a.com" {
		t.Errorf("CORSOrigins[0] = %q", cfg.CORSOrigins[0])
	}
	if cfg.MaxUploadMB != 20 {
		t.Errorf("custom MaxUploadMB = %d, want 20", cfg.MaxUploadMB)
	}
}

func TestLoad_emptyCSV(t *testing.T) {
	os.Setenv("CORS_ORIGINS", " , , ")
	defer os.Unsetenv("CORS_ORIGINS")

	cfg := Load()
	if len(cfg.CORSOrigins) != 0 {
		t.Errorf("empty CSV CORSOrigins = %v, want empty", cfg.CORSOrigins)
	}
}
