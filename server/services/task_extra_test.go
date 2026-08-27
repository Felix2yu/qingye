package services

import (
	"testing"

	"qingye/server/models"
	"qingye/server/repositories"
)

func TestTaskService_Create_植物不存在(t *testing.T) {
	setupTestDB(t)
	svc := NewTaskService()
	_, err := svc.Create(&models.Task{PlantID: 99999, Type: "water", IntervalDays: 7})
	if err == nil || err.Error() != "植物不存在" {
		t.Errorf("err = %v", err)
	}
}

func TestTaskService_Done_已停用(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()
	p := &models.Plant{Name: "X"}
	db.Create(p)
	task, _ := svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7})
	// 直接置为停用
	repositories.DB.Model(&models.Task{}).Where("id = ?", task.ID).Update("active", false)
	_, err := svc.Done(task.ID, "")
	if err == nil || err.Error() != "任务已停用" {
		t.Errorf("err = %v", err)
	}
}

func TestTaskService_Postpone_已停用(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTaskService()
	p := &models.Plant{Name: "X"}
	db.Create(p)
	task, _ := svc.Create(&models.Task{PlantID: p.ID, Type: "water", IntervalDays: 7})
	repositories.DB.Model(&models.Task{}).Where("id = ?", task.ID).Update("active", false)
	_, err := svc.Postpone(task.ID, 3, "")
	if err == nil || err.Error() != "任务已停用" {
		t.Errorf("err = %v", err)
	}
}
