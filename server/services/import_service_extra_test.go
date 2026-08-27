package services

import (
	"testing"
	"time"

	"qingye/server/models"
)

func TestImportService_ConfirmPlants(t *testing.T) {
	db := setupTestDB(t)
	svc := NewImportService()

	csv := "name,species,room\n玫瑰,蔷薇,花园\n薄荷,唇形科,阳台"
	preview, err := svc.PreviewPlants(csv)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Valid != 2 {
		t.Fatalf("valid = %d, want 2", preview.Valid)
	}

	res, err := svc.ConfirmPlants(preview, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Created != 2 {
		t.Errorf("created = %d, want 2", res.Created)
	}

	var plantCount, roomCount int64
	db.Model(&models.Plant{}).Count(&plantCount)
	db.Model(&models.Room{}).Count(&roomCount)
	if plantCount != 2 {
		t.Errorf("plants = %d, want 2", plantCount)
	}
	if roomCount != 2 { // 花园、阳台两个新房间
		t.Errorf("rooms = %d, want 2", roomCount)
	}
}

func TestImportService_ConfirmPlants_AcceptedSubset(t *testing.T) {
	db := setupTestDB(t)
	svc := NewImportService()
	csv := "name,species\n玫瑰,蔷薇\n百合,百合科"
	preview, _ := svc.PreviewPlants(csv)
	// 仅接受第 1 行（line=1）
	res, err := svc.ConfirmPlants(preview, []int{1})
	if err != nil {
		t.Fatal(err)
	}
	if res.Created != 1 {
		t.Errorf("created = %d, want 1", res.Created)
	}
	var c int64
	db.Model(&models.Plant{}).Count(&c)
	if c != 1 {
		t.Errorf("plants = %d, want 1", c)
	}
}

func TestImportService_ConfirmTasks(t *testing.T) {
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
		t.Fatalf("valid = %d, want 1", preview.Valid)
	}
	res, err := svc.ConfirmTasks(preview, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Created != 1 {
		t.Errorf("created tasks = %d, want 1", res.Created)
	}
	var c int64
	db.Model(&models.Task{}).Count(&c)
	if c != 1 {
		t.Errorf("tasks = %d, want 1", c)
	}
}

func TestImportService_PreviewTemplate(t *testing.T) {
	db := setupTestDB(t)
	svc := NewImportService()
	src := &models.Plant{Name: "源"}
	db.Create(src)
	db.Create(&models.Task{PlantID: src.ID, Type: "water", IntervalDays: 7, NextDue: time.Now(), Active: true})
	tgt := &models.Plant{Name: "目标"}
	db.Create(tgt)
	preview, err := svc.PreviewTemplate(src.ID, []uint{tgt.ID})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Valid != 1 {
		t.Fatalf("valid = %d, want 1", preview.Valid)
	}
	if preview.Rows[0].Data.(map[string]any)["taskCount"].(int) != 1 {
		t.Errorf("taskCount not 1")
	}
}

func TestImportService_ConfirmTemplate(t *testing.T) {
	db := setupTestDB(t)
	svc := NewImportService()
	src := &models.Plant{Name: "源"}
	db.Create(src)
	db.Create(&models.Task{PlantID: src.ID, Type: "water", IntervalDays: 7, NextDue: time.Now(), Active: true})
	db.Create(&models.Task{PlantID: src.ID, Type: "fertilize", IntervalDays: 30, NextDue: time.Now(), Active: true})
	tgt := &models.Plant{Name: "目标"}
	db.Create(tgt)
	res, err := svc.ConfirmTemplate(src.ID, []uint{tgt.ID})
	if err != nil {
		t.Fatal(err)
	}
	if res.Created != 2 {
		t.Errorf("created = %d, want 2", res.Created)
	}
}

func TestImportService_PreviewTemplate_NoTargets(t *testing.T) {
	setupTestDB(t)
	svc := NewImportService()
	if _, err := svc.PreviewTemplate(1, nil); err == nil {
		t.Error("empty targets should error")
	}
}

func TestImportService_PreviewTemplate_MissingTarget(t *testing.T) {
	db := setupTestDB(t)
	svc := NewImportService()
	src := &models.Plant{Name: "源"}
	db.Create(src)
	preview, err := svc.PreviewTemplate(src.ID, []uint{99999})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Invalid != 1 {
		t.Errorf("invalid = %d, want 1", preview.Invalid)
	}
}

func TestNormalizeTaskType_Table(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"water", TaskTypeWater, true},
		{"浇水", TaskTypeWater, true},
		{"fertilize", TaskTypeFertilize, true},
		{"施肥", TaskTypeFertilize, true},
		{"mist", TaskTypeMist, true},
		{"喷雾", TaskTypeMist, true},
		{"repot", TaskTypeRepot, true},
		{"换盆", TaskTypeRepot, true},
		{"prune", TaskTypePrune, true},
		{"修剪", TaskTypePrune, true},
		{"clean", TaskTypeClean, true},
		{"清理", TaskTypeClean, true},
		{"pesticide", TaskTypePesticide, true},
		{"除虫", TaskTypePesticide, true},
		{"other", TaskTypeOther, true},
		{"其他", TaskTypeOther, true},
		{"unknown", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := normalizeTaskType(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Errorf("normalizeTaskType(%q) = (%q,%v), want (%q,%v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestImportHelpers(t *testing.T) {
	// parseCSV 正常
	recs, err := parseCSV("a,b\n1,2\n3,4")
	if err != nil || len(recs) != 3 {
		t.Fatalf("parseCSV: %v len=%d", err, len(recs))
	}
	// parseCSV 错误（未闭合引号）
	if _, err := parseCSV("a,\"unclosed"); err == nil {
		t.Error("parseCSV should error on unclosed quote")
	}
	// columnIndex
	if columnIndex([]string{"Name", "Type"}, "name", "名称") != 0 {
		t.Error("columnIndex #0")
	}
	if columnIndex([]string{"Name", "Type"}, "x") != -1 {
		t.Error("columnIndex should be -1")
	}
	// cell
	if cell([]string{"a", "b"}, 1) != "b" {
		t.Error("cell #1")
	}
	if cell([]string{"a"}, 5) != "" {
		t.Error("cell out of range")
	}
	// joinReason
	if joinReason("", "x") != "x" {
		t.Error("joinReason empty base")
	}
	if joinReason("a", "b") != "a；b" {
		t.Error("joinReason join")
	}
	// toSet
	if toSet(nil) != nil {
		t.Error("toSet(nil)")
	}
	s := toSet([]int{1, 2})
	if !s[1] || !s[2] {
		t.Error("toSet")
	}
	// optStr
	if optStr("x") != "x" {
		t.Error("optStr string")
	}
	if optStr(123) != "" {
		t.Error("optStr non-string")
	}
}
