package seed

import (
	"log"
	"time"

	"qingye/server/models"

	"gorm.io/gorm"
)

// Run 写入种子数据：植物资料库、初始设置，以及首次运行的演示数据
func Run(db *gorm.DB) error {
	// 1. 植物资料库
	var libCount int64
	db.Model(&models.PlantLibrary{}).Count(&libCount)
	if libCount == 0 {
		if err := db.Create(&libraryData).Error; err != nil {
			return err
		}
		log.Printf("已写入 %d 条植物资料库数据", len(libraryData))
	}

	// 2. 初始设置：默认工作日周一至周五
	var setCount int64
	db.Model(&models.UserSetting{}).Count(&setCount)
	if setCount == 0 {
		if err := db.Create(&models.UserSetting{Workdays: "1,2,3,4,5", Prefs: "{}"}).Error; err != nil {
			return err
		}
	}

	// 3. 首次运行演示数据（仅当没有任何植物时）
	var plantCount int64
	db.Model(&models.Plant{}).Count(&plantCount)
	if plantCount == 0 {
		if err := seedDemo(db); err != nil {
			return err
		}
		log.Println("已写入演示植物 / 任务数据")
	}
	return nil
}

var libraryData = []models.PlantLibrary{}

// seedDemo 写入演示用的房间、植物与任务，便于首次启动即见完整效果
func seedDemo(db *gorm.DB) error {
	rooms := []models.Room{
		{Name: "客厅", Sort: 1, Icon: "sofa"},
		{Name: "阳台", Sort: 2, Icon: "sun"},
		{Name: "卧室", Sort: 3, Icon: "bed"},
	}
	if err := db.Create(&rooms).Error; err != nil {
		return err
	}

	now := time.Now()
	plants := []models.Plant{
		{Name: "绿萝", Species: "Epipremnum aureum", RoomID: rooms[0].ID, Note: "客厅的常驻伙伴，见干见湿", CreatedAt: now, UpdatedAt: now},
		{Name: "龟背竹", Species: "Monstera deliciosa", RoomID: rooms[0].ID, Note: "喜欢明亮散射光", CreatedAt: now, UpdatedAt: now},
		{Name: "虎皮兰", Species: "Sansevieria trifasciata", RoomID: rooms[1].ID, Note: "耐旱选手，宁干勿湿", CreatedAt: now, UpdatedAt: now},
		{Name: "薄荷", Species: "Mentha spicata", RoomID: rooms[1].ID, Note: "喜光喜水，可采收泡茶", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&plants).Error; err != nil {
		return err
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tasks := []models.Task{
		{PlantID: plants[0].ID, Type: "water", Title: "浇水", IntervalDays: 3, NextDue: today.AddDate(0, 0, 1), Active: true, CreatedAt: now, UpdatedAt: now},
		{PlantID: plants[1].ID, Type: "water", Title: "浇水", IntervalDays: 4, NextDue: today, Active: true, CreatedAt: now, UpdatedAt: now},
		{PlantID: plants[2].ID, Type: "water", Title: "浇水", IntervalDays: 7, NextDue: today.AddDate(0, 0, -1), Active: true, CreatedAt: now, UpdatedAt: now},
		{PlantID: plants[3].ID, Type: "water", Title: "浇水", IntervalDays: 2, NextDue: today, Active: true, CreatedAt: now, UpdatedAt: now},
		{PlantID: plants[0].ID, Type: "fertilize", Title: "施肥", IntervalDays: 30, NextDue: today.AddDate(0, 0, 5), Active: true, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&tasks).Error; err != nil {
		return err
	}

	diaries := []models.PhotoDiary{
		{PlantID: plants[0].ID, Image: "", Caption: "新成员到家，先适应一周环境", TakenAt: now.AddDate(0, 0, -7), CreatedAt: now.AddDate(0, 0, -7)},
		{PlantID: plants[1].ID, Image: "", Caption: "换盆后长出了新叶片", TakenAt: now.AddDate(0, 0, -3), CreatedAt: now.AddDate(0, 0, -3)},
	}
	return db.Create(&diaries).Error
}
