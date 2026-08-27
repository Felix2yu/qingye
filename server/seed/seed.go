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

var libraryData = []models.PlantLibrary{
	{
		PID:   "local:绿萝",
		Name:  "绿萝",
		Alias: "黄金葛",
		Guide: "浇水：见干见湿，土表发白再浇透；光照：耐阴，明亮散射光最佳；养护：每月施一次稀薄液肥，垂枝过长可修剪扦插。",
	},
	{
		PID:   "local:龟背竹",
		Name:  "龟背竹",
		Alias: "蓬莱蕉",
		Guide: "浇水：保持土壤微润，避免积水；光照：喜欢明亮的散射光，忌暴晒；养护：叶面常喷水增湿，气生根可引导入盆。",
	},
	{
		PID:   "local:虎皮兰",
		Name:  "虎皮兰",
		Alias: "虎尾兰",
		Guide: "浇水：极耐旱，盆土干透再浇，冬季断水；光照：全日照到半阴均可；养护：浇水过多易烂根，宁干勿湿。",
	},
	{
		PID:   "local:琴叶榕",
		Name:  "琴叶榕",
		Alias: "Fiddle-leaf fig",
		Guide: "浇水：见干见湿，浇透不积水；光照：需要充足明亮的散射光；养护：叶片怕灰尘，定期擦拭；喜温暖湿润环境。",
	},
	{
		PID:   "local:吊兰",
		Name:  "吊兰",
		Alias: "Chlorophytum",
		Guide: "浇水：保持盆土湿润，忌积水；光照：半阴环境生长良好；养护：空气干燥时叶尖易枯，经常喷水；易长走茎小苗。",
	},
	{
		PID:   "local:多肉植物",
		Name:  "多肉植物",
		Alias: "Succulents",
		Guide: "浇水：严格控水，干透浇透，春秋生长季可勤一些；光照：喜充足日照，光照不足易徒长；养护：夏季高温休眠，注意遮阴通风。",
	},
	{
		PID:   "local:薄荷",
		Name:  "薄荷",
		Alias: "Mentha",
		Guide: "浇水：喜水，保持土壤湿润；光照：喜充足阳光；养护：勤打顶促进分枝，长势旺时可多次采收嫩叶。",
	},
	{
		PID:   "local:富贵竹",
		Name:  "富贵竹",
		Alias: "转运竹",
		Guide: "浇水：水培时水位不超过根部 1/3，水变浑浊即换；光照：喜散射光，忌强光直射；养护：避免水质过肥，可滴几滴营养液。",
	},
	{
		PID:   "local:散尾葵",
		Name:  "散尾葵",
		Alias: "散尾竹",
		Guide: "浇水：保持盆土微润，叶面勤喷水；光照：耐半阴，忌烈日直射；养护：喜高温高湿，冬季注意保暖防寒。",
	},
	{
		PID:   "local:鹅掌柴",
		Name:  "鹅掌柴",
		Alias: "鸭脚木",
		Guide: "浇水：见干见湿；光照：耐阴也耐半日照；养护：生长迅速，需定期修剪塑形；室内空气干燥时向叶片喷雾。",
	},
}

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
