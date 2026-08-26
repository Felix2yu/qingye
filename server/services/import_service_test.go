package services

import "testing"

// ---- parseCSV ----

func TestParseCSV_正常(t *testing.T) {
	csv := "name,species,room\n绿萝,Epipremnum,客厅\n龟背竹,Monstera,卧室"
	records, err := parseCSV(csv)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("rows = %d, want 3", len(records))
	}
	if records[0][0] != "name" {
		t.Errorf("header[0] = %q", records[0][0])
	}
	if records[1][1] != "Epipremnum" {
		t.Errorf("row1[1] = %q", records[1][1])
	}
}

func TestParseCSV_空行(t *testing.T) {
	csv := "name\n"
	records, err := parseCSV(csv)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("rows = %d, want 1", len(records))
	}
}

func TestParseCSV_引号字段(t *testing.T) {
	csv := `"name","note"
绿萝,"有藤, 叶大"`
	records, err := parseCSV(csv)
	if err != nil {
		t.Fatal(err)
	}
	if records[1][1] != "有藤, 叶大" {
		t.Errorf("quoted field = %q", records[1][1])
	}
}

// ---- columnIndex ----

func TestColumnIndex_命中(t *testing.T) {
	header := []string{"Name", "Species", "Room", "Note"}
	idx := columnIndex(header, "name")
	if idx != 0 {
		t.Errorf("columnIndex(\"name\") = %d, want 0", idx)
	}
	idx = columnIndex(header, "ROOM", "room")
	if idx != 2 {
		t.Errorf("columnIndex(\"ROOM\") = %d, want 2", idx)
	}
}

func TestColumnIndex_未命中(t *testing.T) {
	header := []string{"Name", "Species"}
	idx := columnIndex(header, "missing")
	if idx != -1 {
		t.Errorf("columnIndex(\"missing\") = %d, want -1", idx)
	}
}

func TestColumnIndex_空表头(t *testing.T) {
	idx := columnIndex(nil, "name")
	if idx != -1 {
		t.Errorf("columnIndex(nil) = %d, want -1", idx)
	}
}

// ---- cell ----

func TestCell_正常(t *testing.T) {
	row := []string{"a", "b", "c"}
	if got := cell(row, 1); got != "b" {
		t.Errorf("cell(row, 1) = %q, want b", got)
	}
}

func TestCell_越界(t *testing.T) {
	row := []string{"a"}
	if got := cell(row, 5); got != "" {
		t.Errorf("cell(row, 5) = %q, want empty", got)
	}
	if got := cell(row, -1); got != "" {
		t.Errorf("cell(row, -1) = %q, want empty", got)
	}
}

func TestCell_带空格(t *testing.T) {
	row := []string{"  hello  "}
	if got := cell(row, 0); got != "hello" {
		t.Errorf("cell(row, 0) = %q, want hello", got)
	}
}

// ---- joinReason ----

func TestJoinReason(t *testing.T) {
	tests := []struct{ base, extra, want string }{
		{"", "问题A", "问题A"},
		{"问题A", "问题B", "问题A；问题B"},
	}
	for _, tt := range tests {
		if got := joinReason(tt.base, tt.extra); got != tt.want {
			t.Errorf("joinReason(%q, %q) = %q, want %q", tt.base, tt.extra, got, tt.want)
		}
	}
}

// ---- toSet ----

func TestToSet(t *testing.T) {
	if got := toSet(nil); got != nil {
		t.Errorf("toSet(nil) = %v, want nil", got)
	}
	s := toSet([]int{1, 3, 5})
	if len(s) != 3 {
		t.Fatalf("toSet len = %d, want 3", len(s))
	}
	if !s[3] {
		t.Error("toSet: 3 not found")
	}
	if s[2] {
		t.Error("toSet: 2 should not exist")
	}
}

// ---- optStr ----

func TestOptStr(t *testing.T) {
	if got := optStr("hello"); got != "hello" {
		t.Errorf("optStr(\"hello\") = %q", got)
	}
	if got := optStr(42); got != "" {
		t.Errorf("optStr(42) = %q, want empty", got)
	}
	if got := optStr(nil); got != "" {
		t.Errorf("optStr(nil) = %q, want empty", got)
	}
}

// ---- normalizeTaskType ----

func TestNormalizeTaskType(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"water", TaskTypeWater, true},
		{"浇水", TaskTypeWater, true},
		{"fertilize", TaskTypeFertilize, true},
		{"施肥", TaskTypeFertilize, true},
		{"repot", TaskTypeRepot, true},
		{"换盆", TaskTypeRepot, true},
		{"unknown", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := normalizeTaskType(tt.input)
			if ok != tt.ok || got != tt.want {
				t.Errorf("normalizeTaskType(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// ---- PreviewPlants 纯解析 ----

func TestPreviewPlants_正常(t *testing.T) {
	svc := &ImportService{}
	csv := "name,species,room,note\n绿萝,Epipremnum,客厅,测试\n龟背竹,Monstera,,"
	preview, err := svc.PreviewPlants(csv)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Valid != 2 {
		t.Errorf("Valid = %d, want 2", preview.Valid)
	}
	if preview.Invalid != 0 {
		t.Errorf("Invalid = %d, want 0", preview.Invalid)
	}
}

func TestPreviewPlants_缺少名称(t *testing.T) {
	svc := &ImportService{}
	csv := "name,species\n,Epipremnum"
	preview, err := svc.PreviewPlants(csv)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Invalid != 1 {
		t.Errorf("Invalid = %d, want 1", preview.Invalid)
	}
}

func TestPreviewPlants_空文件(t *testing.T) {
	svc := &ImportService{}
	_, err := svc.PreviewPlants("name\n")
	if err == nil {
		t.Error("应返回错误")
	}
}
