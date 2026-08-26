package models

// PlantLibrary 植物资料库（内置养护指南）
type PlantLibrary struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	PID       string `gorm:"uniqueIndex;size:100" json:"pid"` // Plantbook 学名唯一键（留空表示本地条目）
	Name      string `gorm:"index;size:100" json:"name"`
	Alias     string `gorm:"size:100" json:"alias"`
	Guide     string `gorm:"type:text" json:"guide"`
	Image     string `gorm:"size:255" json:"image"`
	SyncedAt  int64  `gorm:"autoUpdateTime" json:"syncedAt"` // 最近一次在线同步时间戳
}
