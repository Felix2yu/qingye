package models

import "time"

// Plant 植物档案
type Plant struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null;size:100" json:"name"`
	Species   string    `gorm:"size:100" json:"species"`
	Photo     string    `gorm:"size:255" json:"photo"`
	RoomID    uint      `gorm:"index" json:"roomId"`
	Room      Room      `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Note      string    `gorm:"size:500" json:"note"`
	// 参考 hortusfox 扩充的属性字段（向后兼容，默认空）
	Location     string     `gorm:"size:100" json:"location"`      // 详细方位（如阳台左侧）
	AcquiredDate *time.Time `json:"acquiredDate"`                  // 获得/入库日期
	LightReq     string     `gorm:"size:50" json:"lightReq"`       // 光照需求（如 散射光/直射光/耐阴）
	Attributes   string     `gorm:"type:text" json:"attributes"`   // 结构化扩展（JSON：土壤/湿度偏好等）
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
