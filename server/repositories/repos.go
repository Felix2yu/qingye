package repositories

import (
	"time"

	"qingye/server/models"

	"gorm.io/gorm"
)

// DB 全局数据库句柄（由 main 初始化）
var DB *gorm.DB

// SetDB 注入数据库连接
func SetDB(db *gorm.DB) { DB = db }

// ---------- Plant ----------

type PlantRepo struct{ db *gorm.DB }

func NewPlantRepo() *PlantRepo { return &PlantRepo{db: DB} }

func (r *PlantRepo) Create(p *models.Plant) error { return r.db.Create(p).Error }

func (r *PlantRepo) Update(p *models.Plant) error {
	return r.db.Model(p).Updates(map[string]any{
		"name":          p.Name,
		"species":       p.Species,
		"photo":         p.Photo,
		"room_id":       p.RoomID,
		"note":          p.Note,
		"location":      p.Location,
		"acquired_date": p.AcquiredDate,
		"light_req":     p.LightReq,
		"attributes":    p.Attributes,
	}).Error
}

func (r *PlantRepo) Delete(id uint) error { return r.db.Delete(&models.Plant{}, id).Error }

func (r *PlantRepo) Get(id uint) (*models.Plant, error) {
	var p models.Plant
	err := r.db.Preload("Room").First(&p, id).Error
	return &p, err
}

// List 按房间过滤（roomID=0 表示全部）
func (r *PlantRepo) List(roomID uint) ([]models.Plant, error) {
	var list []models.Plant
	q := r.db.Preload("Room").Order("created_at DESC")
	if roomID > 0 {
		q = q.Where("room_id = ?", roomID)
	}
	err := q.Find(&list).Error
	return list, err
}

// ---------- Room ----------

type RoomRepo struct{ db *gorm.DB }

func NewRoomRepo() *RoomRepo { return &RoomRepo{db: DB} }

func (r *RoomRepo) Create(room *models.Room) error { return r.db.Create(room).Error }

func (r *RoomRepo) Update(room *models.Room) error {
	return r.db.Model(room).Updates(map[string]any{"name": room.Name, "sort": room.Sort, "is_outdoor": room.IsOutdoor}).Error
}

func (r *RoomRepo) Delete(id uint) error { return r.db.Delete(&models.Room{}, id).Error }

func (r *RoomRepo) List() ([]models.Room, error) {
	var list []models.Room
	err := r.db.Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

// ---------- Task ----------

type TaskRepo struct{ db *gorm.DB }

func NewTaskRepo() *TaskRepo { return &TaskRepo{db: DB} }

func (r *TaskRepo) Create(t *models.Task) error { return r.db.Create(t).Error }

func (r *TaskRepo) Update(t *models.Task) error {
	return r.db.Model(t).Updates(map[string]any{
		"plant_id":           t.PlantID,
		"type":               t.Type,
		"title":              t.Title,
		"interval_days":      t.IntervalDays,
		"base_interval_days": t.BaseIntervalDays,
		"last_done_at":       t.LastDoneAt,
		"next_due":           t.NextDue,
		"active":             t.Active,
	}).Error
}

// ListActiveByType 所有 active 任务（可选按类型过滤，type="" 表示全部）
func (r *TaskRepo) ListActiveByType(taskType string) ([]models.Task, error) {
	q := r.db.Preload("Plant.Room").Where("active = ?", true)
	if taskType != "" {
		q = q.Where("type = ?", taskType)
	}
	var list []models.Task
	err := q.Find(&list).Error
	return list, err
}

// ListActiveByPlantType 某植物的 active 任务（按类型过滤）
func (r *TaskRepo) ListActiveByPlantType(plantID uint, taskType string) ([]models.Task, error) {
	var list []models.Task
	err := r.db.Where("active = ? AND plant_id = ? AND type = ?", true, plantID, taskType).Find(&list).Error
	return list, err
}

// SetInterval 更新任务周期（base 为原始周期，interval 为当前生效周期）
func (r *TaskRepo) SetInterval(id uint, base, interval int) error {
	return r.db.Model(&models.Task{}).Where("id = ?", id).
		Updates(map[string]any{"base_interval_days": base, "interval_days": interval}).Error
}

// SetNextDue 更新下次到期时间
func (r *TaskRepo) SetNextDue(id uint, due time.Time) error {
	return r.db.Model(&models.Task{}).Where("id = ?", id).Update("next_due", due).Error
}

func (r *TaskRepo) Delete(id uint) error { return r.db.Delete(&models.Task{}, id).Error }

func (r *TaskRepo) Get(id uint) (*models.Task, error) {
	var t models.Task
	err := r.db.Preload("Plant").First(&t, id).Error
	return &t, err
}

func (r *TaskRepo) GetWithPlant(id uint) (*models.Task, error) {
	var t models.Task
	err := r.db.Preload("Plant.Room").First(&t, id).Error
	return &t, err
}

// List 任务列表：type=空表示全部；includeDone=false 时仅返回 active 任务
func (r *TaskRepo) List(taskType string, includeDone bool, plantID uint) ([]models.Task, error) {
	var list []models.Task
	q := r.db.Preload("Plant.Room").Order("next_due ASC")
	if taskType != "" {
		q = q.Where("type = ?", taskType)
	}
	if !includeDone {
		q = q.Where("active = ?", true)
	}
	if plantID > 0 {
		q = q.Where("plant_id = ?", plantID)
	}
	err := q.Find(&list).Error
	return list, err
}

// DueBefore 查询 next_due 在 deadline 之前且 active 的任务
func (r *TaskRepo) DueBefore(deadline interface{}, plantID uint) ([]models.Task, error) {
	var list []models.Task
	q := r.db.Preload("Plant.Room").
		Where("active = ? AND next_due <= ?", true, deadline).
		Order("next_due ASC")
	if plantID > 0 {
		q = q.Where("plant_id = ?", plantID)
	}
	err := q.Find(&list).Error
	return list, err
}

// DueBetween 查询 next_due 在 [start, end] 内且 active 的任务
func (r *TaskRepo) DueBetween(start, end interface{}) ([]models.Task, error) {
	var list []models.Task
	err := r.db.Preload("Plant.Room").
		Where("active = ? AND next_due BETWEEN ? AND ?", true, start, end).
		Order("next_due ASC").
		Find(&list).Error
	return list, err
}

// ---------- TaskLog ----------

type TaskLogRepo struct{ db *gorm.DB }

func NewTaskLogRepo() *TaskLogRepo { return &TaskLogRepo{db: DB} }

func (r *TaskLogRepo) Create(l *models.TaskLog) error { return r.db.Create(l).Error }

func (r *TaskLogRepo) ListByTask(taskID uint) ([]models.TaskLog, error) {
	var list []models.TaskLog
	err := r.db.Where("task_id = ?", taskID).Order("at DESC").Find(&list).Error
	return list, err
}

func (r *TaskLogRepo) DeleteByTask(taskID uint) error {
	return r.db.Where("task_id = ?", taskID).Delete(&models.TaskLog{}).Error
}

// ---------- PhotoDiary ----------

type DiaryRepo struct{ db *gorm.DB }

func NewDiaryRepo() *DiaryRepo { return &DiaryRepo{db: DB} }

func (r *DiaryRepo) Create(d *models.PhotoDiary) error { return r.db.Create(d).Error }

func (r *DiaryRepo) Delete(id uint) error { return r.db.Delete(&models.PhotoDiary{}, id).Error }

// Page 分页时间线（plantID=0 表示全部），按 taken_at 倒序
func (r *DiaryRepo) Page(plantID uint, offset, limit int) ([]models.PhotoDiary, error) {
	var list []models.PhotoDiary
	q := r.db.Preload("Plant").Order("taken_at DESC, id DESC").Offset(offset).Limit(limit)
	if plantID > 0 {
		q = q.Where("plant_id = ?", plantID)
	}
	err := q.Find(&list).Error
	return list, err
}

// ---------- PlantLibrary ----------

type LibraryRepo struct{ db *gorm.DB }

func NewLibraryRepo() *LibraryRepo { return &LibraryRepo{db: DB} }

func (r *LibraryRepo) Search(keyword string) ([]models.PlantLibrary, error) {
	var list []models.PlantLibrary
	q := r.db.Order("name ASC")
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR alias LIKE ?", like, like)
	}
	err := q.Find(&list).Error
	return list, err
}

// GetByPID 按 Plantbook 学名唯一键查询（已同步的本地条目）
func (r *LibraryRepo) GetByPID(pid string) (*models.PlantLibrary, error) {
	var lib models.PlantLibrary
	err := r.db.Where("pid = ?", pid).First(&lib).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lib, nil
}

// ExistingMetrics 本地已同步条目：pid → 是否已含结构化指标。
// 批量同步据此跳过「已完整」的条目；缺指标的老条目会被重新拉取补齐。
func (r *LibraryRepo) ExistingMetrics() (map[string]bool, error) {
	type row struct {
		PID     string
		Metrics string
	}
	var rows []row
	err := r.db.Model(&models.PlantLibrary{}).
		Select("pid", "metrics").Where("pid <> ''").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(rows))
	for _, p := range rows {
		set[p.PID] = p.Metrics != ""
	}
	return set, nil
}

// UpsertByPID 以 pid 为唯一键写入/更新资料库条目（在线同步后沉淀为本地缓存）
func (r *LibraryRepo) UpsertByPID(lib *models.PlantLibrary) error {
	var existing models.PlantLibrary
	err := r.db.Where("pid = ?", lib.PID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(lib).Error
	}
	if err != nil {
		return err
	}
	lib.ID = existing.ID
	return r.db.Model(&existing).Updates(map[string]any{
		"name":         lib.Name,
		"display_pid":  lib.DisplayPID,
		"alias":        lib.Alias,
		"category":     lib.Category,
		"origin":       lib.Origin,
		"common_names": lib.CommonNames,
		"guide":        lib.Guide,
		"metrics":      lib.Metrics,
		"image":        lib.Image,
		"link":         lib.Link,
		"synced_at":    lib.SyncedAt,
	}).Error
}

// ---------- UserSetting ----------

type SettingRepo struct{ db *gorm.DB }

func NewSettingRepo() *SettingRepo { return &SettingRepo{db: DB} }

// Get 返回第一条设置；不存在时返回零值
func (r *SettingRepo) Get() (*models.UserSetting, error) {
	var s models.UserSetting
	err := r.db.First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return &s, nil
	}
	return &s, err
}

func (r *SettingRepo) Save(s *models.UserSetting) error {
	if s.ID == 0 {
		return r.db.Create(s).Error
	}
	return r.db.Save(s).Error
}

// ---------- CareLog ----------

type CareLogRepo struct{ db *gorm.DB }

func NewCareLogRepo() *CareLogRepo { return &CareLogRepo{db: DB} }

func (r *CareLogRepo) Create(l *models.CareLog) error { return r.db.Create(l).Error }

// ListByPlant 某植物的养护时间线（按时间倒序）
func (r *CareLogRepo) ListByPlant(plantID uint) ([]models.CareLog, error) {
	var list []models.CareLog
	err := r.db.Preload("Plant").Where("plant_id = ?", plantID).Order("at DESC, id DESC").Find(&list).Error
	return list, err
}

// List 所有养护时间线（按时间倒序），limit 限制条数
func (r *CareLogRepo) List(limit int) ([]models.CareLog, error) {
	var list []models.CareLog
	q := r.db.Preload("Plant").Order("at DESC, id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&list).Error
	return list, err
}

// ---------- WeatherLog ----------

type WeatherLogRepo struct{ db *gorm.DB }

func NewWeatherLogRepo() *WeatherLogRepo { return &WeatherLogRepo{db: DB} }

func (r *WeatherLogRepo) Create(l *models.WeatherLog) error { return r.db.Create(l).Error }

// List 按时间倒序，limit 限制条数
func (r *WeatherLogRepo) List(limit int) ([]models.WeatherLog, error) {
	var list []models.WeatherLog
	q := r.db.Order("at DESC, id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&list).Error
	return list, err
}
