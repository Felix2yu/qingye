package main

import (
	"log"
	"os"
	"path/filepath"

	"qingye/server/config"
	"qingye/server/models"
	"qingye/server/repositories"
	"qingye/server/router"
	"qingye/server/seed"
	"qingye/server/services"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.Load()

	// 确保数据与上传目录存在
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("创建上传目录失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// AutoMigrate：表结构随模型演进
	if err := db.AutoMigrate(
		&models.Room{}, &models.Plant{}, &models.Task{}, &models.TaskLog{},
		&models.PhotoDiary{}, &models.PlantLibrary{}, &models.UserSetting{}, &models.CareLog{},
		&models.WeatherLog{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 种子数据（资料库 + 初始设置 + 首次演示数据）
	if err := seed.Run(db); err != nil {
		log.Fatalf("写入种子数据失败: %v", err)
	}

	repositories.SetDB(db)

	// 后台天气轮询：依据实时天气调整养护策略（未配置 QWEATHER_KEY 时自动跳过）
	services.StartWeatherScheduler()

	// 后台通知：每日推送「今日养护任务」摘要（未配置通知地址时自动跳过）
	services.StartNotifier()

	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	// CSV 导入文件可能较大，放宽 multipart 内存上限（32MB）
	r.MaxMultipartMemory = 32 << 20
	router.Setup(r, cfg)

	log.Printf("青野后端已启动: http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
