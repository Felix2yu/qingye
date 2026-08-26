package models

// PlantLibrary 植物资料库（内置养护指南）
type PlantLibrary struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	PID         string `gorm:"uniqueIndex;size:100" json:"pid"` // Plantbook 学名唯一键（留空表示本地条目）
	DisplayPID  string `gorm:"size:100" json:"displayPid"`      // 学名展示形式（Title Case）
	Name        string `gorm:"index;size:100" json:"name"`
	Alias       string `gorm:"size:100" json:"alias"`
	Category    string `gorm:"size:100" json:"category"`   // 植物类别（Plantbook 分类）
	Origin      string `gorm:"size:100" json:"origin"`     // 原产地
	CommonNames string `gorm:"type:text" json:"commonNames"` // 全部常见名（JSON 字符串数组）
	Guide       string `gorm:"type:text" json:"guide"`       // 中文养护文本指南
	Metrics     string `gorm:"type:text" json:"metrics"`     // 结构化环境阈值（JSON，见 services.PlantMetrics）
	Image       string `gorm:"size:255" json:"image"`
	Link        string `gorm:"size:255" json:"link"`           // Plantbook 在线详情页
	SyncedAt    int64  `gorm:"autoUpdateTime" json:"syncedAt"` // 最近一次在线同步时间戳
}
