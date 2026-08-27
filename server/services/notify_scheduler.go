package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"qingye/server/repositories"
)

// rescheduleCh 用于在设置保存后让调度器按最新小时重新计算下一次推送时间。
var rescheduleCh = make(chan struct{}, 1)

// RescheduleNotifier 触发每日摘要调度器按最新配置重排下一次推送（设置保存后调用）。
func RescheduleNotifier() {
	select {
	case rescheduleCh <- struct{}{}:
	default:
	}
}

// StartNotifier 启动通知后台任务：每天在用户配置的小时（默认 08:00）推送一次「今日养护任务」摘要。
// 未配置通知地址时仍会按点判断（无任务则不发）；按本地日期每天最多推送一次（重启不会重复推送同一天）。
func StartNotifier() {
	svc := NewNotifyService()
	go func() {
		// 等待数据库就绪
		time.Sleep(3 * time.Second)
		lastDate := ""

		sendNow := func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[notify] 每日摘要异常: %v", r)
				}
			}()
			today := time.Now().Format("2006-01-02")
			if lastDate == today {
				return
			}
			msg, ok := buildDailyDigest()
			if !ok {
				// 今日无任务，仍标记为已处理，避免反复判断
				lastDate = today
				return
			}
			if err := svc.Send(msg); err == nil {
				lastDate = today
			}
		}

		// 计算到下一次推送（按当前配置小时）的等待时长
		wait := func() time.Duration {
			return time.Until(nextOccurrence(digestHour()))
		}

		log.Printf("[notify] 每日摘要调度已启动，下次推送: %s", nextOccurrence(digestHour()).Format("2006-01-02 15:04"))
		timer := time.NewTimer(wait())
		for {
			select {
			case <-timer.C:
				sendNow()
				timer.Reset(wait())
			case <-rescheduleCh:
				// 配置变更：按新时间重排下一次推送
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				log.Printf("[notify] 已按新时间重排，下次推送: %s", nextOccurrence(digestHour()).Format("2006-01-02 15:04"))
				timer.Reset(wait())
			}
		}
	}()
}

// digestHour 读取配置的通知小时，非法值回退到 8
func digestHour() int {
	h := NewSettingService().DigestHour()
	if h < 0 || h > 23 {
		return 8
	}
	return h
}

// nextOccurrence 计算当天/次日指定小时(:00:00)的下一次时刻（使用服务器本地时区）
func nextOccurrence(hour int) time.Time {
	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !target.After(now) {
		target = target.Add(24 * time.Hour)
	}
	return target
}

// buildDailyDigest 生成今日养护任务摘要；无任务返回 ok=false
func buildDailyDigest() (string, bool) {
	tasks, err := NewTaskService().Today()
	if err != nil || len(tasks) == 0 {
		return "", false
	}

	// 植物名映射（一次查询，用于拼装「植物名：任务」）
	plants, _ := repositories.NewPlantRepo().List(0)
	nameOf := make(map[uint]string, len(plants))
	for _, p := range plants {
		nameOf[p.ID] = p.Name
	}

	var b strings.Builder
	b.WriteString("🌿 青野集 · 今日养护任务\n")
	for i, t := range tasks {
		if i >= 20 {
			b.WriteString(fmt.Sprintf("…等共 %d 项\n", len(tasks)))
			break
		}
		name := nameOf[t.PlantID]
		if name == "" {
			name = "未知植物"
		}
		b.WriteString(fmt.Sprintf("%s %s：%s\n", taskTypeEmoji(t.Type), name, t.Title))
	}
	return b.String(), true
}
