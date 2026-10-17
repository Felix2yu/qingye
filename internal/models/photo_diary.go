package models

import "time"

// PhotoDiary 照片日记
type PhotoDiary struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PlantID   uint      `gorm:"index" json:"plantId"`
	Plant     Plant     `gorm:"foreignKey:PlantID" json:"plant,omitempty"`
	Image     string    `gorm:"size:255" json:"image"`
	Caption   string    `gorm:"size:500" json:"caption"`
	TakenAt   time.Time `gorm:"index" json:"takenAt"`
	CreatedAt time.Time `json:"createdAt"`
}
