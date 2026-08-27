package models

// UserSetting 用户设置（工作日 / 偏好 / 通知）
type UserSetting struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Workdays   string `gorm:"size:50" json:"workdays"`   // "1,2,3,4,5"，1=周一 … 7=周日
	Prefs      string `gorm:"type:text" json:"prefs"`    // JSON 对象，扩展偏好
	NotifyURL  string `gorm:"size:500" json:"notifyURL"` // shoutrrr 通知地址，空表示不推送
	DigestHour int    `gorm:"default:8" json:"digestHour"` // 每日摘要推送小时（0-23，默认 8）
}
