package services

import (
	"testing"
	"time"

	"qingye/server/models"
	"qingye/server/repositories"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(
		&models.Plant{},
		&models.Room{},
		&models.Task{},
		&models.TaskLog{},
		&models.CareLog{},
		&models.PhotoDiary{},
		&models.PlantLibrary{},
		&models.UserSetting{},
		&models.WeatherLog{},
	)
	repositories.SetDB(db)
	return db
}

// ---- PlantService ----

func TestPlantService_Create_验证(t *testing.T) {
	setupTestDB(t)
	svc := NewPlantService()

	_, err := svc.Create(&models.Plant{})
	if err == nil || err.Error() != "植物名称不能为空" {
		t.Errorf("empty name: err = %v", err)
	}
}

func TestPlantService_Create_正常(t *testing.T) {
	setupTestDB(t)
	svc := NewPlantService()

	p, err := svc.Create(&models.Plant{Name: "绿萝"})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == 0 || p.Name != "绿萝" {
		t.Errorf("Create: id=%d name=%q", p.ID, p.Name)
	}
}

func TestPlantService_Update(t *testing.T) {
	setupTestDB(t)
	svc := NewPlantService()

	p, _ := svc.Create(&models.Plant{Name: "绿萝"})

	// 空名称
	_, err := svc.Update(&models.Plant{ID: p.ID, Name: ""})
	if err == nil {
		t.Error("empty name should error")
	}

	// 正常更新
	p.Name = "黄金葛"
	p, err = svc.Update(p)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "黄金葛" {
		t.Errorf("name = %q", p.Name)
	}
}

func TestPlantService_Delete(t *testing.T) {
	setupTestDB(t)
	svc := NewPlantService()

	p, _ := svc.Create(&models.Plant{Name: "Test"})

	// 删除不存在
	err := svc.Delete(99999)
	if err == nil {
		t.Error("delete nonexistent should error")
	}

	// 正常删除
	if err := svc.Delete(p.ID); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Get(p.ID)
	if err == nil {
		t.Error("should error after delete")
	}
}

func TestPlantService_Delete_级联(t *testing.T) {
	db := setupTestDB(t)
	svc := NewPlantService()

	p, _ := svc.Create(&models.Plant{Name: "Test"})

	// 创建关联任务和日记
	db.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: time.Now(), Active: true})
	db.Create(&models.PhotoDiary{PlantID: p.ID, Image: "a.jpg", TakenAt: time.Now()})
	db.Create(&models.CareLog{PlantID: p.ID, Type: "water", At: time.Now()})

	svc.Delete(p.ID)

	var count int64
	db.Model(&models.Task{}).Where("plant_id = ?", p.ID).Count(&count)
	if count != 0 {
		t.Errorf("tasks after delete: %d, want 0", count)
	}
	db.Model(&models.PhotoDiary{}).Where("plant_id = ?", p.ID).Count(&count)
	if count != 0 {
		t.Errorf("diaries after delete: %d, want 0", count)
	}
}

func TestPlantService_List(t *testing.T) {
	setupTestDB(t)
	svc := NewPlantService()

	svc.Create(&models.Plant{Name: "A"})
	svc.Create(&models.Plant{Name: "B"})

	list, err := svc.List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("List = %d, want 2", len(list))
	}
}

// ---- RoomService ----

func TestRoomService_Create_验证(t *testing.T) {
	setupTestDB(t)
	svc := NewRoomService()

	_, err := svc.Create(&models.Room{})
	if err == nil || err.Error() != "房间名称不能为空" {
		t.Errorf("empty name: err = %v", err)
	}
}

func TestRoomService_Create_正常(t *testing.T) {
	setupTestDB(t)
	svc := NewRoomService()

	r, err := svc.Create(&models.Room{Name: "客厅"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "客厅" {
		t.Errorf("name = %q", r.Name)
	}
}

func TestRoomService_Update(t *testing.T) {
	setupTestDB(t)
	svc := NewRoomService()

	r, _ := svc.Create(&models.Room{Name: "客厅"})

	// 空名称
	_, err := svc.Update(&models.Room{ID: r.ID, Name: ""})
	if err == nil {
		t.Error("empty name should error")
	}

	r.Name = "新客厅"
	r, err = svc.Update(r)
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "新客厅" {
		t.Errorf("name = %q", r.Name)
	}
}

func TestRoomService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoomService()

	r, _ := svc.Create(&models.Room{Name: "客厅"})

	// 有植物时不能删除
	db.Create(&models.Plant{Name: "P", RoomID: r.ID})
	err := svc.Delete(r.ID)
	if err == nil {
		t.Error("room with plants should not delete")
	}

	// 无植物时可以删除
	db.Where("room_id = ?", r.ID).Delete(&models.Plant{})
	if err := svc.Delete(r.ID); err != nil {
		t.Fatal(err)
	}
}

func TestRoomService_ListWithStats(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoomService()

	r, _ := svc.Create(&models.Room{Name: "客厅"})
	db.Create(&models.Plant{Name: "A", RoomID: r.ID})
	db.Create(&models.Plant{Name: "B", RoomID: r.ID})

	list, err := svc.ListWithStats()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("ListWithStats = %d, want 1", len(list))
	}
	if list[0]["count"].(int) != 2 {
		t.Errorf("count = %v, want 2", list[0]["count"])
	}
}

// ---- TaskService ----

func TestTaskService_Create_验证(t *testing.T) {
	setupTestDB(t)
	svc := NewTaskService()

	// 空植物
	_, err := svc.Create(&models.Task{Type: "water", IntervalDays: 7})
	if err == nil || err.Error() != "植物不能为空" {
		t.Errorf("empty plant: err = %v", err)
	}

	// 周期无效
	_, err = svc.Create(&models.Task{PlantID: 1, Type: "water", IntervalDays: 0})
	if err == nil || err.Error() != "任务周期必须大于 0 天" {
		t.Errorf("zero interval: err = %v", err)
	}
}

func TestTaskService_Create_正常(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)

	task, err := svc.Create(&models.Task{
		PlantID:      p.ID,
		Type:         "water",
		IntervalDays: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.ID == 0 || task.Title != "浇水" {
		t.Errorf("Create: id=%d title=%q", task.ID, task.Title)
	}
	if task.BaseIntervalDays != 7 {
		t.Errorf("BaseIntervalDays = %d", task.BaseIntervalDays)
	}
}

func TestTaskService_Done(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)
	task, _ := svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7})

	// 完成任务
	result, err := svc.Done(task.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.LastDoneAt == nil {
		t.Error("LastDoneAt should be set")
	}

	// 验证 TaskLog 已创建
	var count int64
	db.Model(&models.TaskLog{}).Where("task_id = ?", task.ID).Count(&count)
	if count != 1 {
		t.Errorf("TaskLog count = %d, want 1", count)
	}
}

func TestTaskService_Postpone(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)
	task, _ := svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7})

	result, err := svc.Postpone(task.ID, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	// nextDue 应推迟 3 天
	due := result.NextDue
	if due.Before(time.Now().AddDate(0, 0, 2)) {
		t.Errorf("nextDue too early: %v", due)
	}
}

func TestTaskService_List_Today(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)

	svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: time.Now()})
	svc.Create(&models.Task{PlantID: p.ID, Type: "fertilize", IntervalDays: 30, NextDue: time.Now().AddDate(0, 0, 30)})

	// List
	list, _ := svc.List("", false, 0)
	if len(list) != 2 {
		t.Fatalf("List = %d, want 2", len(list))
	}

	// Today
	today, _ := svc.Today()
	if len(today) != 1 {
		t.Fatalf("Today = %d, want 1", len(today))
	}
}

// ---- CareLogService ----

func TestCareLogService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := NewCareLogService()

	p := &models.Plant{Name: "X"}
	db.Create(p)
	db.Create(&models.CareLog{PlantID: p.ID, Type: "water", At: time.Now()})
	db.Create(&models.CareLog{PlantID: p.ID, Type: "fertilize", At: time.Now()})

	list, err := svc.List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("List = %d", len(list))
	}

	list, _ = svc.ListByPlant(p.ID)
	if len(list) != 2 {
		t.Fatalf("ListByPlant = %d", len(list))
	}
}

// ---- DiaryService ----

func TestDiaryService_Create_验证(t *testing.T) {
	setupTestDB(t)
	svc := NewDiaryService()

	// 空植物
	_, err := svc.Create(&models.PhotoDiary{Image: "a.jpg"})
	if err == nil || err.Error() != "植物不能为空" {
		t.Errorf("empty plant: err = %v", err)
	}

	// 空图片
	_, err = svc.Create(&models.PhotoDiary{PlantID: 1})
	if err == nil || err.Error() != "图片不能为空" {
		t.Errorf("empty image: err = %v", err)
	}
}

func TestDiaryService_Create_植物不存在(t *testing.T) {
	setupTestDB(t)
	svc := NewDiaryService()

	_, err := svc.Create(&models.PhotoDiary{PlantID: 99999, Image: "a.jpg"})
	if err == nil || err.Error() != "植物不存在" {
		t.Errorf("nonexistent plant: err = %v", err)
	}
}

func TestDiaryService_Create_正常(t *testing.T) {
	db := setupTestDB(t)
	svc := NewDiaryService()

	p := &models.Plant{Name: "X"}
	db.Create(p)

	d, err := svc.Create(&models.PhotoDiary{PlantID: p.ID, Image: "a.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if d.ID == 0 {
		t.Error("ID should be populated")
	}
}

func TestDiaryService_Page(t *testing.T) {
	db := setupTestDB(t)
	svc := NewDiaryService()

	p := &models.Plant{Name: "X"}
	db.Create(p)
	for i := 0; i < 5; i++ {
		db.Create(&models.PhotoDiary{PlantID: p.ID, Image: "a.jpg", TakenAt: time.Now()})
	}

	list, total, err := svc.Page(0, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(list) != 3 {
		t.Errorf("page1 = %d, want 3", len(list))
	}

	list, total, _ = svc.Page(0, 2, 3)
	if len(list) != 2 {
		t.Errorf("page2 = %d, want 2", len(list))
	}
}

// ---- SettingService ----

func TestSettingService_Get_Update(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()

	// 获取默认
	s, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if s.Workdays != "1,2,3,4,5" {
		t.Errorf("default workdays = %q", s.Workdays)
	}

	// 更新
	s, err = svc.Update([]int{1, 3, 5}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Workdays != "1,3,5" {
		t.Errorf("workdays = %q", s.Workdays)
	}
}

func TestSettingService_Update_验证(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()

	// 空工作日
	_, err := svc.Update(nil, nil)
	if err == nil {
		t.Error("empty workdays should error")
	}

	// 超出范围
	_, err = svc.Update([]int{0, 8}, nil)
	if err == nil {
		t.Error("out of range should error")
	}
}

func TestSettingService_IsWorkday(t *testing.T) {
	setupTestDB(t)
	svc := NewSettingService()

	// 默认周一至周五
	monday := time.Date(2025, 3, 17, 0, 0, 0, 0, time.UTC) // 周一
	isWork, _ := svc.IsWorkday(monday)
	if !isWork {
		t.Error("Monday should be workday")
	}

	sunday := time.Date(2025, 3, 16, 0, 0, 0, 0, time.UTC) // 周日
	isWork, _ = svc.IsWorkday(sunday)
	if isWork {
		t.Error("Sunday should not be workday")
	}
}

// ---- TaskService 补充 ----

func TestTaskService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)
	task, _ := svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7})

	task.Title = "新标题"
	task.IntervalDays = 14
	if err := svc.Update(task); err != nil {
		t.Fatal(err)
	}
	got, _ := repositories.NewTaskRepo().Get(task.ID)
	if got.Title != "新标题" || got.IntervalDays != 14 {
		t.Errorf("after update: title=%q interval=%d", got.Title, got.IntervalDays)
	}
}

func TestTaskService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)
	task, _ := svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7})

	if err := svc.Delete(task.ID); err != nil {
		t.Fatal(err)
	}
	_, err := repositories.NewTaskRepo().Get(task.ID)
	if err == nil {
		t.Error("should error after delete")
	}
}

func TestTaskService_Upcoming(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)

	// 今天到期
	svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: time.Now()})
	// 3天后到期
	svc.Create(&models.Task{PlantID: p.ID, Type: "fertilize", IntervalDays: 30, NextDue: time.Now().AddDate(0, 0, 3)})
	// 20天后到期（在30天窗口内）
	svc.Create(&models.Task{PlantID: p.ID, Type: "repot", IntervalDays: 90, NextDue: time.Now().AddDate(0, 0, 20)})

	// Upcoming(7) 包含今天 + 3天后的
	upcoming, err := svc.Upcoming(7)
	if err != nil {
		t.Fatal(err)
	}
	if len(upcoming) != 2 {
		t.Fatalf("Upcoming(7d) = %d, want 2", len(upcoming))
	}

	// Upcoming(30) 包含三个
	upcoming, _ = svc.Upcoming(30)
	if len(upcoming) != 3 {
		t.Fatalf("Upcoming(30d) = %d, want 3", len(upcoming))
	}
}

func TestTaskService_RecordManual(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)

	log, err := svc.RecordManual(p.ID, "water", "浇水", "手动浇水", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if log.ID == 0 {
		t.Error("log ID should be populated")
	}
	if log.Source != "manual" {
		t.Errorf("source = %q, want manual", log.Source)
	}
}

func TestTaskService_RecordManual_植物不存在(t *testing.T) {
	setupTestDB(t)
	svc := NewTaskService()

	_, err := svc.RecordManual(99999, "water", "浇水", "", time.Now())
	if err == nil {
		t.Error("nonexistent plant should error")
	}
}

func TestTaskService_History(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)
	task, _ := svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7})

	// 完成一次
	svc.Done(task.ID, "")

	// 获取历史
	logs, err := svc.History(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("History = %d, want 1", len(logs))
	}
}

func TestTaskService_List_ByType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()

	p := &models.Plant{Name: "X"}
	db.Create(p)

	svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: time.Now()})
	svc.Create(&models.Task{PlantID: p.ID, Type: "fertilize", IntervalDays: 30, NextDue: time.Now()})

	list, _ := svc.List("water", false, 0)
	if len(list) != 1 {
		t.Fatalf("List(water) = %d, want 1", len(list))
	}

	list, _ = svc.List("", false, p.ID)
	if len(list) != 2 {
		t.Fatalf("List(byPlant) = %d, want 2", len(list))
	}
}

// ---- ImportService PreviewTasks ----

func TestImportService_PreviewTasks(t *testing.T) {
	db := setupTestDB(t)
	svc := NewImportService()

	p := &models.Plant{Name: "绿萝"}
	db.Create(p)

	csv := "plant,type,intervalDays,title\n绿萝,water,7,浇水"
	preview, err := svc.PreviewTasks(csv)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Valid != 1 {
		t.Errorf("valid = %d, want 1", preview.Valid)
	}
}

func TestImportService_PreviewTasks_植物不存在(t *testing.T) {
	setupTestDB(t)
	svc := NewImportService()

	csv := "plant,type,intervalDays\n不存在,water,7"
	preview, err := svc.PreviewTasks(csv)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Invalid != 1 {
		t.Errorf("invalid = %d, want 1", preview.Invalid)
	}
}

func TestImportService_PreviewTasks_未知类型(t *testing.T) {
	db := setupTestDB(t)
	svc := NewImportService()

	p := &models.Plant{Name: "X"}
	db.Create(p)

	csv := "plant,type,intervalDays\nX,unknown,7"
	preview, err := svc.PreviewTasks(csv)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Invalid != 1 {
		t.Errorf("invalid = %d, want 1", preview.Invalid)
	}
}

func TestImportService_ensureRoom(t *testing.T) {
	db := setupTestDB(t)
	svc := NewImportService()

	// 不存在则创建
	room, err := svc.ensureRoom("新房间")
	if err != nil {
		t.Fatal(err)
	}
	if room.Name != "新房间" {
		t.Errorf("name = %q", room.Name)
	}

	// 已存在则复用
	room2, err := svc.ensureRoom("新房间")
	if err != nil {
		t.Fatal(err)
	}
	if room2.ID != room.ID {
		t.Error("should return same room")
	}

	var count int64
	db.Model(&models.Room{}).Where("name = ?", "新房间").Count(&count)
	if count != 1 {
		t.Errorf("room count = %d, want 1", count)
	}
}
