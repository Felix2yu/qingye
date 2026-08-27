package services

import (
	"testing"
	"time"

	"qingye/server/models"
	"qingye/server/repositories"
)

func newTaskPlant(t *testing.T) (*models.Plant, *models.Task) {
	t.Helper()
	p := &models.Plant{Name: "测试植物"}
	if err := repositories.NewPlantRepo().Create(p); err != nil {
		t.Fatalf("create plant: %v", err)
	}
	tk := &models.Task{
		PlantID:      p.ID,
		Type:         TaskTypeWater,
		Title:        "浇水",
		IntervalDays: 7,
		BaseIntervalDays: 7,
		NextDue:      time.Now().Add(24 * time.Hour),
		Active:       true,
	}
	if err := repositories.NewTaskRepo().Create(tk); err != nil {
		t.Fatalf("create task: %v", err)
	}
	return p, tk
}

func TestTaskService_Update_Success(t *testing.T) {
	setupTestDB(t)
	_, tk := newTaskPlant(t)
	svc := NewTaskService()
	tk.Title = "浇透水"
	if err := svc.Update(tk); err != nil {
		t.Fatalf("update err: %v", err)
	}
	got, err := repositories.NewTaskRepo().Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "浇透水" {
		t.Errorf("title = %q, want 浇透水", got.Title)
	}
}

func TestTaskService_Update_NotFound(t *testing.T) {
	setupTestDB(t)
	svc := NewTaskService()
	if err := svc.Update(&models.Task{ID: 99999, Title: "x"}); err == nil {
		t.Error("update nonexistent should error")
	}
}

func TestTaskService_Update_EmptyID(t *testing.T) {
	setupTestDB(t)
	svc := NewTaskService()
	if err := svc.Update(&models.Task{Title: "x"}); err == nil {
		t.Error("empty id should error")
	}
}

func TestTaskService_Delete_Success(t *testing.T) {
	setupTestDB(t)
	_, tk := newTaskPlant(t)
	svc := NewTaskService()
	if err := svc.Delete(tk.ID); err != nil {
		t.Fatalf("delete err: %v", err)
	}
	if _, err := repositories.NewTaskRepo().Get(tk.ID); err == nil {
		t.Error("task should be gone after delete")
	}
}

func TestTaskService_Delete_NotFound(t *testing.T) {
	setupTestDB(t)
	svc := NewTaskService()
	if err := svc.Delete(99999); err == nil {
		t.Error("delete nonexistent should error")
	}
}

func TestTaskService_RecordManual_Success(t *testing.T) {
	setupTestDB(t)
	p, _ := newTaskPlant(t)
	svc := NewTaskService()
	log, err := svc.RecordManual(p.ID, "fertilize", "", "", time.Time{})
	if err != nil {
		t.Fatalf("record manual err: %v", err)
	}
	if log == nil || log.Type != "fertilize" {
		t.Errorf("unexpected care log: %+v", log)
	}
}

func TestTaskService_RecordManual_PlantNotFound(t *testing.T) {
	setupTestDB(t)
	svc := NewTaskService()
	if _, err := svc.RecordManual(99999, "water", "", "", time.Time{}); err == nil {
		t.Error("record manual for missing plant should error")
	}
}

func TestTaskService_History_Success(t *testing.T) {
	setupTestDB(t)
	_, tk := newTaskPlant(t)
	svc := NewTaskService()
	logs, err := svc.History(tk.ID)
	if err != nil {
		t.Fatalf("history err: %v", err)
	}
	if logs == nil {
		t.Error("history should return slice")
	}
}

func TestTaskService_Postpone_Success(t *testing.T) {
	setupTestDB(t)
	_, tk := newTaskPlant(t)
	svc := NewTaskService()
	before := tk.NextDue
	got, err := svc.Postpone(tk.ID, 3, "出差")
	if err != nil {
		t.Fatalf("postpone err: %v", err)
	}
	if !got.NextDue.After(before) {
		t.Errorf("next_due should move later: before=%v after=%v", before, got.NextDue)
	}
}

func TestTaskService_Done_Success(t *testing.T) {
	setupTestDB(t)
	_, tk := newTaskPlant(t)
	svc := NewTaskService()
	got, err := svc.Done(tk.ID, "已完成")
	if err != nil {
		t.Fatalf("done err: %v", err)
	}
	if got.LastDoneAt == nil {
		t.Error("last_done_at should be set")
	}
}
