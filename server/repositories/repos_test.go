package repositories

import (
	"testing"
	"time"

	"qingye/server/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setup(t *testing.T) *gorm.DB {
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
	SetDB(db)
	return db
}

// ---- Room ----

func TestRoomRepo_Create_List(t *testing.T) {
	setup(t)
	repo := NewRoomRepo()

	repo.Create(&models.Room{Name: "客厅", Sort: 1})
	repo.Create(&models.Room{Name: "卧室", Sort: 0})

	list, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("rooms = %d, want 2", len(list))
	}
	if list[0].Name != "卧室" {
		t.Errorf("first = %q, want 卧室", list[0].Name)
	}
}

func TestRoomRepo_Update_Delete(t *testing.T) {
	setup(t)
	repo := NewRoomRepo()

	r := &models.Room{Name: "客厅"}
	repo.Create(r)
	r.Name = "新客厅"
	repo.Update(r)
	list, _ := repo.List()
	if list[0].Name != "新客厅" {
		t.Errorf("after update: %q", list[0].Name)
	}

	repo.Delete(r.ID)
	list, _ = repo.List()
	if len(list) != 0 {
		t.Error("should be empty after delete")
	}
}

// ---- Plant ----

func TestPlantRepo_Create_Get_List(t *testing.T) {
	setup(t)
	roomRepo := NewRoomRepo()
	plantRepo := NewPlantRepo()

	room := &models.Room{Name: "客厅"}
	roomRepo.Create(room)

	p := &models.Plant{Name: "绿萝", RoomID: room.ID}
	if err := plantRepo.Create(p); err != nil {
		t.Fatal(err)
	}
	if p.ID == 0 {
		t.Fatal("ID should be populated")
	}

	got, err := plantRepo.Get(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "绿萝" || got.Room.Name != "客厅" {
		t.Errorf("Get: name=%q room=%q", got.Name, got.Room.Name)
	}

	list, _ := plantRepo.List(0)
	if len(list) != 1 {
		t.Fatalf("List(0) = %d, want 1", len(list))
	}
}

func TestPlantRepo_ListByRoom(t *testing.T) {
	setup(t)
	roomRepo := NewRoomRepo()
	plantRepo := NewPlantRepo()

	r1 := &models.Room{Name: "客厅"}
	r2 := &models.Room{Name: "卧室"}
	roomRepo.Create(r1)
	roomRepo.Create(r2)

	plantRepo.Create(&models.Plant{Name: "A", RoomID: r1.ID})
	plantRepo.Create(&models.Plant{Name: "B", RoomID: r1.ID})
	plantRepo.Create(&models.Plant{Name: "C", RoomID: r2.ID})

	list, _ := plantRepo.List(r1.ID)
	if len(list) != 2 {
		t.Fatalf("List(r1) = %d, want 2", len(list))
	}
}

func TestPlantRepo_Update_Delete(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()

	p := &models.Plant{Name: "A"}
	plantRepo.Create(p)

	p.Name = "B"
	p.Species = "Rosa"
	plantRepo.Update(p)
	got, _ := plantRepo.Get(p.ID)
	if got.Name != "B" || got.Species != "Rosa" {
		t.Errorf("update: name=%q species=%q", got.Name, got.Species)
	}

	plantRepo.Delete(p.ID)
	_, err := plantRepo.Get(p.ID)
	if err == nil {
		t.Error("should error after delete")
	}
}

// ---- Task ----

func TestTaskRepo_CRUD(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	task := &models.Task{
		PlantID:          p.ID,
		Type:             "water",
		Title:            "浇水",
		IntervalDays:     7,
		BaseIntervalDays: 7,
		NextDue:          time.Now().AddDate(0, 0, 7),
		Active:           true,
	}
	if err := taskRepo.Create(task); err != nil {
		t.Fatal(err)
	}
	if task.ID == 0 {
		t.Fatal("ID should be populated")
	}

	got, _ := taskRepo.Get(task.ID)
	if got.Type != "water" || got.Title != "浇水" {
		t.Errorf("Get: type=%q title=%q", got.Type, got.Title)
	}

	list, _ := taskRepo.List("", false, 0)
	if len(list) != 1 {
		t.Fatalf("List = %d", len(list))
	}
}

func TestTaskRepo_ListActiveByType(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	taskRepo.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: time.Now(), Active: true})
	taskRepo.Create(&models.Task{PlantID: p.ID, Type: "fertilize", IntervalDays: 30, NextDue: time.Now(), Active: true})

	water, _ := taskRepo.ListActiveByType("water")
	if len(water) != 1 {
		t.Fatalf("water = %d", len(water))
	}
	all, _ := taskRepo.ListActiveByType("")
	if len(all) != 2 {
		t.Fatalf("all = %d", len(all))
	}
}

func TestTaskRepo_DueBefore(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	soon := time.Now().Add(2 * 24 * time.Hour)
	later := time.Now().Add(30 * 24 * time.Hour)
	taskRepo.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: soon, Active: true})
	taskRepo.Create(&models.Task{PlantID: p.ID, Type: "fertilize", IntervalDays: 30, NextDue: later, Active: true})

	list, _ := taskRepo.DueBefore(time.Now().Add(5*24*time.Hour), 0)
	if len(list) != 1 {
		t.Fatalf("DueBefore = %d", len(list))
	}
}

func TestTaskRepo_SetInterval_SetNextDue(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	task := &models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, BaseIntervalDays: 7, NextDue: time.Now(), Active: true}
	taskRepo.Create(task)

	taskRepo.SetInterval(task.ID, 7, 10)
	got, _ := taskRepo.Get(task.ID)
	if got.IntervalDays != 10 || got.BaseIntervalDays != 7 {
		t.Errorf("interval=%d base=%d", got.IntervalDays, got.BaseIntervalDays)
	}

	newDue := time.Now().AddDate(0, 0, 10)
	taskRepo.SetNextDue(task.ID, newDue)
	got, _ = taskRepo.Get(task.ID)
	if got.NextDue.Day() != newDue.Day() {
		t.Errorf("due day = %d", got.NextDue.Day())
	}
}

// ---- CareLog ----

func TestCareLogRepo_Create_List(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	logRepo := NewCareLogRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	logRepo.Create(&models.CareLog{PlantID: p.ID, Type: "water", Title: "浇水", At: time.Now(), Source: "manual"})
	logRepo.Create(&models.CareLog{PlantID: p.ID, Type: "fertilize", Title: "施肥", At: time.Now(), Source: "task"})

	list, _ := logRepo.List(0)
	if len(list) != 2 {
		t.Fatalf("List = %d", len(list))
	}
	list, _ = logRepo.ListByPlant(p.ID)
	if len(list) != 2 {
		t.Fatalf("ListByPlant = %d", len(list))
	}
}

// ---- PhotoDiary ----

func TestDiaryRepo_CRUD(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	diaryRepo := NewDiaryRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	diaryRepo.Create(&models.PhotoDiary{PlantID: p.ID, Image: "a.jpg", Caption: "A", TakenAt: time.Now()})
	diaryRepo.Create(&models.PhotoDiary{PlantID: p.ID, Image: "b.jpg", Caption: "B", TakenAt: time.Now()})

	page, _ := diaryRepo.Page(0, 0, 10)
	if len(page) != 2 {
		t.Fatalf("Page = %d", len(page))
	}
	diaryRepo.Delete(page[0].ID)
	page, _ = diaryRepo.Page(0, 0, 10)
	if len(page) != 1 {
		t.Fatalf("After delete: %d", len(page))
	}
}

// ---- PlantLibrary ----

func TestLibraryRepo_Upsert_GetByPID_Search(t *testing.T) {
	db := setup(t)
	repo := NewLibraryRepo()

	// 直接用 db.Create 测试基础功能
	lib := &models.PlantLibrary{PID: "monstera", Name: "龟背竹", Guide: "浇水：适中"}
	if err := db.Create(lib).Error; err != nil {
		t.Fatal(err)
	}
	if lib.ID == 0 {
		t.Fatal("db.Create should populate ID")
	}

	// 测试 GetByPID
	got, _ := repo.GetByPID("monstera")
	if got == nil || got.Name != "龟背竹" {
		t.Fatalf("GetByPID: got=%v", got)
	}

	// 测试 Search
	list, _ := repo.Search("龟背")
	if len(list) != 1 {
		t.Fatalf("Search = %d", len(list))
	}
}

func TestLibraryRepo_UpsertByPID(t *testing.T) {
	setup(t)
	repo := NewLibraryRepo()

	// 第一次 upsert = 插入
	lib := &models.PlantLibrary{PID: "a", Name: "A", Guide: "原始"}
	if err := repo.UpsertByPID(lib); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetByPID("a")
	if got == nil || got.Guide != "原始" {
		t.Fatalf("first upsert: got=%v", got)
	}

	// 第二次 upsert = 更新
	lib.Guide = "更新后"
	if err := repo.UpsertByPID(lib); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetByPID("a")
	if got.Guide != "更新后" {
		t.Errorf("second upsert: guide=%q", got.Guide)
	}
}

func TestLibraryRepo_ExistingMetrics(t *testing.T) {
	db := setup(t)
	repo := NewLibraryRepo()

	// 直接用 db.Create 确保存在
	db.Create(&models.PlantLibrary{PID: "a", Name: "A", Metrics: `{"minTemp":10}`})
	db.Create(&models.PlantLibrary{PID: "b", Name: "B"})

	set, err := repo.ExistingMetrics()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ExistingMetrics: %v", set)
	if !set["a"] {
		t.Error("a should have metrics")
	}
	if set["b"] {
		t.Error("b should not have metrics")
	}
}

// ---- UserSetting ----

func TestSettingRepo_Get_Save(t *testing.T) {
	setup(t)
	repo := NewSettingRepo()

	s, _ := repo.Get()
	if s.ID != 0 {
		t.Error("first Get should be zero ID")
	}

	s.Workdays = "1,2,3,4,5"
	repo.Save(s)

	got, _ := repo.Get()
	if got.Workdays != "1,2,3,4,5" {
		t.Errorf("Workdays = %q", got.Workdays)
	}
}

// ---- WeatherLog ----

func TestWeatherLogRepo_Create_List(t *testing.T) {
	setup(t)
	repo := NewWeatherLogRepo()

	repo.Create(&models.WeatherLog{At: time.Now(), Kind: "cold"})
	repo.Create(&models.WeatherLog{At: time.Now(), Kind: "rain"})

	list, _ := repo.List(0)
	if len(list) != 2 {
		t.Fatalf("List = %d", len(list))
	}
	list, _ = repo.List(1)
	if len(list) != 1 {
		t.Fatalf("List(1) = %d", len(list))
	}
}

// ---- TaskLog ----

func TestTaskLogRepo_Create_List_DeleteByTask(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()
	logRepo := NewTaskLogRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)
	task := &models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: time.Now(), Active: true}
	taskRepo.Create(task)

	logRepo.Create(&models.TaskLog{TaskID: task.ID, Action: "done", At: time.Now()})
	logRepo.Create(&models.TaskLog{TaskID: task.ID, Action: "postpone", At: time.Now()})

	list, _ := logRepo.ListByTask(task.ID)
	if len(list) != 2 {
		t.Fatalf("ListByTask = %d", len(list))
	}

	logRepo.DeleteByTask(task.ID)
	list, _ = logRepo.ListByTask(task.ID)
	if len(list) != 0 {
		t.Fatalf("After delete: %d", len(list))
	}
}

// ---- TaskRepo 补充 ----

func TestTaskRepo_Update(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	task := &models.Task{PlantID: p.ID, Type: "water", Title: "浇水", IntervalDays: 7, NextDue: time.Now(), Active: true}
	taskRepo.Create(task)

	task.Title = "新标题"
	task.IntervalDays = 14
	task.Active = false
	if err := taskRepo.Update(task); err != nil {
		t.Fatal(err)
	}

	got, _ := taskRepo.Get(task.ID)
	if got.Title != "新标题" || got.IntervalDays != 14 || got.Active {
		t.Errorf("after update: title=%q interval=%d active=%v", got.Title, got.IntervalDays, got.Active)
	}
}

func TestTaskRepo_GetWithPlant(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	task := &models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: time.Now(), Active: true}
	taskRepo.Create(task)

	got, err := taskRepo.GetWithPlant(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Plant.Name != "X" {
		t.Errorf("Plant.Name = %q", got.Plant.Name)
	}
}

func TestTaskRepo_ListActiveByPlantType(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	taskRepo.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: time.Now(), Active: true})
	taskRepo.Create(&models.Task{PlantID: p.ID, Type: "fertilize", IntervalDays: 30, NextDue: time.Now(), Active: true})

	list, _ := taskRepo.ListActiveByPlantType(p.ID, "water")
	if len(list) != 1 {
		t.Fatalf("ListActiveByPlantType(water) = %d, want 1", len(list))
	}
}

func TestTaskRepo_DueBetween(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)

	soon := time.Now().Add(2 * 24 * time.Hour)
	later := time.Now().Add(30 * 24 * time.Hour)
	taskRepo.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: soon, Active: true})
	taskRepo.Create(&models.Task{PlantID: p.ID, Type: "fertilize", IntervalDays: 30, NextDue: later, Active: true})

	start := time.Now().Add(1 * 24 * time.Hour)
	end := time.Now().Add(5 * 24 * time.Hour)
	list, _ := taskRepo.DueBetween(start, end)
	if len(list) != 1 {
		t.Fatalf("DueBetween = %d, want 1", len(list))
	}
}

func TestTaskRepo_Delete(t *testing.T) {
	setup(t)
	plantRepo := NewPlantRepo()
	taskRepo := NewTaskRepo()

	p := &models.Plant{Name: "X"}
	plantRepo.Create(p)
	task := &models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7, NextDue: time.Now(), Active: true}
	taskRepo.Create(task)

	if err := taskRepo.Delete(task.ID); err != nil {
		t.Fatal(err)
	}
	_, err := taskRepo.Get(task.ID)
	if err == nil {
		t.Error("should error after delete")
	}
}
