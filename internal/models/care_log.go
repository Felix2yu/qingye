package models

import "time"

// CareLog 通用养护事件（独立于任务的一次动作）
// 覆盖"无任务来源的一次性养护动作"（如随手浇水、换盆记录），
// 与 task_logs 并存：task_logs 记录任务维度的完成/推迟，care_logs 作为统一养护时间线来源。
type CareLog struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	PlantID uint      `gorm:"index" json:"plantId"`
	Plant   Plant     `gorm:"foreignKey:PlantID" json:"plant,omitempty"`
	Type    string    `gorm:"size:20;index" json:"type"`  // water / fertilize / repot / other
	Title   string    `gorm:"size:100" json:"title"`
	At      time.Time `gorm:"index" json:"at"`
	Note    string    `gorm:"size:500" json:"note"`
	Source  string    `gorm:"size:20" json:"source"` // manual / task
	TaskID  *uint     `gorm:"index" json:"taskId,omitempty"`
}

// CareLog 来源
const (
	CareSourceManual = "manual"
	CareSourceTask   = "task"
)
