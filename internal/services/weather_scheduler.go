package services

import (
	"log"
	"time"
)

// StartWeatherScheduler 启动后台天气轮询任务。
// 每 pollMinutes 分钟执行一次策略调整；未配置 QWEATHER_KEY 或未启用时自动跳过。
// 首次启动延迟 3 秒再执行，避免与初始化抢占。
func StartWeatherScheduler() {
	svc := NewWeatherService()
	go func() {
		// 等待数据库就绪
		time.Sleep(3 * time.Second)
		runOnce := func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[weather] 轮询异常: %v", r)
				}
			}()
			svc.Poll()
		}
		runOnce()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		var lastPoll time.Time
		for range ticker.C {
			cfg := svc.LoadConfig()
			interval := cfg.PollMinutes
			if interval < 1 {
				interval = 60
			}
			if time.Since(lastPoll) >= time.Duration(interval)*time.Minute {
				runOnce()
				lastPoll = time.Now()
			}
		}
	}()
}
