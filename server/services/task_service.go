package services

import (
	"errors"
	"time"

	"qingye/server/models"
	"qingye/server/repositories"

	"gorm.io/gorm"
)

// 任务类型
const (
	TaskTypeWater     = "water"
	TaskTypeFertilize = "fertilize"
	TaskTypeMist      = "mist"
	TaskTypeRepot     = "repot"
	TaskTypePrune     = "prune"
	TaskTypeClean     = "clean"
	TaskTypePesticide = "pesticide"
	TaskTypeOther     = "other"
)

// 任务动作
const (
	ActionDone     = "done"
	ActionPostpone = "postpone"
)

var taskTypeNames = map[string]string{
	TaskTypeWater:     "浇水",
	TaskTypeFertilize: "施肥",
	TaskTypeMist:      "喷雾",
	TaskTypeRepot:     "换盆",
	TaskTypePrune:     "修剪",
	TaskTypeClean:     "清理",
	TaskTypePesticide: "除虫",
	TaskTypeOther:     "其他",
}

// TaskTypeName 返回任务类型中文名
func TaskTypeName(t string) string {
	if n, ok := taskTypeNames[t]; ok {
		return n
	}
	return t
}

// TaskService 任务业务：今日计算、完成、推迟、历史
type TaskService struct {
	tasks   *repositories.TaskRepo
	logs    *repositories.TaskLogRepo
	careLogs *repositories.CareLogRepo
	plants  *repositories.PlantRepo
}

func NewTaskService() *TaskService {
	return &TaskService{
		tasks:   repositories.NewTaskRepo(),
		logs:    repositories.NewTaskLogRepo(),
		careLogs: repositories.NewCareLogRepo(),
		plants:  repositories.NewPlantRepo(),
	}
}

// List 任务列表：type=空表示全部；includeDone=false 仅 active
func (s *TaskService) List(taskType string, includeDone bool, plantID uint) ([]models.Task, error) {
	return s.tasks.List(taskType, includeDone, plantID)
}

// Create 创建任务；未指定 next_due 时从今天起按周期计算
func (s *TaskService) Create(t *models.Task) (*models.Task, error) {
	if t.PlantID == 0 {
		return nil, errors.New("植物不能为空")
	}
	if t.IntervalDays <= 0 {
		return nil, errors.New("任务周期必须大于 0 天")
	}
	if _, err := s.plants.Get(t.PlantID); err != nil {
		return nil, errors.New("植物不存在")
	}
	if t.Title == "" {
		t.Title = TaskTypeName(t.Type)
	}
	if t.BaseIntervalDays <= 0 {
		t.BaseIntervalDays = t.IntervalDays
	}
	now := time.Now()
	if t.NextDue.IsZero() {
		t.NextDue = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
			AddDate(0, 0, t.IntervalDays)
	}
	t.Active = true
	if err := s.tasks.Create(t); err != nil {
		return nil, err
	}
	return s.tasks.GetWithPlant(t.ID)
}

// Update 更新任务
func (s *TaskService) Update(t *models.Task) error {
	if t.ID == 0 {
		return errors.New("任务 ID 不能为空")
	}
	if _, err := s.tasks.Get(t.ID); err != nil {
		return err
	}
	return s.tasks.Update(t)
}

// Delete 删除任务及其日志
func (s *TaskService) Delete(id uint) error {
	if _, err := s.tasks.Get(id); err != nil {
		return err
	}
	if err := s.logs.DeleteByTask(id); err != nil {
		return err
	}
	return s.tasks.Delete(id)
}

// Today 今日任务：仅在工作日展示到期任务；休息日不展示（任务不过期，顺延显示）
func (s *TaskService) Today() ([]models.Task, error) {
	now := time.Now()
	loc := now.Location()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1).Add(-time.Nanosecond)

	isWork, err := NewSettingService().IsWorkday(now)
	if err != nil {
		return nil, err
	}
	if !isWork {
		return []models.Task{}, nil
	}
	return s.tasks.DueBefore(end, 0)
}

// Upcoming 未来 days 天内（不含今天）的到期任务
func (s *TaskService) Upcoming(days int) ([]models.Task, error) {
	if days <= 0 {
		days = 3
	}
	now := time.Now()
	loc := now.Location()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, days).Add(-time.Nanosecond)
	return s.tasks.DueBetween(start, end)
}

// Done 完成任务（事务）：更新 last_done_at / next_due 并写入日志
func (s *TaskService) Done(taskID uint, note string) (*models.Task, error) {
	t, err := s.tasks.GetWithPlant(taskID)
	if err != nil {
		return nil, err
	}
	if !t.Active {
		return nil, errors.New("任务已停用")
	}
	now := time.Now()
	nextDue := now.AddDate(0, 0, t.IntervalDays)

	err = repositories.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(t).Updates(map[string]any{
			"last_done_at": now,
			"next_due":     nextDue,
			"active":       true,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.TaskLog{
			TaskID: taskID,
			Action: ActionDone,
			At:     now,
			Note:   note,
		}).Error; err != nil {
			return err
		}
		// 写入统一养护事件表（时间线来源），与 task_logs 并存
		return tx.Create(&models.CareLog{
			PlantID: t.PlantID,
			Type:    t.Type,
			Title:   t.Title,
			At:      now,
			Note:    note,
			Source:  models.CareSourceTask,
			TaskID:  &t.ID,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	t.LastDoneAt = &now
	t.NextDue = nextDue
	return t, nil
}

// RecordManual 记录一次无任务来源的人工养护（写 care_logs，可选生成任务）
func (s *TaskService) RecordManual(plantID uint, careType, title, note string, at time.Time) (*models.CareLog, error) {
	if _, err := s.plants.Get(plantID); err != nil {
		return nil, errors.New("植物不存在")
	}
	if careType == "" {
		careType = "other"
	}
	if title == "" {
		title = TaskTypeName(careType)
	}
	if at.IsZero() {
		at = time.Now()
	}
	log := &models.CareLog{
		PlantID: plantID,
		Type:    careType,
		Title:   title,
		At:      at,
		Note:    note,
		Source:  models.CareSourceManual,
	}
	if err := s.careLogs.Create(log); err != nil {
		return nil, err
	}
	return log, nil
}

// Postpone 推迟任务（事务）：next_due 顺延 days 天并写入日志
// 已过期的任务从今天起算，未过期的从当前 next_due 起算
func (s *TaskService) Postpone(taskID uint, days int, note string) (*models.Task, error) {
	if days <= 0 {
		days = 1
	}
	t, err := s.tasks.GetWithPlant(taskID)
	if err != nil {
		return nil, err
	}
	if !t.Active {
		return nil, errors.New("任务已停用")
	}
	now := time.Now()
	loc := now.Location()
	base := t.NextDue
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if t.NextDue.Before(now) {
		base = today
	}
	newDue := base.AddDate(0, 0, days)

	err = repositories.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(t).Update("next_due", newDue).Error; err != nil {
			return err
		}
		return tx.Create(&models.TaskLog{
			TaskID: taskID,
			Action: ActionPostpone,
			At:     now,
			Note:   note,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	t.NextDue = newDue
	return t, nil
}

// History 任务历史日志
func (s *TaskService) History(taskID uint) ([]models.TaskLog, error) {
	if _, err := s.tasks.Get(taskID); err != nil {
		return nil, err
	}
	return s.logs.ListByTask(taskID)
}
