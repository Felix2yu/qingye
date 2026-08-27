package models

// Room 房间 / 分组
type Room struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"not null;size:50" json:"name"`
	Sort      int    `json:"sort"`
	IsOutdoor bool   `json:"isOutdoor"` // 室外（阳台/花园等）标识，用于降雨推迟等策略
	Icon      string `gorm:"size:50" json:"icon"` // 房间图标（对应前端 Icon.svelte 中的 Lucide 图标名）
}

