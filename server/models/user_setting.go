package models

// UserSetting 用户设置（工作日 / 偏好）
type UserSetting struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Workdays string `gorm:"size:50" json:"workdays"` // "1,2,3,4,5"，1=周一 … 7=周日
	Prefs    string `gorm:"type:text" json:"prefs"`  // JSON 对象，扩展偏好
}
