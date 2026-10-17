package handlers

import (
	"testing"
	"time"
)

func TestToModel_无日期(t *testing.T) {
	b := &plantBody{
		Name:    "绿萝",
		Species: "Epipremnum",
		RoomID:  1,
	}
	p, err := b.toModel(42)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != 42 {
		t.Errorf("ID = %d, want 42", p.ID)
	}
	if p.Name != "绿萝" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.AcquiredDate != nil {
		t.Error("AcquiredDate should be nil")
	}
}

func TestToModel_日期YYYYMMDD(t *testing.T) {
	b := &plantBody{Name: "Test", AcquiredDate: "2025-03-15"}
	p, err := b.toModel(1)
	if err != nil {
		t.Fatal(err)
	}
	if p.AcquiredDate == nil {
		t.Fatal("AcquiredDate should not be nil")
	}
	if p.AcquiredDate.Day() != 15 {
		t.Errorf("day = %d, want 15", p.AcquiredDate.Day())
	}
}

func TestToModel_日期RFC3339(t *testing.T) {
	b := &plantBody{Name: "Test", AcquiredDate: "2025-03-15T10:30:00+08:00"}
	p, err := b.toModel(1)
	if err != nil {
		t.Fatal(err)
	}
	if p.AcquiredDate == nil {
		t.Fatal("AcquiredDate should not be nil")
	}
	if p.AcquiredDate.Hour() != 10 {
		t.Errorf("hour = %d, want 10", p.AcquiredDate.Hour())
	}
}

func TestToModel_无效日期(t *testing.T) {
	b := &plantBody{Name: "Test", AcquiredDate: "not-a-date"}
	_, err := b.toModel(1)
	if err == nil {
		t.Error("invalid date should return error")
	}
}

func TestToModel_字段映射(t *testing.T) {
	b := &plantBody{
		Name:       "Test",
		Species:    "Rosa",
		Photo:      "photo.jpg",
		RoomID:     3,
		Note:       "note",
		Location:   "阳台",
		LightReq:   "明亮",
		Attributes: "attr",
	}
	p, err := b.toModel(1)
	if err != nil {
		t.Fatal(err)
	}
	if p.Species != "Rosa" {
		t.Errorf("Species = %q", p.Species)
	}
	if p.Photo != "photo.jpg" {
		t.Errorf("Photo = %q", p.Photo)
	}
	if p.RoomID != 3 {
		t.Errorf("RoomID = %d", p.RoomID)
	}
	if p.Location != "阳台" {
		t.Errorf("Location = %q", p.Location)
	}
	if p.LightReq != "明亮" {
		t.Errorf("LightReq = %q", p.LightReq)
	}
	if p.Attributes != "attr" {
		t.Errorf("Attributes = %q", p.Attributes)
	}
}

func TestParseFlexTime_RFC3339_时区(t *testing.T) {
	s := "2025-03-15T10:30:00Z"
	got := parseFlexTime(s)
	if got.IsZero() {
		t.Fatal("RFC3339 UTC parse failed")
	}
	if got.Location() != time.UTC {
		t.Errorf("location = %v, want UTC", got.Location())
	}
}
