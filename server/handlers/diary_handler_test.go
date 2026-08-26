package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseFlexTime_空串(t *testing.T) {
	if !parseFlexTime("").IsZero() {
		t.Error("empty string should return zero time")
	}
}

func TestParseFlexTime_RFC3339(t *testing.T) {
	s := "2025-03-15T10:30:00+08:00"
	got := parseFlexTime(s)
	if got.IsZero() {
		t.Fatal("RFC3339 parse failed")
	}
	if got.Year() != 2025 || got.Month() != 3 || got.Day() != 15 {
		t.Errorf("date = %v", got)
	}
}

func TestParseFlexTime_DateTime(t *testing.T) {
	s := "2025-03-15 10:30:00"
	got := parseFlexTime(s)
	if got.IsZero() {
		t.Fatal("DateTime parse failed")
	}
	if got.Hour() != 10 {
		t.Errorf("hour = %d, want 10", got.Hour())
	}
}

func TestParseFlexTime_DateOnly(t *testing.T) {
	s := "2025-03-15"
	got := parseFlexTime(s)
	if got.IsZero() {
		t.Fatal("DateOnly parse failed")
	}
	if got.Day() != 15 {
		t.Errorf("day = %d, want 15", got.Day())
	}
}

func TestParseFlexTime_无效格式(t *testing.T) {
	got := parseFlexTime("not-a-date")
	if !got.IsZero() {
		t.Error("invalid format should return zero time")
	}
}

func TestRandomString_长度(t *testing.T) {
	for _, n := range []int{0, 1, 10, 100} {
		s := randomString(n)
		if len(s) != n {
			t.Errorf("randomString(%d) len = %d", n, len(s))
		}
	}
}

func TestRandomString_字符范围(t *testing.T) {
	s := randomString(1000)
	for i, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			t.Errorf("char at %d = %c, out of range", i, c)
			break
		}
	}
}

func TestParseUintQuery(t *testing.T) {
	tests := []struct {
		query string
		want  uint
		err   bool
	}{
		{"42", 42, false},
		{"0", 0, false},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/?id="+tt.query, nil)

		got, err := parseUintQuery(c, "id")
		if (err != nil) != tt.err {
			t.Errorf("parseUintQuery(%q) err = %v, wantErr %v", tt.query, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseUintQuery(%q) = %d, want %d", tt.query, got, tt.want)
		}
	}
}
