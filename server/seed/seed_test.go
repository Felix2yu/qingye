package seed

import (
	"testing"

	"qingye/server/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRun_WritesDemoData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.PlantLibrary{}, &models.UserSetting{}, &models.Room{},
		&models.Plant{}, &models.Task{}, &models.PhotoDiary{},
	); err != nil {
		t.Fatal(err)
	}

	if err := Run(db); err != nil {
		t.Fatal(err)
	}

	var rooms, plants, tasks, diaries, libs, sets int64
	db.Model(&models.Room{}).Count(&rooms)
	db.Model(&models.Plant{}).Count(&plants)
	db.Model(&models.Task{}).Count(&tasks)
	db.Model(&models.PhotoDiary{}).Count(&diaries)
	db.Model(&models.PlantLibrary{}).Count(&libs)
	db.Model(&models.UserSetting{}).Count(&sets)

	if rooms != 3 {
		t.Errorf("rooms = %d, want 3", rooms)
	}
	if plants != 4 {
		t.Errorf("plants = %d, want 4", plants)
	}
	if tasks != 5 {
		t.Errorf("tasks = %d, want 5", tasks)
	}
	if diaries < 2 {
		t.Errorf("diaries = %d, want >= 2", diaries)
	}
	if libs != 10 {
		t.Errorf("libs = %d, want 10", libs)
	}
	if sets != 1 {
		t.Errorf("settings = %d, want 1", sets)
	}

	// 幂等：重复运行不应重复写入演示数据
	if err := Run(db); err != nil {
		t.Fatal(err)
	}
	var plants2 int64
	db.Model(&models.Plant{}).Count(&plants2)
	if plants2 != 4 {
		t.Errorf("after re-run plants = %d, want 4", plants2)
	}
}
