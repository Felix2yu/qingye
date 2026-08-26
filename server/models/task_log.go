package models

import "time"

// TaskLog 任务动作日志（完成 / 推迟）
type TaskLog struct {
	ID     uint      `gorm:"primaryKey" json:"id"`
	TaskID uint      `gorm:"index" json:"taskId"`
	Action string    `gorm:"size:20" json:"action"` // done / postpone
	At     time.Time `json:"at"`
	Note   string    `gorm:"size:255" json:"note"`
}
