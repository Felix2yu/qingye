package models

import "time"

// Task 重复养护任务规则（核心模型）
// 不写死所有实例，而是以「规则 + 上次完成时间」在查询时计算下一次到期。
type Task struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	PlantID      uint       `gorm:"index" json:"plantId"`
	Plant        Plant      `gorm:"foreignKey:PlantID" json:"plant,omitempty"`
	Type         string     `gorm:"index;size:20" json:"type"` // water / fertilize / repot
	Title        string     `gorm:"size:100" json:"title"`
	IntervalDays int        `gorm:"not null" json:"intervalDays"` // 当前生效周期（天气调整后可能变化）
	BaseIntervalDays int    `gorm:"not null;default:0" json:"baseIntervalDays"` // 原始周期（天气策略恢复基准）
	LastDoneAt   *time.Time `json:"lastDoneAt"`
	NextDue      time.Time  `gorm:"index" json:"nextDue"`
	Active       bool       `gorm:"default:true" json:"active"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}
