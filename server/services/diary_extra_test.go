package services

import (
	"testing"
	"time"

	"qingye/server/models"
)

func TestDiaryService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewDiaryService()
	p := &models.Plant{Name: "X"}
	db.Create(p)
	d, _ := svc.Create(&models.PhotoDiary{PlantID: p.ID, Image: "a.jpg"})
	if err := svc.Delete(d.ID); err != nil {
		t.Fatal(err)
	}
	var recheck models.PhotoDiary
	if err := db.Where("id = ?", d.ID).First(&recheck).Error; err == nil {
		t.Error("should error after delete")
	}
}

func TestDiaryService_Page_Branches(t *testing.T) {
	db := setupTestDB(t)
	svc := NewDiaryService()
	p1 := &models.Plant{Name: "A"}
	p2 := &models.Plant{Name: "B"}
	db.Create(p1)
	db.Create(p2)
	for i := 0; i < 5; i++ {
		db.Create(&models.PhotoDiary{PlantID: p1.ID, Image: "a.jpg", TakenAt: time.Now()})
	}
	for i := 0; i < 3; i++ {
		db.Create(&models.PhotoDiary{PlantID: p2.ID, Image: "a.jpg", TakenAt: time.Now()})
	}
	// 默认分页边界（page<=0 / pageSize<=0 → 默认 20）
	list, total, _ := svc.Page(0, 0, 0)
	if total != 8 {
		t.Errorf("total = %d, want 8", total)
	}
	if len(list) != 8 {
		t.Errorf("len = %d, want 8", len(list))
	}
	// 按植物过滤
	list2, total2, _ := svc.Page(p1.ID, 1, 100)
	if total2 != 5 {
		t.Errorf("total filtered = %d, want 5", total2)
	}
	if len(list2) != 5 {
		t.Errorf("len filtered = %d, want 5", len(list2))
	}
	// pageSize 上限 100
	list3, _, _ := svc.Page(0, 1, 200)
	if len(list3) > 100 {
		t.Errorf("pageSize capped at 100, got %d", len(list3))
	}
}
