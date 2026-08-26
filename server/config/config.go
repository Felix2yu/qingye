package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config 应用配置
type Config struct {
	Port        string   // HTTP 监听端口
	DBPath      string   // SQLite 数据库文件路径
	UploadDir   string   // 照片上传目录
	CORSOrigins []string // 允许跨域的来源
	MaxUploadMB int64    // 单张照片大小上限（MB）
	PlantbookToken string // Plantbook API token（在线植物资料，留空则禁用在线匹配）
	WebDir     string     // 前端静态构建目录（为空则只提供 API，不托管页面）
}

// Load 从环境变量加载配置（支持 .env 文件）
func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:        getEnv("PORT", "8081"),
		DBPath:      getEnv("DB_PATH", "./data/qingye.db"),
		UploadDir:   getEnv("UPLOAD_DIR", "./uploads"),
		CORSOrigins: splitCSV(getEnv("CORS_ORIGINS", "http://localhost:5173")),
		MaxUploadMB: int64(parseInt(getEnv("MAX_UPLOAD_MB", "10"), 10)),
		PlantbookToken: getEnv("PLANTBOOK_TOKEN", ""),
		WebDir:         getEnv("WEB_DIR", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseInt(s string, fallback int64) int64 {
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return fallback
	}
	return n
}
