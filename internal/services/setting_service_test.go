package services

import (
	"testing"
	"time"
)

// ---- ParseWorkdays ----

func TestParseWorkdays_正常(t *testing.T) {
	set := ParseWorkdays("1,2,3,4,5")
	if len(set) != 5 {
		t.Fatalf("len = %d, want 5", len(set))
	}
	for _, d := range []int{1, 2, 3, 4, 5} {
		if !set[d] {
			t.Errorf("day %d not found", d)
		}
	}
}

func TestParseWorkdays_空串(t *testing.T) {
	set := ParseWorkdays("")
	if len(set) != 0 {
		t.Errorf("empty = %v, want empty set", set)
	}
}

func TestParseWorkdays_带空格(t *testing.T) {
	set := ParseWorkdays(" 1 , 3 , 5 ")
	if len(set) != 3 {
		t.Fatalf("len = %d, want 3", len(set))
	}
}

func TestParseWorkdays_忽略无效值(t *testing.T) {
	set := ParseWorkdays("1,abc,9,0,7")
	if len(set) != 2 { // 1 and 7
		t.Fatalf("len = %d, want 2", len(set))
	}
	if !set[1] || !set[7] {
		t.Error("should contain 1 and 7")
	}
}

// ---- FormatWorkdays ----

func TestFormatWorkdays_正常(t *testing.T) {
	set := WorkdaySet{5: true, 1: true, 3: true}
	got := FormatWorkdays(set)
	if got != "1,3,5" {
		t.Errorf("FormatWorkdays = %q, want \"1,3,5\"", got)
	}
}

func TestFormatWorkdays_空集(t *testing.T) {
	got := FormatWorkdays(WorkdaySet{})
	if got != "" {
		t.Errorf("FormatWorkdays(empty) = %q, want empty", got)
	}
}

// ---- WeekdayToInt ----

func TestWeekdayToInt(t *testing.T) {
	tests := []struct {
		d    time.Weekday
		want int
	}{
		{time.Monday, 1},
		{time.Tuesday, 2},
		{time.Wednesday, 3},
		{time.Thursday, 4},
		{time.Friday, 5},
		{time.Saturday, 6},
		{time.Sunday, 7},
	}
	for _, tt := range tests {
		t.Run(tt.d.String(), func(t *testing.T) {
			if got := WeekdayToInt(tt.d); got != tt.want {
				t.Errorf("WeekdayToInt(%s) = %d, want %d", tt.d, got, tt.want)
			}
		})
	}
}
